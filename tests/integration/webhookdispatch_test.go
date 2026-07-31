//go:build integration

package integration_test

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge/appmux"
	"github.com/e6qu/sharecrop/internal/webhook"
	"github.com/e6qu/sharecrop/internal/webhookdispatch"
	"github.com/jackc/pgx/v5/pgxpool"
)

// recordedWebhookRequest captures one receiver-side delivery for assertions.
type recordedWebhookRequest struct {
	id        string
	timestamp string
	signature string
	body      []byte
}

type webhookReceiver struct {
	mu       sync.Mutex
	status   int
	requests []recordedWebhookRequest
	server   *httptest.Server
}

func newWebhookReceiver(t *testing.T, status int) *webhookReceiver {
	t.Helper()
	receiver := &webhookReceiver{status: status}
	receiver.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receiver.mu.Lock()
		receiver.requests = append(receiver.requests, recordedWebhookRequest{
			id:        r.Header.Get(webhookdispatch.HeaderWebhookID),
			timestamp: r.Header.Get(webhookdispatch.HeaderWebhookTimestamp),
			signature: r.Header.Get(webhookdispatch.HeaderWebhookSignature),
			body:      body,
		})
		receiver.mu.Unlock()
		w.WriteHeader(receiver.status)
	}))
	t.Cleanup(receiver.server.Close)
	return receiver
}

func (receiver *webhookReceiver) recorded() []recordedWebhookRequest {
	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	values := make([]recordedWebhookRequest, len(receiver.requests))
	copy(values, receiver.requests)
	return values
}

// newWebhookDispatchServer stands up the real HTTP API over the db pool for
// the management calls the scenario makes.
func newWebhookDispatchServer(t *testing.T, pool *pgxpool.Pool) *httptest.Server {
	t.Helper()
	secret := requireAccessTokenSecret(t, appRouteSecret)
	server := httptest.NewServer(appmux.New(secret, appmuxStores(pool)))
	t.Cleanup(server.Close)
	return server
}

func webhookDispatchPost(t *testing.T, serverURL string, path string, token string, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, serverURL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post %s: %v", path, err)
	}
	return response
}

