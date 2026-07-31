package db

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
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

// Allow reports whether the key is under its rate budget. On a storage
// failure it DENIES (fails closed): rate limiting is a protection control,
// and a storage outage must not silently disable it. The in-memory limiter
// (internal/http) cannot fail, so the shared bool signature stays honest for
// both implementations.
func (limiter RateLimiter) Allow(key string) bool {
	allowed, err := limiter.allow(context.Background(), key)
	if err != nil {
		return false
	}
	return allowed
}

func (limiter RateLimiter) allow(ctx context.Context, key string) (bool, error) {
	tx, err := limiter.db.Begin(ctx)
	if err != nil {
		return false, err
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
		return false, err
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
		return false, err
	}
	tokens += now.Sub(updatedAt).Seconds() * limiter.refillPerSec
	if tokens > limiter.capacity {
		tokens = limiter.capacity
	}

	if tokens < 1 {
		if err := limiter.storeBucket(ctx, tx, bucketKey, tokens, now); err != nil {
			return false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return false, nil
	}

	if err := limiter.storeBucket(ctx, tx, bucketKey, tokens-1, now); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// ActiveBuckets counts this limiter's live buckets after evicting the stale
// ones, reporting a storage failure as the rejection variant instead of
// panicking.
func (limiter RateLimiter) ActiveBuckets() httpserver.ActiveBucketsResult {
	ctx := context.Background()
	if err := limiter.EvictExpired(ctx); err != nil {
		return httpserver.ActiveBucketsUnavailable{Reason: core.NewDomainError(core.ErrorCodeUnavailable, "evict rate limit buckets failed")}
	}
	var count int
	err := limiter.db.QueryRow(ctx, `
		select count(*)
		from rate_limit_buckets
		where key like $1
	`, limiter.prefix+":%").Scan(&count)
	if err != nil {
		return httpserver.ActiveBucketsUnavailable{Reason: core.NewDomainError(core.ErrorCodeUnavailable, "count rate limit buckets failed")}
	}
	return httpserver.ActiveBucketsCounted{Count: count}
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

// EvictExpired removes this limiter's buckets that have been idle long enough
// to be full again, so keys cannot accumulate without bound. The lifecycle
// runner calls it on a fixed cadence; ActiveBuckets also runs it before
// counting.
func (limiter RateLimiter) EvictExpired(ctx context.Context) error {
	_, err := limiter.db.Exec(ctx, `
		delete from rate_limit_buckets
		where key like $1 and updated_at < $2
	`, limiter.prefix+":%", limiter.now().UTC().Add(-limiter.fullAfter))
	return err
}

func (limiter RateLimiter) bucketKey(key string) string {
	return limiter.prefix + ":" + key
}
