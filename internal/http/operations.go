package httpserver

import (
	"context"
	"net/http"

	"github.com/e6qu/sharecrop/internal/core"
)

type operationsResponse struct {
	Status                   string `json:"status"`
	AccountTokenDelivery     string `json:"account_token_delivery"`
	MCPStorage               string `json:"mcp_storage"`
	RateLimitStorage         string `json:"rate_limit_storage"`
	ActiveMCPSessions        int    `json:"active_mcp_sessions"`
	ActiveIPRateBuckets      int    `json:"active_ip_rate_buckets"`
	ActiveSubjectRateBuckets int    `json:"active_subject_rate_buckets"`
	SecureCookies            string `json:"secure_cookies"`
}

func (operationsResponse) writableResponse() {}

func (server Server) operationsStatus(w http.ResponseWriter, r *http.Request) {
	_, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}
	secureCookies := "enabled"
	if !server.secureCookies {
		secureCookies = "disabled"
	}
	ipBucketsResult := server.ipRateLimiter.ActiveBuckets()
	ipBuckets, ipMatched := ipBucketsResult.(ActiveBucketsCounted)
	if !ipMatched {
		writeDomainError(w, ipBucketsResult.(ActiveBucketsUnavailable).Reason)
		return
	}
	subjectBucketsResult := server.subjectRateLimiter.ActiveBuckets()
	subjectBuckets, subjectMatched := subjectBucketsResult.(ActiveBucketsCounted)
	if !subjectMatched {
		writeDomainError(w, subjectBucketsResult.(ActiveBucketsUnavailable).Reason)
		return
	}
	writeJSON(w, http.StatusOK, operationsResponse{
		Status:                   "ok",
		AccountTokenDelivery:     server.accountTokens.mode,
		MCPStorage:               server.mcpSessions.storageKind(),
		RateLimitStorage:         server.ipRateLimiter.StorageKind(),
		ActiveMCPSessions:        server.mcpSessions.activeSessionCount(),
		ActiveIPRateBuckets:      ipBuckets.Count,
		ActiveSubjectRateBuckets: subjectBuckets.Count,
		SecureCookies:            secureCookies,
	})
}

// OpsCountersView is one snapshot of the operational aggregates behind
// GET /api/admin/operations/counters, already flattened to the wire shape.
// The day totals cover the current UTC calendar day.
// OldestPendingWebhookAgeSeconds is 0 when no delivery is pending (a pending
// delivery younger than one second also reports 0).
type OpsCountersView struct {
	OutboxRecordedBacklog          int64 `json:"outbox_recorded_backlog"`
	OutboxDispatchFailed           int64 `json:"outbox_dispatch_failed"`
	WebhookDeliveriesPending       int64 `json:"webhook_deliveries_pending"`
	WebhookDeliveriesDead          int64 `json:"webhook_deliveries_dead"`
	OldestPendingWebhookAgeSeconds int64 `json:"oldest_pending_webhook_age_seconds"`
	SignupGrantsToday              int64 `json:"signup_grants_today"`
	PeerTransfersToday             int64 `json:"peer_transfers_today"`
	PeerTransferCreditsToday       int64 `json:"peer_transfer_credits_today"`
	BudgetRefusalsToday            int64 `json:"budget_refusals_today"`
}

func (OpsCountersView) writableResponse() {}

// OpsCountersReader reads one consistent operations-counters snapshot. The
// production implementation is internal/db's OpsCountersStore (which
// structurally satisfies this interface); runtimes without direct database
// access wire NewUnavailableOpsCountersReader.
type OpsCountersReader interface {
	ReadOpsCounters(ctx context.Context) OpsCountersReadResult
}

type OpsCountersReadResult interface {
	opsCountersReadResult()
}

type OpsCountersRead struct {
	Value OpsCountersView
}

type OpsCountersReadRejected struct {
	Reason core.DomainError
}

func (OpsCountersRead) opsCountersReadResult() {}

func (OpsCountersReadRejected) opsCountersReadResult() {}

// unavailableOpsCountersReader is the explicit absence of the counters read
// model: the WASI app guest and the in-memory default runtime have no
// host-side Postgres aggregates to read. In WASI hosting the host serves
// GET /api/admin/operations/counters natively (the route never reaches the
// guest), so this reader answers only on runtimes that genuinely cannot
// provide the counters.
type unavailableOpsCountersReader struct{}

func (unavailableOpsCountersReader) ReadOpsCounters(context.Context) OpsCountersReadResult {
	return OpsCountersReadRejected{Reason: core.NewDomainError(core.ErrorCodeUnavailable, "operations counters are not available on this runtime")}
}

// NewUnavailableOpsCountersReader builds the reader for runtimes without the
// host-side counters store.
func NewUnavailableOpsCountersReader() OpsCountersReader {
	return unavailableOpsCountersReader{}
}

// operationsCounters serves the platform-admin operations counters: outbox
// backlog, webhook delivery health, and the current UTC day's economy totals.
func (server Server) operationsCounters(w http.ResponseWriter, r *http.Request) {
	_, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}
	result := server.opsCounters.ReadOpsCounters(r.Context())
	read, matched := result.(OpsCountersRead)
	if !matched {
		writeDomainError(w, result.(OpsCountersReadRejected).Reason)
		return
	}
	writeJSON(w, http.StatusOK, read.Value)
}