func registerWebhookDispatchUser(t *testing.T, serverURL string) (subjectID string, accessToken string) {
	t.Helper()
	response := webhookDispatchPost(t, serverURL, "/api/auth/register", "",
		fmt.Sprintf(`{"email":%q,"password":"correct horse battery staple"}`, uniqueIntegrationEmail(t, "webhook-dispatch")))
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", response.StatusCode)
	}
	var registered struct {
		SubjectID   string `json:"subject_id"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&registered); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	return registered.SubjectID, registered.AccessToken
}

func newTestDispatcher(t *testing.T, pool *pgxpool.Pool, receiver *webhookReceiver, policy webhookdispatch.DialPolicy) webhookdispatch.Dispatcher {
	t.Helper()
	dispatcher := webhookdispatch.New(db.NewWebhookStore(pool), policy, time.Now, rand.NewPCG(1, 2))
	if receiver != nil {
		transport, isTransport := receiver.server.Client().Transport.(*http.Transport)
		if !isTransport {
			t.Fatalf("receiver client transport is not *http.Transport")
		}
		dispatcher = dispatcher.WithTLSClientConfig(transport.TLSClientConfig)
	}
	return dispatcher
}

func runDispatcherOnce(t *testing.T, dispatcher webhookdispatch.Dispatcher) webhookdispatch.RunCompleted {
	t.Helper()
	result := dispatcher.RunOnce(context.Background())
	completed, matched := result.(webhookdispatch.RunCompleted)
	if !matched {
		t.Fatalf("run rejected: %#v", result)
	}
	return completed
}

// TestWebhookDispatchEndToEnd is the full loop: a subscription created over
// the HTTP API, a real service action (task open) emitting the event, and
// RunOnce delivering exactly one correctly signed request to the receiver.
func TestWebhookDispatchEndToEnd(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	drainWebhookPump(t, store)
	parkPendingDeliveries(t, pool)
	server := newWebhookDispatchServer(t, pool)
	receiver := newWebhookReceiver(t, http.StatusOK)

	_, accessToken := registerWebhookDispatchUser(t, server.URL)
	_, taskID := createAndOpenWebhookDispatchTask(t, server.URL, accessToken, receiver, store)

	dispatcher := newTestDispatcher(t, pool, receiver, webhookdispatch.AllowEveryAddress())
	completed := runDispatcherOnce(t, dispatcher)
	if completed.Delivered < 1 {
		t.Fatalf("run delivered %d, want at least the receiver's delivery", completed.Delivered)
	}

	recorded := receiver.recorded()
	if len(recorded) != 1 {
		t.Fatalf("receiver got %d requests, want exactly 1", len(recorded))
	}
	request := recorded[0]
	if request.id == "" || request.timestamp == "" {
		t.Fatalf("delivery headers missing: %+v", request)
	}

	var deliveredBody struct {
		Event struct {
			Kind   string `json:"kind"`
			TaskID string `json:"task_id"`
			Cursor string `json:"cursor"`
		} `json:"event"`
		SubscriptionID string `json:"subscription_id"`
	}
	if err := json.Unmarshal(request.body, &deliveredBody); err != nil {
		t.Fatalf("decode delivery body: %v", err)
	}
	if deliveredBody.Event.Kind != "task_opened" || deliveredBody.Event.TaskID != taskID {
		t.Fatalf("delivered event = %+v, want task_opened for %s", deliveredBody.Event, taskID)
	}
	subscriptionSecret := webhookDispatchSecrets[t.Name()]
	verifyWebhookSignature(t, subscriptionSecret, request)
	if deliveredBody.SubscriptionID == "" {
		t.Fatalf("delivery body carries no subscription id")
	}
}

// webhookDispatchSecrets passes the create-time secret from the scenario
// helper to its test without widening helper signatures.
var webhookDispatchSecrets = map[string]string{}

// createAndOpenWebhookDispatchTask creates the subscription over the API,
// then performs the real service action (create + open task) that emits the
// task_opened event.
func createAndOpenWebhookDispatchTask(t *testing.T, serverURL string, accessToken string, receiver *webhookReceiver, store db.WebhookStore) (subjectID string, taskID string) {
	t.Helper()
	createSubResponse := webhookDispatchPost(t, serverURL, "/api/webhook-subscriptions", accessToken,
		`{"url":"`+receiver.server.URL+`/hooks","kinds":["task_opened"]}`)
	defer createSubResponse.Body.Close()
	if createSubResponse.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createSubResponse.Body)
		t.Fatalf("create subscription status = %d (%s)", createSubResponse.StatusCode, body)
	}
	var createdSub struct {
		Subscription struct {
			ID          string `json:"id"`
			OwnerUserID string `json:"owner_user_id"`
		} `json:"subscription"`
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(createSubResponse.Body).Decode(&createdSub); err != nil {
		t.Fatalf("decode subscription: %v", err)
	}
	if !strings.HasPrefix(createdSub.Secret, "scrop_whsec_") {
		t.Fatalf("subscription secret = %q", createdSub.Secret)
	}
	webhookDispatchSecrets[t.Name()] = createdSub.Secret

	taskBody := `{
		"owner":{"kind":"user","user_id":"` + createdSub.Subscription.OwnerUserID + `","team_id":"","organization_id":""},
		"title":"Webhook dispatch task",
		"description":"A task whose opening is delivered to a webhook receiver.",
		"reward":{"kind":"none","credit_amount":0},
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
	createTaskResponse := webhookDispatchPost(t, serverURL, "/api/tasks", accessToken, taskBody)
	defer createTaskResponse.Body.Close()
	if createTaskResponse.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createTaskResponse.Body)
		t.Fatalf("create task status = %d (%s)", createTaskResponse.StatusCode, body)
	}
	var createdTask struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(createTaskResponse.Body).Decode(&createdTask); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	openResponse := webhookDispatchPost(t, serverURL, "/api/tasks/"+createdTask.ID+"/open", accessToken, `{}`)
	defer openResponse.Body.Close()
	if openResponse.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(openResponse.Body)
		t.Fatalf("open task status = %d (%s)", openResponse.StatusCode, body)
	}
	return createdSub.Subscription.OwnerUserID, createdTask.ID
}

