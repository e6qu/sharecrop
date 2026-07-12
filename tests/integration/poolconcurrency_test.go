//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/wasibridge/notificationbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestPoolServesConcurrentUnitsWithoutCrossTalk is the safety proof for instance
// pooling: many concurrent units of work, each expecting a DISTINCT result, run
// through a pool of fewer long-lived guest instances than there are goroutines,
// so instances are heavily reused under contention. If pooling reintroduced the
// Phase 1 shared-instance corruption, a goroutine would see another recipient's
// notification (or the guest would crash). Each goroutine must see exactly its
// own recipient's single seeded notification, proving each pooled instance's
// state stays private to the one unit of work it is serving at a time.
func TestPoolServesConcurrentUnitsWithoutCrossTalk(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewNotificationStore(pool)

	const recipients = 24
	const poolSize = 4
	const rounds = 6

	type seed struct {
		recipient core.UserID
		notifyID  string
	}
	seeds := make([]seed, recipients)
	for i := range seeds {
		recipient := createUser(t, pool, fmt.Sprintf("pool-recipient-%d", i))
		actor := createUser(t, pool, fmt.Sprintf("pool-actor-%d", i))
		seeded := seedNotification(t, ctx, dbStore, recipient, actor)
		seeds[i] = seed{recipient: recipient, notifyID: seeded.ID.String()}
	}

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	rpcPool, err := rpc.NewPool(ctx, guestWASM, storehost.Dispatcher(pool), poolSize)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(func() { _ = rpcPool.Close(ctx) })

	bridgeStore := notificationbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return rpcPool.Call(ctx, method, args)
	})
	page := requirePage(t, 50, 0)

	// Each goroutine hammers its recipient's list several times; with poolSize <
	// recipients, every instance serves many different recipients in sequence.
	var wg sync.WaitGroup
	failures := make(chan string, recipients*rounds)
	for round := 0; round < rounds; round++ {
		for i := range seeds {
			wg.Add(1)
			go func(s seed) {
				defer wg.Done()
				listed, matched := bridgeStore.List(ctx, s.recipient, page).(notification.ListStoreAccepted)
				if !matched {
					failures <- fmt.Sprintf("recipient %s: list rejected", s.recipient)
					return
				}
				if len(listed.Values) != 1 {
					failures <- fmt.Sprintf("recipient %s: got %d notifications, want 1", s.recipient, len(listed.Values))
					return
				}
				if got := listed.Values[0].ID.String(); got != s.notifyID {
					failures <- fmt.Sprintf("recipient %s: got notification %s, want %s (cross-talk)", s.recipient, got, s.notifyID)
				}
			}(seeds[i])
		}
	}
	wg.Wait()
	close(failures)

	for message := range failures {
		t.Error(message)
	}
}
