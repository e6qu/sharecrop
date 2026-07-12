package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RateLimiter struct {
	db           Beginner
	prefix       string
	capacity     float64
	refillPerSec float64
	fullAfter    time.Duration
	now          func() time.Time
}

func NewRateLimiter(pool *pgxpool.Pool, prefix string, capacity int, refillPerSec float64) RateLimiter {
	return NewRateLimiterFromHandle(NewPGX(pool), prefix, capacity, refillPerSec)
}

func NewRateLimiterFromHandle(handle Beginner, prefix string, capacity int, refillPerSec float64) RateLimiter {
	return RateLimiter{
		db:           handle,
		prefix:       prefix,
		capacity:     float64(capacity),
		refillPerSec: refillPerSec,
		fullAfter:    time.Duration(float64(time.Second) * (float64(capacity) / refillPerSec)),
		now:          time.Now,
	}
}

func (limiter RateLimiter) Allow(key string) bool {
	ctx := context.Background()
	tx, err := limiter.db.Begin(ctx)
	if err != nil {
		panic(fmt.Sprintf("begin rate limit transaction failed: %v", err))
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		_ = rollbackErr
	}()

	now := limiter.now().UTC()
	bucketKey := limiter.bucketKey(key)

	// Ensure the row exists before locking it. "select ... for update" on a
	// not-yet-existing row has nothing to lock, so two concurrent first
	// touches of a brand-new key could both read "no bucket yet" and both
	// grant, over-admitting past capacity. Postgres serializes concurrent
	// inserts of the same conflicting key (a second inserter blocks until
	// the first commits), so this closes that race: only one caller's
	// insert actually creates the row, and every caller — including the
	// first — then locks and reads the same, single row below.
	if _, err := tx.Exec(ctx, `
		insert into rate_limit_buckets (key, tokens, updated_at)
		values ($1, $2, $3)
		on conflict (key) do nothing
	`, bucketKey, limiter.capacity, now); err != nil {
		panic(fmt.Sprintf("seed rate limit bucket failed: %v", err))
	}

	var tokens float64
	var updatedAt time.Time
	err = tx.QueryRow(ctx, `
		select tokens, updated_at
		from rate_limit_buckets
		where key = $1
		for update
	`, bucketKey).Scan(&tokens, &updatedAt)
	if err != nil {
		panic(fmt.Sprintf("read rate limit bucket failed: %v", err))
	}
	tokens += now.Sub(updatedAt).Seconds() * limiter.refillPerSec
	if tokens > limiter.capacity {
		tokens = limiter.capacity
	}

	if tokens < 1 {
		if err := limiter.storeBucket(ctx, tx, bucketKey, tokens, now); err != nil {
			panic(fmt.Sprintf("store rate limit bucket failed: %v", err))
		}
		if err := tx.Commit(ctx); err != nil {
			panic(fmt.Sprintf("commit rate limit transaction failed: %v", err))
		}
		return false
	}

	if err := limiter.storeBucket(ctx, tx, bucketKey, tokens-1, now); err != nil {
		panic(fmt.Sprintf("store rate limit bucket failed: %v", err))
	}
	if err := tx.Commit(ctx); err != nil {
		panic(fmt.Sprintf("commit rate limit transaction failed: %v", err))
	}
	return true
}

func (limiter RateLimiter) ActiveBuckets() int {
	ctx := context.Background()
	limiter.evictExpired(ctx)
	var count int
	err := limiter.db.QueryRow(ctx, `
		select count(*)
		from rate_limit_buckets
		where key like $1
	`, limiter.prefix+":%").Scan(&count)
	if err != nil {
		panic(fmt.Sprintf("count rate limit buckets failed: %v", err))
	}
	return count
}

func (limiter RateLimiter) StorageKind() string {
	return "postgres"
}

func (limiter RateLimiter) storeBucket(ctx context.Context, tx Tx, key string, tokens float64, now time.Time) error {
	_, err := tx.Exec(ctx, `
		insert into rate_limit_buckets (key, tokens, updated_at)
		values ($1, $2, $3)
		on conflict (key) do update
		set tokens = excluded.tokens, updated_at = excluded.updated_at
	`, key, tokens, now)
	return err
}

func (limiter RateLimiter) evictExpired(ctx context.Context) {
	_, err := limiter.db.Exec(ctx, `
		delete from rate_limit_buckets
		where key like $1 and updated_at < $2
	`, limiter.prefix+":%", limiter.now().UTC().Add(-limiter.fullAfter))
	if err != nil {
		panic(fmt.Sprintf("evict rate limit buckets failed: %v", err))
	}
}

func (limiter RateLimiter) bucketKey(key string) string {
	return limiter.prefix + ":" + key
}