func verifyWebhookSignature(t *testing.T, rawSecret string, request recordedWebhookRequest) {
	t.Helper()
	secret, matched := webhook.ParseSecret(rawSecret).(webhook.SecretAccepted)
	if !matched {
		t.Fatalf("recorded secret rejected")
	}
	var timestamp int64
	if _, err := fmt.Sscanf(request.timestamp, "%d", &timestamp); err != nil {
		t.Fatalf("timestamp header %q: %v", request.timestamp, err)
	}
	expected := webhookdispatch.ComputeSignature(secret.Value, timestamp, request.body)
	if !hmac.Equal([]byte(expected), []byte(request.signature)) {
		t.Fatalf("signature = %q, want %q", request.signature, expected)
	}
}

// TestWebhookDispatchRetriesAndDies drives a failing receiver through the
// whole retry walk: the first RunOnce schedules a retry with attempt 1, and
// forcing each scheduled slot into the past walks the delivery to dead after
// six total attempts.
func TestWebhookDispatchRetriesAndDies(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	drainWebhookPump(t, store)
	parkPendingDeliveries(t, pool)
	server := newWebhookDispatchServer(t, pool)
	receiver := newWebhookReceiver(t, http.StatusInternalServerError)

	_, accessToken := registerWebhookDispatchUser(t, server.URL)
	_, _ = createAndOpenWebhookDispatchTask(t, server.URL, accessToken, receiver, store)
	dispatcher := newTestDispatcher(t, pool, receiver, webhookdispatch.AllowEveryAddress())

	subscriptionID := findSubscriptionForSecret(t, pool, webhookDispatchSecrets[t.Name()])
	runDispatcherOnce(t, dispatcher)
	delivery := readDeliveryRow(t, pool, subscriptionID)
	if delivery.state != "pending" || delivery.attemptCount != 1 || delivery.lastStatus != "500" {
		t.Fatalf("after first failure: %+v", delivery)
	}
	if !delivery.nextAttemptAt.After(time.Now().Add(20 * time.Second)) {
		t.Fatalf("first retry scheduled too soon: %v", delivery.nextAttemptAt)
	}

	for attempt := 2; attempt <= 6; attempt++ {
		forceDeliveriesDue(t, pool, subscriptionID)
		runDispatcherOnce(t, dispatcher)
	}
	delivery = readDeliveryRow(t, pool, subscriptionID)
	if delivery.state != "dead" || delivery.attemptCount != 6 {
		t.Fatalf("after the walk: %+v, want dead after 6 attempts", delivery)
	}
	if got := len(receiver.recorded()); got != 6 {
		t.Fatalf("receiver saw %d attempts, want 6", got)
	}

	// A dead delivery is never attempted again.
	forceDeadDue(t, pool, subscriptionID)
	runDispatcherOnce(t, dispatcher)
	if got := len(receiver.recorded()); got != 6 {
		t.Fatalf("dead delivery was retried: %d attempts", got)
	}
}

// TestWebhookDispatchBlocksPrivateAddresses proves the SSRF guard: the URL
// constructor accepts a well-formed https URL to a private address, and the
// strict dial policy rejects it at delivery time, recording the block.
func TestWebhookDispatchBlocksPrivateAddresses(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	drainWebhookPump(t, store)
	parkPendingDeliveries(t, pool)
	server := newWebhookDispatchServer(t, pool)

	_, accessToken := registerWebhookDispatchUser(t, server.URL)
	createSubResponse := webhookDispatchPost(t, server.URL, "/api/webhook-subscriptions", accessToken,
		`{"url":"https://127.0.0.1:9/hooks","kinds":["task_opened"]}`)
	defer createSubResponse.Body.Close()
	if createSubResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create subscription with a loopback url should pass the form check, got %d", createSubResponse.StatusCode)
	}
	var createdSub struct {
		Subscription struct {
			ID          string `json:"id"`
			OwnerUserID string `json:"owner_user_id"`
		} `json:"subscription"`
	}
	if err := json.NewDecoder(createSubResponse.Body).Decode(&createdSub); err != nil {
		t.Fatalf("decode subscription: %v", err)
	}
	openWebhookDispatchTask(t, server.URL, accessToken, createdSub.Subscription.OwnerUserID)

	dispatcher := newTestDispatcher(t, pool, nil, webhookdispatch.StrictDialPolicy())
	completed := runDispatcherOnce(t, dispatcher)
	if completed.Delivered != 0 {
		t.Fatalf("blocked delivery reported as delivered")
	}

	subscriptionID := mustParseSubscriptionID(t, createdSub.Subscription.ID)
	delivery := readDeliveryRow(t, pool, subscriptionID)
	if delivery.state != "pending" || delivery.attemptCount != 1 {
		t.Fatalf("blocked delivery = %+v, want a scheduled retry", delivery)
	}
	if !strings.Contains(delivery.lastStatus, "blocked") {
		t.Fatalf("blocked delivery status = %q, want a blocked marker", delivery.lastStatus)
	}
}

