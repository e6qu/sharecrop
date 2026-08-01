//go:build integration

package integration_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/webhook"
	"github.com/e6qu/sharecrop/internal/webhookdispatch"
)

// verifyingReceiver implements the receiver-side contract documented in
// docs/api_reference.md ("Signature Verification") and mirrored by the
// runnable sample in tools/webhook_receiver_sample.ts: recompute
// hex(HMAC-SHA256(secret, "<timestamp>.<raw body>")) with a "v1=" prefix,
// compare in constant time, reject stale timestamps, and dedupe by the
// delivery id header. It is deliberately independent of the dispatcher's own
// signature helper, so the test fails if the wire format drifts from the
// documented recipe.
type verifyingReceiver struct {
	mu       sync.Mutex
	secret   string
	seenIDs  map[string]int
	accepted int
	failures []string
	server   *httptest.Server
}

const receiverTimestampSkewLimit = 5 * time.Minute

func newVerifyingReceiver(t *testing.T) *verifyingReceiver {
	t.Helper()
	receiver := &verifyingReceiver{seenIDs: map[string]int{}}
	receiver.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		reason := receiver.verify(r.Header, body)
		receiver.mu.Lock()
		if reason == "" {
			receiver.accepted++
		} else {
			receiver.failures = append(receiver.failures, reason)
		}
		receiver.mu.Unlock()
		if reason != "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(receiver.server.Close)
	return receiver
}

// verify reports the empty string on success, or the rejection reason.
func (receiver *verifyingReceiver) verify(headers http.Header, body []byte) string {
	id := headers.Get("Sharecrop-Webhook-Id")
	rawTimestamp := headers.Get("Sharecrop-Webhook-Timestamp")
	signature := headers.Get("Sharecrop-Webhook-Signature")
	if id == "" || rawTimestamp == "" || signature == "" {
		return "missing header"
	}
	timestamp, err := strconv.ParseInt(rawTimestamp, 10, 64)
	if err != nil {
		return "malformed timestamp"
	}
	skew := time.Since(time.Unix(timestamp, 0))
	if skew < -receiverTimestampSkewLimit || skew > receiverTimestampSkewLimit {
		return "stale timestamp"
	}
	mac := hmac.New(sha256.New, []byte(receiver.secret))
	mac.Write([]byte(rawTimestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := "v1=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return "signature mismatch"
	}
	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	receiver.seenIDs[id]++
	if receiver.seenIDs[id] > 1 {
		return "duplicate delivery id"
	}
	return ""
}

// TestDispatcherDeliveryPassesReceiverVerification proves a real delivery
// from the dispatcher verifies end to end at a receiver that implements the
// documented recipe: all three headers present, the timestamp fresh, and the
// signature matching the raw body. The production dial policy blocks
// loopback, so the test uses the same permissive test policy the other
// dispatch tests use.
func TestDispatcherDeliveryPassesReceiverVerification(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	parkActiveSubscriptions(t, pool)
	dispatchAllRecordedEvents(t, pool)
	parkPendingDeliveries(t, pool)
	receiver := newVerifyingReceiver(t)

	recipient := createUser(t, pool, "verify-recipient")
	created, createdMatched := webhook.NewService(store).Create(context.Background(), webhook.OwnerUser{ID: recipient},
		webhook.NewEndpointURL(receiver.server.URL+"/hooks").(webhook.EndpointURLAccepted).Value,
		webhook.NewKindFilter([]event.Kind{event.KindTaskOpened}).(webhook.KindFilterAccepted).Value,
		webhook.RecipientAudience{},
	).(webhook.SubscriptionCreated)
	if !createdMatched {
		t.Fatalf("create subscription rejected")
	}
	receiver.secret = created.Secret.String()

	dispatchStoredEvent(t, pool, appendWebhookTestEvent(t, pool, event.KindTaskOpened, recipient, []core.UserID{recipient}, event.NoOrganization{}))

	tlsReceiver := &webhookReceiver{server: receiver.server}
	dispatcher := newTestDispatcher(t, pool, tlsReceiver, webhookdispatch.AllowEveryAddress())
	completed := runDispatcherOnce(t, dispatcher)
	if completed.Delivered != 1 {
		t.Fatalf("run delivered %d, want the one verified delivery", completed.Delivered)
	}

	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	if len(receiver.failures) != 0 {
		t.Fatalf("receiver rejected deliveries: %v", receiver.failures)
	}
	if receiver.accepted != 1 || len(receiver.seenIDs) != 1 {
		t.Fatalf("receiver accepted %d deliveries across %d ids, want exactly one", receiver.accepted, len(receiver.seenIDs))
	}
}
