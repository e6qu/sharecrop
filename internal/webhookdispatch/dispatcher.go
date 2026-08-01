//go:build !wasip1

// Package webhookdispatch is the host-only webhook delivery engine: it
// claims due pending delivery rows (created by the per-event dispatch step
// in db.EventStore.Dispatch) and dispatches them with signed HTTPS POSTs,
// retrying on a bounded backoff schedule. It is imported ONLY by
// cmd/sharecrop (the lifecycle runner calls RunOnce) and talks to Postgres
// through struct-only methods on db.WebhookStore, so nothing here can ever
// be reached from the WASI guest or the browser build.
package webhookdispatch

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/webhook"
)

const (
	// PumpInterval is the recommended cadence for the lifecycle runner to
	// call RunOnce.
	PumpInterval = 5 * time.Second
	// claimBatchSize is how many due deliveries one claim locks. It stays
	// small so a single claim transaction holds few row locks and a crashed
	// dispatcher strands little work, while RunOnce drains a backlog by
	// claiming repeatedly until a claim comes back short.
	claimBatchSize = 10
	// maxClaimBatchesPerRun caps how many claim-and-deliver rounds one
	// RunOnce performs (claimBatchSize * maxClaimBatchesPerRun deliveries),
	// so a deep backlog cannot monopolize the lifecycle runner's sweep slot;
	// the next tick continues the drain.
	maxClaimBatchesPerRun = 50
	// requestTimeout bounds each delivery POST.
	requestTimeout = 10 * time.Second
	// claimHold is how long a claim keeps its rows out of the due window.
	// It is derived from the batch worst case — every delivery in a claimed
	// batch timing out (claimBatchSize * requestTimeout) — plus one extra
	// requestTimeout of slack, so another replica can never re-claim (and
	// duplicate-POST) a row that a slow batch is still working through. A
	// crashed dispatcher's rows become claimable again once the hold lapses;
	// delivery stays at-least-once (see db.WebhookStore.ClaimDueDeliveries).
	claimHold = claimBatchSize*requestTimeout + requestTimeout
	// responseBodyReadLimit caps how much of a receiver's response body is
	// read (and discarded) before the connection is released.
	responseBodyReadLimit = 4 * 1024
	// lastStatusLimit bounds the recorded last_status string.
	lastStatusLimit = 120
)

// Dispatcher delivers pending webhook rows. Time and randomness are injected
// so tests can walk the retry schedule deterministically.
type Dispatcher struct {
	store  db.WebhookStore
	client *http.Client
	now    func() time.Time
	jitter *rand.Rand
}

// New builds a dispatcher over the given store and dial policy. Production
// passes StrictDialPolicy(); tests pass AllowEveryAddress() plus a client
// TLS config (WithTLSClientConfig) trusting their local receiver.
func New(store db.WebhookStore, policy DialPolicy, now func() time.Time, source rand.Source) Dispatcher {
	dialer := &net.Dialer{
		Timeout: requestTimeout,
		Control: func(network string, address string, _ syscall.RawConn) error {
			return policy(network, address)
		},
	}
	client := &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			DialContext:         dialer.DialContext,
			TLSHandshakeTimeout: requestTimeout,
		},
		// Redirects are never followed: a 3xx answer is returned as-is and
		// counts as a failed attempt, so a receiver cannot bounce the
		// dispatcher toward an address the dial policy never saw.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return Dispatcher{store: store, client: client, now: now, jitter: rand.New(source)}
}

// WithTLSClientConfig returns a copy of the dispatcher whose HTTPS client
// trusts the given TLS configuration. Tests use it to reach an httptest
// receiver with a self-signed certificate; production never calls it.
func (dispatcher Dispatcher) WithTLSClientConfig(config *tls.Config) Dispatcher {
	transport := dispatcher.client.Transport.(*http.Transport).Clone()
	transport.TLSClientConfig = config
	dispatcher.client = &http.Client{
		Timeout:       dispatcher.client.Timeout,
		Transport:     transport,
		CheckRedirect: dispatcher.client.CheckRedirect,
	}
	return dispatcher
}

type RunResult interface {
	runResult()
}

type RunCompleted struct {
	Attempted int
	Delivered int
}

type RunRejected struct {
	Reason core.DomainError
}

func (RunCompleted) runResult() {}

func (RunRejected) runResult() {}

