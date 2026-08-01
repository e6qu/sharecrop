package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/webhook"
)

func webhookTestHandler(t *testing.T) http.Handler {
	t.Helper()
	runtime := DefaultRuntimeState(map[string]bool{})
	return NewWithRuntimeState(testStaticFiles(), testAuthService(), testVerifier{}, testOrganizationService{}, testTaskService{}, testSubmissionService{}, testLedgerService{}, testAgentService{}, testOrgCredentialService{}, testAssetService{}, runtime)
}

type decodedWebhookSubscription struct {
	ID                  string   `json:"id"`
	OwnerKind           string   `json:"owner_kind"`
	OwnerUserID         string   `json:"owner_user_id"`
	OwnerOrganizationID string   `json:"owner_organization_id"`
	URL                 string   `json:"url"`
	Kinds               []string `json:"kinds"`
	State               string   `json:"state"`
}

type decodedWebhookCreated struct {
	Subscription decodedWebhookSubscription `json:"subscription"`
	Secret       string                     `json:"secret"`
}

func createTestWebhookSubscription(t *testing.T, handler http.Handler, body string) decodedWebhookCreated {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/webhook-subscriptions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d (%s)", response.Code, http.StatusCreated, response.Body.String())
	}
	var created decodedWebhookCreated
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	return created
}

