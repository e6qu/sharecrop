package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// fixedOpsCountersReader serves one fixed snapshot, standing in for the
// host-side database read model.
type fixedOpsCountersReader struct {
	value OpsCountersView
}

func (reader fixedOpsCountersReader) ReadOpsCounters(context.Context) OpsCountersReadResult {
	return OpsCountersRead{Value: reader.value}
}

func countersHandlerWithReader(reader OpsCountersReader) http.Handler {
	runtime := DefaultRuntimeState(ParseAdminUserIDsForRuntime(os.Getenv("SHARECROP_ADMIN_USER_IDS")))
	runtime.OpsCounters = reader
	return NewWithRuntimeState(testStaticFiles(), testAuthService(), testVerifier{}, testOrganizationService{}, testTaskService{}, testSubmissionService{}, testLedgerService{}, testAgentService{}, testOrgCredentialService{}, testAssetService{}, runtime)
}

func getOperationsCounters(handler http.Handler) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/api/admin/operations/counters", nil)
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestOperationsCountersRequirePlatformAdmin(t *testing.T) {
	// The caller authenticates as a regular user (no admin allowlist entry).
	response := getOperationsCounters(countersHandlerWithReader(fixedOpsCountersReader{}))
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestOperationsCountersServeTheSnapshot(t *testing.T) {
	t.Setenv("SHARECROP_ADMIN_USER_IDS", stableTestUserID.String())
	snapshot := OpsCountersView{
		OutboxRecordedBacklog:          4,
		OutboxDispatchFailed:           1,
		WebhookDeliveriesPending:       2,
		WebhookDeliveriesDead:          3,
		OldestPendingWebhookAgeSeconds: 61,
		SignupGrantsToday:              5,
		PeerTransfersToday:             6,
		PeerTransferCreditsToday:       70,
		BudgetRefusalsToday:            8,
	}
	response := getOperationsCounters(countersHandlerWithReader(fixedOpsCountersReader{value: snapshot}))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusOK, response.Body.String())
	}
	var body OpsCountersView
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body != snapshot {
		t.Fatalf("counters = %+v, want %+v", body, snapshot)
	}
}

// TestOperationsCountersUnavailableWithoutHostStore pins the explicit-absence
// behavior of the default runtime (and of the WASI app guest, which wires the
// same reader): the endpoint answers 503 unavailable rather than fabricating
// counters.
func TestOperationsCountersUnavailableWithoutHostStore(t *testing.T) {
	t.Setenv("SHARECROP_ADMIN_USER_IDS", stableTestUserID.String())
	response := getOperationsCounters(testHandler())
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusServiceUnavailable, response.Body.String())
	}
}