func openWebhookDispatchTask(t *testing.T, serverURL string, accessToken string, subjectID string) {
	t.Helper()
	taskBody := `{
		"owner":{"kind":"user","user_id":"` + subjectID + `","team_id":"","organization_id":""},
		"title":"Webhook blocked-address task",
		"description":"A task whose opening exercises the dial policy.",
		"reward":{"kind":"none","credit_amount":0},
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
	createTaskResponse := webhookDispatchPost(t, serverURL, "/api/tasks", accessToken, taskBody)
	defer createTaskResponse.Body.Close()
	if createTaskResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create task status = %d", createTaskResponse.StatusCode)
	}
	var createdTask struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(createTaskResponse.Body).Decode(&createdTask); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	openResponse := webhookDispatchPost(t, serverURL, "/api/tasks/"+createdTask.ID+"/open", accessToken, `{}`)
	defer openResponse.Body.Close()
	if openResponse.StatusCode != http.StatusOK {
		t.Fatalf("open task status = %d", openResponse.StatusCode)
	}
}

// findSubscriptionForSecret resolves the subscription whose stored signing
// secret matches — possible precisely because secrets are stored as written.
func findSubscriptionForSecret(t *testing.T, pool *pgxpool.Pool, secret string) core.WebhookSubscriptionID {
	t.Helper()
	var raw string
	if err := pool.QueryRow(context.Background(), `select id::text from webhook_subscriptions where secret = $1`, secret).Scan(&raw); err != nil {
		t.Fatalf("find subscription for secret: %v", err)
	}
	return mustParseSubscriptionID(t, raw)
}

func mustParseSubscriptionID(t *testing.T, raw string) core.WebhookSubscriptionID {
	t.Helper()
	parsed, matched := core.ParseWebhookSubscriptionID(raw).(core.WebhookSubscriptionIDCreated)
	if !matched {
		t.Fatalf("subscription id %q rejected", raw)
	}
	return parsed.Value
}

// forceDeadDue rewinds next_attempt_at on every delivery of the subscription
// regardless of state, proving the claim predicate alone (state = pending)
// keeps dead rows out of the dispatch pool.
func forceDeadDue(t *testing.T, pool *pgxpool.Pool, subscriptionID core.WebhookSubscriptionID) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		update webhook_deliveries set next_attempt_at = now() - interval '1 second'
		where subscription_id = $1
	`, subscriptionID.String()); err != nil {
		t.Fatalf("force dead due: %v", err)
	}
}

type deliveryRow struct {
	state         string
	attemptCount  int64
	lastStatus    string
	nextAttemptAt time.Time
}

func readDeliveryRow(t *testing.T, pool *pgxpool.Pool, subscriptionID core.WebhookSubscriptionID) deliveryRow {
	t.Helper()
	var row deliveryRow
	if err := pool.QueryRow(context.Background(), `
		select state, attempt_count, last_status, next_attempt_at
		from webhook_deliveries where subscription_id = $1
	`, subscriptionID.String()).Scan(&row.state, &row.attemptCount, &row.lastStatus, &row.nextAttemptAt); err != nil {
		t.Fatalf("read delivery row: %v", err)
	}
	return row
}
