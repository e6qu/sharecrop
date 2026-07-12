//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/wasibridge/privacybridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// TestPrivacyBridgeDualRun exercises the privacy service - an internal/http
// RuntimeState service - through the compiled wasip1 guest + host bridge: create
// a request, list it (scoped and unscoped), resolve it, record a sensitive-field
// access (with an empty-field submission, so no FK setup is needed while still
// carrying a submission across the bridge), and run retention.
func TestPrivacyBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewPrivacyStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return privacybridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := privacybridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	requester := createUser(t, pool, "privacy-requester")
	page := requirePage(t, 50, 0)

	var requestID string

	t.Run("create then list for requester matches a direct call", func(t *testing.T) {
		saved, matched := bridgeStore.Create(ctx, requester, "data_export").(httpserver.PrivacyRequestSaved)
		if !matched {
			t.Fatalf("bridge Create did not save")
		}
		if saved.Value.RequestedBy != requester || saved.Value.Kind != "data_export" || saved.Value.State != "queued" {
			t.Errorf("created request = %+v", saved.Value)
		}
		requestID = saved.Value.ID

		viaBridge := requirePrivacyListed(t, bridgeStore.ListForRequester(ctx, requester, page))
		direct := requirePrivacyListed(t, dbStore.ListForRequester(ctx, requester, page))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("request counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
		if viaBridge[0].ID != direct[0].ID || viaBridge[0].ID != requestID {
			t.Errorf("listed request = %s, want %s", viaBridge[0].ID, requestID)
		}
	})

	t.Run("resolve through the bridge marks the request resolved", func(t *testing.T) {
		saved, matched := bridgeStore.Resolve(ctx, requestID, "handled").(httpserver.PrivacyRequestSaved)
		if !matched {
			t.Fatalf("bridge Resolve did not save")
		}
		if saved.Value.State != "resolved" || saved.Value.ResolutionNote != "handled" {
			t.Errorf("resolved request = %+v", saved.Value)
		}
	})

	t.Run("record sensitive-field access carries a submission across the bridge", func(t *testing.T) {
		submissionID, matched := core.NewSubmissionID().(core.SubmissionIDCreated)
		if !matched {
			t.Fatalf("submission id rejected")
		}
		// An empty-field submission records nothing (no FK needed) but still round-
		// trips the submission's id through the bridge.
		if _, matched := bridgeStore.RecordSensitiveFieldAccess(ctx, requester, submission.Submission{ID: submissionID.Value}).(httpserver.PrivacyRequestSaved); !matched {
			t.Fatalf("bridge RecordSensitiveFieldAccess did not save")
		}
	})

	t.Run("run retention through the bridge", func(t *testing.T) {
		if _, matched := bridgeStore.RunRetention(ctx, requester).(httpserver.PrivacyRetentionRun); !matched {
			t.Fatalf("bridge RunRetention did not run")
		}
	})
}

func requirePrivacyListed(t *testing.T, result httpserver.PrivacyListResult) []httpserver.PrivacyRequestRecord {
	t.Helper()
	listed, matched := result.(httpserver.PrivacyRequestsListed)
	if !matched {
		t.Fatalf("list result = %T, want listed", result)
	}
	return listed.Values
}
