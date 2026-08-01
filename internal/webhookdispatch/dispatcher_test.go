//go:build !wasip1

package webhookdispatch

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/webhook"
	"github.com/jackc/pgx/v5/pgxpool"
)

// unreachableStore builds a real WebhookStore over a pool pointing at a
// closed port; the pool connects lazily, so construction succeeds and every
// query fails. This exercises the dispatcher's store-failure paths without a
// database.
func unreachableStore(t *testing.T) db.WebhookStore {
	t.Helper()
	config, err := pgxpool.ParseConfig("postgres://nobody:nothing@127.0.0.1:1/nowhere?sslmode=disable")
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return db.NewWebhookStore(pool)
}

func fixtureClaimedDelivery(t *testing.T, rawURL string) db.ClaimedWebhookDelivery {
	t.Helper()
	endpoint, matched := webhook.NewEndpointURL(rawURL).(webhook.EndpointURLAccepted)
	if !matched {
		t.Fatalf("fixture url rejected: %q", rawURL)
	}
	taskID := core.NewTaskID().(core.TaskIDCreated).Value
	subject := event.NoSubjectRefs()
	subject.Task = event.TaskSubject{ID: taskID}
	return db.ClaimedWebhookDelivery{
		ID:           core.NewWebhookDeliveryID().(core.WebhookDeliveryIDCreated).Value,
		Subscription: core.NewWebhookSubscriptionID().(core.WebhookSubscriptionIDCreated).Value,
		URL:          endpoint.Value,
		Secret:       webhook.NewSecret().(webhook.SecretAccepted).Value,
		AttemptCount: 0,
		Event: event.StoredEvent{
			Event: event.Event{
				ID:         core.NewDomainEventID().(core.DomainEventIDCreated).Value,
				Kind:       event.KindTaskOpened,
				Actor:      event.ActorSystem{},
				Subject:    subject,
				Metadata:   event.EmptyMetadata(),
				OccurredAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			},
			Cursor: event.CursorFromSequence(9),
		},
	}
}

func TestRunOnceSurfacesClaimFailure(t *testing.T) {
	dispatcher := New(unreachableStore(t), StrictDialPolicy(), time.Now, rand.NewPCG(1, 2))
	result := dispatcher.RunOnce(context.Background())
	if _, matched := result.(RunRejected); !matched {
		t.Fatalf("run over an unreachable store = %#v, want RunRejected", result)
	}
}

func TestAttemptDeliveryPostsOneSignedRequest(t *testing.T) {
	var mu sync.Mutex
	var gotBody []byte
	var gotHeaders http.Header
	receiver := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		gotBody = body
		gotHeaders = r.Header.Clone()
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	dispatcher := New(unreachableStore(t), AllowEveryAddress(), func() time.Time { return now }, rand.NewPCG(1, 2)).
		WithTLSClientConfig(receiver.Client().Transport.(*http.Transport).TLSClientConfig)

	claimed := fixtureClaimedDelivery(t, receiver.URL+"/hooks")
	if !dispatcher.attemptDelivery(context.Background(), claimed) {
		t.Fatalf("delivery to a 200 receiver reported failure")
	}

	mu.Lock()
	defer mu.Unlock()
	if gotHeaders.Get("Content-Type") != "application/json" {
		t.Fatalf("content type = %q", gotHeaders.Get("Content-Type"))
	}
	if gotHeaders.Get(HeaderWebhookID) != claimed.ID.String() {
		t.Fatalf("webhook id header = %q, want %q", gotHeaders.Get(HeaderWebhookID), claimed.ID.String())
	}
	if gotHeaders.Get(HeaderWebhookTimestamp) != strconv.FormatInt(now.Unix(), 10) {
		t.Fatalf("timestamp header = %q", gotHeaders.Get(HeaderWebhookTimestamp))
	}
	expectedSignature := ComputeSignature(claimed.Secret, now.Unix(), gotBody)
	if !hmac.Equal([]byte(gotHeaders.Get(HeaderWebhookSignature)), []byte(expectedSignature)) {
		t.Fatalf("signature = %q, want %q", gotHeaders.Get(HeaderWebhookSignature), expectedSignature)
	}

	var body struct {
		Event struct {
			Kind   string `json:"kind"`
			Cursor string `json:"cursor"`
		} `json:"event"`
		SubscriptionID string `json:"subscription_id"`
	}
	if err := json.Unmarshal(gotBody, &body); err != nil {
		t.Fatalf("decode delivered body: %v", err)
	}
	if body.Event.Kind != "task_opened" || body.Event.Cursor != "9" || body.SubscriptionID != claimed.Subscription.String() {
		t.Fatalf("delivered body = %+v", body)
	}
}

func TestAttemptDeliveryFailsOnServerError(t *testing.T) {
	receiver := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer receiver.Close()

	dispatcher := New(unreachableStore(t), AllowEveryAddress(), time.Now, rand.NewPCG(1, 2)).
		WithTLSClientConfig(receiver.Client().Transport.(*http.Transport).TLSClientConfig)
	if dispatcher.attemptDelivery(context.Background(), fixtureClaimedDelivery(t, receiver.URL+"/hooks")) {
		t.Fatalf("delivery to a 500 receiver reported success")
	}
}

func TestAttemptDeliveryFailsOnRedirect(t *testing.T) {
	receiver := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://elsewhere.example.com/hooks", http.StatusTemporaryRedirect)
	}))
	defer receiver.Close()

	dispatcher := New(unreachableStore(t), AllowEveryAddress(), time.Now, rand.NewPCG(1, 2)).
		WithTLSClientConfig(receiver.Client().Transport.(*http.Transport).TLSClientConfig)
	// Redirects are not followed: the 307 is the final answer and counts as
	// a failed attempt.
	if dispatcher.attemptDelivery(context.Background(), fixtureClaimedDelivery(t, receiver.URL+"/hooks")) {
		t.Fatalf("redirecting receiver reported success")
	}
}

func TestStrictDialPolicyBlocksLoopbackReceivers(t *testing.T) {
	requests := 0
	receiver := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))
	defer receiver.Close()

	dispatcher := New(unreachableStore(t), StrictDialPolicy(), time.Now, rand.NewPCG(1, 2)).
		WithTLSClientConfig(receiver.Client().Transport.(*http.Transport).TLSClientConfig)
	if dispatcher.attemptDelivery(context.Background(), fixtureClaimedDelivery(t, receiver.URL+"/hooks")) {
		t.Fatalf("strict policy delivered to a loopback receiver")
	}
	if requests != 0 {
		t.Fatalf("strict policy let %d requests through", requests)
	}
	if err := AllowEveryAddress()("tcp", "127.0.0.1:443"); err != nil {
		t.Fatalf("permissive policy blocked loopback: %v", err)
	}
}

func TestClassifyDeliveryError(t *testing.T) {
	if got := classifyDeliveryError(syscall.ECONNREFUSED); got != "connection refused" {
		t.Fatalf("refused = %q", got)
	}
	if got := classifyDeliveryError(context.DeadlineExceeded); got != "timeout" {
		t.Fatalf("deadline = %q", got)
	}
	if got := classifyDeliveryError(errors.New("webhook dial blocked: 10.0.0.1 is a private address")); got != "blocked: webhook dial blocked: 10.0.0.1 is a private address" {
		t.Fatalf("blocked = %q", got)
	}
	if got := classifyDeliveryError(errors.New("mystery failure")); got != "mystery failure" {
		t.Fatalf("unclassified = %q", got)
	}
}