// RunOnce performs one dispatch pass: claim a batch of due deliveries,
// attempt each, and keep claiming until a claim comes back short of the
// batch size (the backlog is drained) or the per-run batch cap is reached.
// Delivery rows are created by the event dispatch step (db.EventStore
// Dispatch), not here. Per-delivery failures are recorded on the delivery
// row (retry schedule or dead), never surfaced as a run failure.
func (dispatcher Dispatcher) RunOnce(ctx context.Context) RunResult {
	attempted := 0
	delivered := 0
	for batch := 0; batch < maxClaimBatchesPerRun; batch++ {
		claimResult := dispatcher.store.ClaimDueDeliveries(ctx, claimBatchSize, claimHold)
		claimed, claimMatched := claimResult.(db.ClaimDueDeliveriesListed)
		if !claimMatched {
			return RunRejected{Reason: claimResult.(db.ClaimDueDeliveriesRejected).Reason}
		}
		attempted += len(claimed.Values)
		for _, delivery := range claimed.Values {
			if dispatcher.attemptDelivery(ctx, delivery) {
				delivered++
			}
		}
		if len(claimed.Values) < claimBatchSize {
			break
		}
	}
	return RunCompleted{Attempted: attempted, Delivered: delivered}
}

// attemptDelivery performs one signed POST and records the outcome. The
// report is true when the receiver acknowledged with a 2xx.
func (dispatcher Dispatcher) attemptDelivery(ctx context.Context, delivery db.ClaimedWebhookDelivery) bool {
	attempt := delivery.AttemptCount + 1
	status, succeeded := dispatcher.postDelivery(ctx, delivery)
	if succeeded {
		_ = dispatcher.store.MarkDelivered(ctx, delivery.ID, attempt, status)
		return true
	}
	switch decision := decideRetry(attempt, dispatcher.now(), dispatcher.jitter).(type) {
	case retryAt:
		_ = dispatcher.store.MarkFailed(ctx, delivery.ID, attempt, status, db.RetryLater{NextAttemptAt: decision.at})
	case retriesExhausted:
		_ = dispatcher.store.MarkFailed(ctx, delivery.ID, attempt, status, db.Dead{})
	}
	return false
}

// postDelivery sends the signed request and classifies the outcome into the
// bounded last_status string plus a success report (2xx only).
func (dispatcher Dispatcher) postDelivery(ctx context.Context, delivery db.ClaimedWebhookDelivery) (string, bool) {
	// Re-check the scheme at delivery time: the URL constructor enforced
	// https at creation, but the dispatcher signs and sends whatever is in
	// the row, so the guarantee is re-established where it matters.
	if !strings.HasPrefix(delivery.URL.String(), "https://") {
		return "blocked: url is not https", false
	}
	body, err := webhook.EncodeDeliveryBody(delivery.Event, delivery.Subscription.String())
	if err != nil {
		return "encode failed", false
	}

	requestCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, http.MethodPost, delivery.URL.String(), bytes.NewReader(body))
	if err != nil {
		return boundedStatus(err.Error()), false
	}
	timestamp := dispatcher.now().Unix()
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(HeaderWebhookID, delivery.ID.String())
	request.Header.Set(HeaderWebhookTimestamp, strconv.FormatInt(timestamp, 10))
	request.Header.Set(HeaderWebhookSignature, ComputeSignature(delivery.Secret, timestamp, body))

	response, err := dispatcher.client.Do(request)
	if err != nil {
		return classifyDeliveryError(err), false
	}
	defer response.Body.Close()
	// Read a bounded slice of the response body and discard it, so the
	// connection can be reused without buffering an attacker-sized reply.
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, responseBodyReadLimit))

	status := strconv.Itoa(response.StatusCode)
	return status, response.StatusCode >= 200 && response.StatusCode <= 299
}

// classifyDeliveryError maps transport failures onto short stable status
// strings; everything unrecognized is recorded as a bounded error excerpt.
func classifyDeliveryError(err error) string {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return "connection refused"
	}
	if strings.Contains(err.Error(), "webhook dial blocked") {
		return boundedStatus("blocked: " + err.Error())
	}
	return boundedStatus(err.Error())
}

func boundedStatus(status string) string {
	if len(status) > lastStatusLimit {
		return status[:lastStatusLimit]
	}
	return status
}