func TestCreateWebhookSubscriptionReturnsTheSecretExactlyOnce(t *testing.T) {
	handler := webhookTestHandler(t)

	created := createTestWebhookSubscription(t, handler, `{"url":"https://receiver.example.com/hooks","kinds":["task_opened","submission_created"]}`)
	if !strings.HasPrefix(created.Secret, "scrop_whsec_") {
		t.Fatalf("secret = %q, want scrop_whsec_ prefix", created.Secret)
	}
	if created.Subscription.OwnerKind != "user" || created.Subscription.OwnerUserID != stableTestUserID.String() {
		t.Fatalf("owner = %+v, want the acting user", created.Subscription)
	}
	if created.Subscription.State != "active" || len(created.Subscription.Kinds) != 2 {
		t.Fatalf("subscription = %+v", created.Subscription)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/webhook-subscriptions", nil)
	listRequest.Header.Set("Authorization", "Bearer test-access-token")
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d (%s)", listResponse.Code, listResponse.Body.String())
	}
	if strings.Contains(listResponse.Body.String(), created.Secret) {
		t.Fatalf("listing leaked the signing secret: %s", listResponse.Body.String())
	}
	var listed struct {
		Subscriptions []decodedWebhookSubscription `json:"subscriptions"`
		NextOffset    int                          `json:"next_offset"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listed.Subscriptions) != 1 || listed.Subscriptions[0].ID != created.Subscription.ID || listed.NextOffset != 0 {
		t.Fatalf("listing = %+v", listed)
	}
}

func TestCreateWebhookSubscriptionRejectsBadInput(t *testing.T) {
	handler := webhookTestHandler(t)

	cases := map[string]string{
		"http url":        `{"url":"http://receiver.example.com/hooks","kinds":["task_opened"]}`,
		"userinfo url":    `{"url":"https://user:secret@receiver.example.com/hooks","kinds":["task_opened"]}`,
		"missing host":    `{"url":"https:///hooks","kinds":["task_opened"]}`,
		"no kinds":        `{"url":"https://receiver.example.com/hooks","kinds":[]}`,
		"unknown kind":    `{"url":"https://receiver.example.com/hooks","kinds":["task_meowed"]}`,
		"malformed json":  `{`,
		"organization id": `{"url":"https://receiver.example.com/hooks","kinds":["task_opened"],"organization_id":"not-an-id"}`,
	}
	for name, body := range cases {
		request := httptest.NewRequest(http.MethodPost, "/api/webhook-subscriptions", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer test-access-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want %d (%s)", name, response.Code, http.StatusBadRequest, response.Body.String())
		}
	}
}

func TestRevokeWebhookSubscriptionEndsTheSubscription(t *testing.T) {
	handler := webhookTestHandler(t)
	created := createTestWebhookSubscription(t, handler, `{"url":"https://receiver.example.com/hooks","kinds":["task_opened"]}`)

	revokeRequest := httptest.NewRequest(http.MethodDelete, "/api/webhook-subscriptions/"+created.Subscription.ID, nil)
	revokeRequest.Header.Set("Authorization", "Bearer test-access-token")
	revokeResponse := httptest.NewRecorder()
	handler.ServeHTTP(revokeResponse, revokeRequest)
	if revokeResponse.Code != http.StatusOK {
		t.Fatalf("revoke status = %d (%s)", revokeResponse.Code, revokeResponse.Body.String())
	}
	var revoked decodedWebhookSubscription
	if err := json.Unmarshal(revokeResponse.Body.Bytes(), &revoked); err != nil {
		t.Fatalf("decode revoke response: %v", err)
	}
	if revoked.State != "revoked" {
		t.Fatalf("revoked state = %q, want revoked", revoked.State)
	}

	// Revoking twice fails: the state transition only applies to an active
	// subscription.
	secondResponse := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodDelete, "/api/webhook-subscriptions/"+created.Subscription.ID, nil)
	secondRequest.Header.Set("Authorization", "Bearer test-access-token")
	handler.ServeHTTP(secondResponse, secondRequest)
	if secondResponse.Code != http.StatusBadRequest {
		t.Fatalf("second revoke status = %d, want %d", secondResponse.Code, http.StatusBadRequest)
	}
}

func TestListWebhookDeliveriesForOwnSubscription(t *testing.T) {
	handler := webhookTestHandler(t)
	created := createTestWebhookSubscription(t, handler, `{"url":"https://receiver.example.com/hooks","kinds":["task_opened"]}`)

	request := httptest.NewRequest(http.MethodGet, "/api/webhook-subscriptions/"+created.Subscription.ID+"/deliveries", nil)
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("deliveries status = %d (%s)", response.Code, response.Body.String())
	}
	if response.Body.String() != "{\"deliveries\":[],\"next_offset\":0,\"total\":0}\n" {
		t.Fatalf("deliveries body = %q", response.Body.String())
	}
}

func TestWebhookSubscriptionRoutesRequireAuth(t *testing.T) {
	handler := webhookTestHandler(t)

	for _, route := range []struct {
		method string
		target string
	}{
		{http.MethodPost, "/api/webhook-subscriptions"},
		{http.MethodGet, "/api/webhook-subscriptions"},
		{http.MethodDelete, "/api/webhook-subscriptions/missing-id"},
		{http.MethodGet, "/api/webhook-subscriptions/missing-id/deliveries"},
	} {
		request := httptest.NewRequest(route.method, route.target, strings.NewReader(`{}`))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, want %d", route.method, route.target, response.Code, http.StatusUnauthorized)
		}
	}
}

func TestUserCreatesOrganizationOwnedWebhookSubscription(t *testing.T) {
	handler := webhookTestHandler(t)
	organizationID := core.NewOrganizationID().(core.OrganizationIDCreated).Value

	created := createTestWebhookSubscription(t, handler, `{"url":"https://receiver.example.com/hooks","kinds":["task_opened"],"organization_id":"`+organizationID.String()+`"}`)
	if created.Subscription.OwnerKind != "organization" || created.Subscription.OwnerOrganizationID != organizationID.String() {
		t.Fatalf("owner = %+v, want the named organization", created.Subscription)
	}
}

// ---- organization credential callers ----

// orgTokenRejectingVerifier refuses bearer tokens carrying the org
// credential prefix, matching production where an org secret is not a JWT,
// so requests fall through to the org credential path.
type orgTokenRejectingVerifier struct{}

func (orgTokenRejectingVerifier) Verify(token auth.AccessToken) auth.SubjectVerifyResult {
	if orgcred.HasSecretPrefix(token.String()) {
		return auth.SubjectVerifyRejected{Reason: core.NewDomainError(core.ErrorCodeUnauthenticated, "not a user access token")}
	}
	return auth.SubjectVerified{Value: auth.UserSubject{ID: stableTestUserID}}
}

// stubOrgCredentialService verifies every org secret as one fixed credential.
type stubOrgCredentialService struct {
	credential orgcred.Credential
}

func (service stubOrgCredentialService) Create(context.Context, core.OrganizationID, agent.Label, agent.ScopeSet, *time.Time) orgcred.CreateResult {
	return orgcred.CreateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (service stubOrgCredentialService) Verify(context.Context, orgcred.SecretPlain) orgcred.VerifyResult {
	return orgcred.CredentialVerified{Subject: auth.OrgSubject{ID: service.credential.OrganizationID}, Credential: service.credential}
}

func (service stubOrgCredentialService) List(context.Context, core.OrganizationID, core.Page) orgcred.ListResult {
	return orgcred.CredentialsListed{Values: []orgcred.Credential{}}
}

func (service stubOrgCredentialService) Revoke(context.Context, core.OrganizationID, core.OrgCredentialID) orgcred.RevokeResult {
	return orgcred.RevokeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func webhookOrgCredentialHandler(t *testing.T, scopes []agent.Scope) (http.Handler, core.OrganizationID, string) {
	t.Helper()
	organizationID := core.NewOrganizationID().(core.OrganizationIDCreated).Value
	credential := orgcred.Credential{
		ID:             core.NewOrgCredentialID().(core.OrgCredentialIDCreated).Value,
		OrganizationID: organizationID,
		Label:          agent.NewLabel("Webhook automation").(agent.LabelAccepted).Value,
		Scopes:         agent.NewScopeSet(scopes),
		State:          agent.StateActive,
	}
	secret := orgcred.NewSecretPlain().(orgcred.SecretPlainAccepted).Value
	runtime := DefaultRuntimeState(map[string]bool{})
	handler := NewWithRuntimeState(testStaticFiles(), testAuthService(), orgTokenRejectingVerifier{}, testOrganizationService{}, testTaskService{}, testSubmissionService{}, testLedgerService{}, testAgentService{}, stubOrgCredentialService{credential: credential}, testAssetService{}, runtime)
	return handler, organizationID, secret.String()
}

func TestOrgCredentialCreatesOrganizationWebhookSubscription(t *testing.T) {
	handler, organizationID, secret := webhookOrgCredentialHandler(t, []agent.Scope{agent.ScopeWebhooksManage, agent.ScopeTasksRead})

	request := httptest.NewRequest(http.MethodPost, "/api/webhook-subscriptions", strings.NewReader(`{"url":"https://receiver.example.com/hooks","kinds":["task_opened","reservation_requested"]}`))
	request.Header.Set("Authorization", "Bearer "+secret)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusCreated, response.Body.String())
	}
	var created decodedWebhookCreated
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.Subscription.OwnerKind != "organization" || created.Subscription.OwnerOrganizationID != organizationID.String() {
		t.Fatalf("owner = %+v, want the credential's organization", created.Subscription)
	}
}

func TestOrgCredentialWebhookCreationRequiresManageScope(t *testing.T) {
	handler, _, secret := webhookOrgCredentialHandler(t, []agent.Scope{agent.ScopeWebhooksRead, agent.ScopeTasksRead})

	request := httptest.NewRequest(http.MethodPost, "/api/webhook-subscriptions", strings.NewReader(`{"url":"https://receiver.example.com/hooks","kinds":["task_opened"]}`))
	request.Header.Set("Authorization", "Bearer "+secret)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusForbidden, response.Body.String())
	}
}

func TestOrgCredentialWebhookCreationRequiresKindEntitlement(t *testing.T) {
	// webhooks_manage alone must not widen the credential: subscribing to
	// ledger-bearing kinds needs ledger_read.
	handler, _, secret := webhookOrgCredentialHandler(t, []agent.Scope{agent.ScopeWebhooksManage, agent.ScopeTasksRead})

	request := httptest.NewRequest(http.MethodPost, "/api/webhook-subscriptions", strings.NewReader(`{"url":"https://receiver.example.com/hooks","kinds":["task_opened","payout_received"]}`))
	request.Header.Set("Authorization", "Bearer "+secret)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (%s)", response.Code, http.StatusForbidden, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), agent.ScopeLedgerRead.String()) {
		t.Fatalf("rejection should name the missing scope: %s", response.Body.String())
	}
}

func TestOrgCredentialCannotManageAnotherOrganizationsWebhooks(t *testing.T) {
	handler, _, secret := webhookOrgCredentialHandler(t, []agent.Scope{agent.ScopeWebhooksManage, agent.ScopeWebhooksRead, agent.ScopeTasksRead})
	otherOrganization := core.NewOrganizationID().(core.OrganizationIDCreated).Value

	createRequest := httptest.NewRequest(http.MethodPost, "/api/webhook-subscriptions", strings.NewReader(`{"url":"https://receiver.example.com/hooks","kinds":["task_opened"],"organization_id":"`+otherOrganization.String()+`"}`))
	createRequest.Header.Set("Authorization", "Bearer "+secret)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusForbidden {
		t.Fatalf("create status = %d, want %d (%s)", createResponse.Code, http.StatusForbidden, createResponse.Body.String())
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/webhook-subscriptions?organization_id="+otherOrganization.String(), nil)
	listRequest.Header.Set("Authorization", "Bearer "+secret)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusForbidden {
		t.Fatalf("list status = %d, want %d (%s)", listResponse.Code, http.StatusForbidden, listResponse.Body.String())
	}
}

// The webhook wire payload (shared by the feed and delivery bodies) is
// pinned in contract_fixture_test.go; this test pins the management DTOs.
func TestWebhookSubscriptionResponseCarriesEveryKind(t *testing.T) {
	handler := webhookTestHandler(t)
	kinds := `["task_opened","task_funded","submission_created","payout_received","collectible_awarded"]`
	created := createTestWebhookSubscription(t, handler, `{"url":"https://receiver.example.com/hooks","kinds":`+kinds+`}`)
	if len(created.Subscription.Kinds) != 5 {
		t.Fatalf("kinds = %+v, want all five", created.Subscription.Kinds)
	}
	if _, matched := webhook.NewEndpointURL(created.Subscription.URL).(webhook.EndpointURLAccepted); !matched {
		t.Fatalf("returned url does not round-trip: %q", created.Subscription.URL)
	}
}
