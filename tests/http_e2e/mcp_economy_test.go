//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// toolFailureText returns the single text item of an isError tool result,
// failing the test when the call was a protocol error or a success.
func toolFailureText(t *testing.T, envelope rpcEnvelope) string {
	t.Helper()
	if envelope.Error != nil {
		t.Fatalf("expected a tool-level failure, got protocol error: %+v", envelope.Error)
	}
	var result struct {
		IsError bool `json:"isError"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if !result.IsError || len(result.Content) != 1 {
		t.Fatalf("expected an isError tool result with one item, got %+v", result)
	}
	return result.Content[0].Text
}

// TestMCPSendCreditsWithIdempotentReplay covers the peer credit send over
// MCP: a ledger_write-scoped credential moves credits once, a replayed
// idempotency key returns the original transfer without double-sending, and
// a credential without the scope neither sees nor may call the tool.
func TestMCPSendCreditsWithIdempotentReplay(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	sender := registerUser(t, server, "mcp-send-sender")
	receiver := registerUser(t, server, "mcp-send-receiver")

	senderAgent := createAgentCredential(t, server, sender.AccessToken, []string{"ledger_read", "ledger_write"})
	session := initializeMCPSession(t, server, senderAgent)

	arguments := `{"source_kind":"self","target_kind":"user","target_id":"` + receiver.SubjectID + `","amount":25,"note":"Great pairing session","idempotency_key":"mcp-send-1"}`
	sent := toolText(t, decodeRPC(t, mcpCall(t, server, senderAgent, session, `1`, "sharecrop.send_credits", arguments)))
	var payload struct {
		EntryID string `json:"entry_id"`
		Amount  int64  `json:"amount"`
	}
	if err := json.Unmarshal([]byte(sent), &payload); err != nil {
		t.Fatalf("decode send payload: %v (%s)", err, sent)
	}
	if payload.EntryID == "" || payload.Amount != 25 {
		t.Fatalf("send payload = %+v", payload)
	}

	// Replaying the same idempotency key returns the original entry and
	// moves no further credits.
	replayed := toolText(t, decodeRPC(t, mcpCall(t, server, senderAgent, session, `2`, "sharecrop.send_credits", arguments)))
	var replay struct {
		EntryID string `json:"entry_id"`
		Amount  int64  `json:"amount"`
	}
	if err := json.Unmarshal([]byte(replayed), &replay); err != nil {
		t.Fatalf("decode replay payload: %v (%s)", err, replayed)
	}
	if replay.EntryID != payload.EntryID || replay.Amount != 25 {
		t.Fatalf("replay = %+v, want the original entry %s", replay, payload.EntryID)
	}
	if balance := getBalance(t, server, sender.AccessToken); balance.SpendableCredits != 75 {
		t.Fatalf("sender balance = %d, want 75 after one send of 25", balance.SpendableCredits)
	}
	if balance := getBalance(t, server, receiver.AccessToken); balance.SpendableCredits != 125 {
		t.Fatalf("receiver balance = %d, want 125 after one send of 25", balance.SpendableCredits)
	}

	// A credential without ledger_write does not see send_credits in
	// tools/list, and calling it anyway fails the scope gate — so existing
	// credentials must be re-minted with the new scope to send.
	readOnlyAgent := createAgentCredential(t, server, sender.AccessToken, []string{"ledger_read"})
	readOnlySession := initializeMCPSession(t, server, readOnlyAgent)
	toolsList := decodeRPC(t, mcpRequest(t, server, readOnlyAgent, readOnlySession, `3`, "tools/list", `{}`))
	if strings.Contains(string(toolsList.Result), "sharecrop.send_credits") {
		t.Fatalf("ledger_read-only credential should not see send_credits")
	}
	denied := decodeRPC(t, mcpCall(t, server, readOnlyAgent, readOnlySession, `4`, "sharecrop.send_credits", arguments))
	if denied.Error == nil || !strings.Contains(denied.Error.Message, "ledger_write") {
		t.Fatalf("expected a ledger_write scope denial, got %+v", denied.Error)
	}
}

// TestMCPOrgCredentialReviewsSubmissions covers the org-credential review
// widening over MCP: an organization credential lists and reads its own org
// task's submissions and accepts one, while tip parameters are refused with
// a clear message (tips move personal value).
func TestMCPOrgCredentialReviewsSubmissions(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-org-review-owner")
	worker := registerUser(t, server, "mcp-org-review-worker")
	organizationID := createOrganization(t, server, owner, "MCP Review Org")
	credential := mintOrgCredential(t, server, owner.AccessToken, organizationID, `["submissions_read","submissions_review"]`)

	createTaskResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(organizationPublicTaskRequestJSON(organizationID)), owner.AccessToken)
	defer createTaskResponse.Body.Close()
	task := decodeTaskHTTPResponse(t, createTaskResponse)
	openTask(t, server, owner.AccessToken, task.ID)
	submitted := submitAuthenticated(t, server, worker.AccessToken, task.ID)

	session := initializeMCPSession(t, server, credential.Secret)

	listed := toolText(t, decodeRPC(t, mcpCall(t, server, credential.Secret, session, `1`, "sharecrop.list_task_submissions", `{"task_id":"`+task.ID+`"}`)))
	if !strings.Contains(listed, submitted.Submission.ID) {
		t.Fatalf("org credential listing missing submission: %s", listed)
	}
	detail := toolText(t, decodeRPC(t, mcpCall(t, server, credential.Secret, session, `2`, "sharecrop.get_submission", `{"submission_id":"`+submitted.Submission.ID+`"}`)))
	if !strings.Contains(detail, "response_json") {
		t.Fatalf("org credential get_submission missing content: %s", detail)
	}

	// Tip parameters act as a person and are refused for the org credential
	// with a clear message.
	tipRefused := toolFailureText(t, decodeRPC(t, mcpCall(t, server, credential.Secret, session, `3`, "sharecrop.accept_submission",
		`{"task_id":"`+task.ID+`","submission_id":"`+submitted.Submission.ID+`","idempotency_key":"org-tip-`+task.ID+`","tip_amount":5}`)))
	if !strings.Contains(tipRefused, "organization credential cannot pay a credit tip") {
		t.Fatalf("tip refusal message = %s", tipRefused)
	}

	accepted := toolText(t, decodeRPC(t, mcpCall(t, server, credential.Secret, session, `4`, "sharecrop.accept_submission",
		`{"task_id":"`+task.ID+`","submission_id":"`+submitted.Submission.ID+`","idempotency_key":"org-accept-`+task.ID+`"}`)))
	if !strings.Contains(accepted, submitted.Submission.ID) {
		t.Fatalf("org credential accept payload missing submission: %s", accepted)
	}
}

// TestMCPCatalogLifecycle covers the platform catalog over MCP end to end:
// an admin adds an entry, awards a numbered copy with provenance, the
// catalog lists state and counts, withdrawal stops awarding, and the
// withdrawn instance and entry can be deleted. A platform_admin-scoped
// credential of a non-admin user is refused by the live re-check.
func TestMCPCatalogLifecycle(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "mcp-catalog-admin")
	holder := registerUser(t, bootstrap, "mcp-catalog-holder")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	adminAgent := createAgentCredential(t, server, admin.AccessToken, []string{"platform_admin", "collectibles_read", "collectibles_manage"})
	session := initializeMCPSession(t, server, adminAgent)

	added := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `1`, "sharecrop.add_catalog_entry",
		`{"slug":"mcp-harvest-star","name":"Harvest Star","kind":"edition","transfer_policy":"transferable_between_users","art":"seedling","max_editions":2}`)))
	if !strings.Contains(added, `"state":"available"`) || !strings.Contains(added, `"max_editions":2`) {
		t.Fatalf("add_catalog_entry payload = %s", added)
	}

	awarded := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `2`, "sharecrop.award_collectible",
		`{"slug":"mcp-harvest-star","recipient_id":"`+holder.SubjectID+`"}`)))
	var instance struct {
		ID                string `json:"id"`
		CatalogSlug       string `json:"catalog_slug"`
		EditionNumber     int64  `json:"edition_number"`
		IssuerDisplayName string `json:"issuer_display_name"`
	}
	if err := json.Unmarshal([]byte(awarded), &instance); err != nil {
		t.Fatalf("decode award payload: %v (%s)", err, awarded)
	}
	if instance.CatalogSlug != "mcp-harvest-star" || instance.EditionNumber != 1 {
		t.Fatalf("award provenance = %+v", instance)
	}

	catalog := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `3`, "sharecrop.collectible_catalog", `{}`)))
	var listing struct {
		Entries []struct {
			Slug        string `json:"slug"`
			State       string `json:"state"`
			MaxEditions int64  `json:"max_editions"`
			MintedCount int64  `json:"minted_count"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(catalog), &listing); err != nil {
		t.Fatalf("decode catalog payload: %v (%s)", err, catalog)
	}
	found := false
	for _, entry := range listing.Entries {
		if entry.Slug == "mcp-harvest-star" {
			found = true
			if entry.State != "available" || entry.MaxEditions != 2 || entry.MintedCount != 1 {
				t.Fatalf("catalog entry = %+v", entry)
			}
		}
	}
	if !found {
		t.Fatalf("catalog missing mcp-harvest-star: %s", catalog)
	}

	// Withdraw the entry: it stays listed with the withdrawn state, and
	// awarding from it is refused.
	withdrawnEntry := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `4`, "sharecrop.withdraw_catalog_entry", `{"slug":"mcp-harvest-star"}`)))
	if !strings.Contains(withdrawnEntry, `"state":"withdrawn"`) {
		t.Fatalf("withdraw_catalog_entry payload = %s", withdrawnEntry)
	}
	refused := toolFailureText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `5`, "sharecrop.award_collectible",
		`{"slug":"mcp-harvest-star","recipient_id":"`+holder.SubjectID+`"}`)))
	if refused == "" {
		t.Fatalf("expected awarding from a withdrawn entry to fail")
	}

	// Withdraw the awarded instance, then hard-delete instance and entry.
	withdrawnInstance := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `6`, "sharecrop.withdraw_collectible", `{"collectible_id":"`+instance.ID+`"}`)))
	if !strings.Contains(withdrawnInstance, `"state":"withdrawn"`) {
		t.Fatalf("withdraw_collectible payload = %s", withdrawnInstance)
	}
	deletedInstance := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `7`, "sharecrop.delete_withdrawn_collectible", `{"collectible_id":"`+instance.ID+`"}`)))
	if !strings.Contains(deletedInstance, `"status":"deleted"`) {
		t.Fatalf("delete_withdrawn_collectible payload = %s", deletedInstance)
	}
	deletedEntry := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `8`, "sharecrop.delete_catalog_entry", `{"slug":"mcp-harvest-star"}`)))
	if !strings.Contains(deletedEntry, `"status":"deleted"`) {
		t.Fatalf("delete_catalog_entry payload = %s", deletedEntry)
	}

	// The live admin re-check refuses a scoped-but-demoted credential.
	holderAgent := createAgentCredential(t, server, holder.AccessToken, []string{"platform_admin"})
	holderSession := initializeMCPSession(t, server, holderAgent)
	deniedText := toolFailureText(t, decodeRPC(t, mcpCall(t, server, holderAgent, holderSession, `9`, "sharecrop.add_catalog_entry",
		`{"slug":"mcp-denied","name":"Denied","kind":"badge","transfer_policy":"transferable_between_users","art":"seedling"}`)))
	if !strings.Contains(deniedText, "platform admin access is required") {
		t.Fatalf("non-admin catalog mutation message = %s", deniedText)
	}
}

// TestMCPTransferOrgCollectiblePaths covers the organization collectible
// transfers over MCP: a user donates one to the organization with
// target_kind organization, a member with manage_collectibles moves it back
// out, and an organization credential transfers one to an active member.
func TestMCPTransferOrgCollectiblePaths(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-orgcoll-owner")
	outsider := registerUser(t, server, "mcp-orgcoll-outsider")
	organizationID := createOrganization(t, server, owner, "MCP Collectible Org")

	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"collectibles_read", "collectibles_manage"})
	session := initializeMCPSession(t, server, ownerAgent)

	// Mint and donate to the organization's trophy case.
	minted := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `1`, "sharecrop.mint_collectible",
		`{"name":"Traveling Trophy","kind":"badge","transfer_policy":"transferable_between_users"}`)))
	var collectible struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(minted), &collectible); err != nil {
		t.Fatalf("decode mint payload: %v (%s)", err, minted)
	}
	donated := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `2`, "sharecrop.transfer_collectible",
		`{"collectible_id":"`+collectible.ID+`","target_kind":"organization","recipient_id":"`+organizationID+`"}`)))
	if !strings.Contains(donated, `"owner_kind":"organization"`) {
		t.Fatalf("donation payload = %s", donated)
	}

	// The org owner (manage_collectibles) moves it out to a non-member user.
	transferred := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `3`, "sharecrop.transfer_org_collectible",
		`{"organization_id":"`+organizationID+`","collectible_id":"`+collectible.ID+`","recipient_id":"`+outsider.SubjectID+`"}`)))
	if !strings.Contains(transferred, `"owner_id":"`+outsider.SubjectID+`"`) {
		t.Fatalf("member-path transfer payload = %s", transferred)
	}

	// Donate a second collectible, then transfer it out with an org-wide
	// credential; the recipient must be an active member for that path.
	secondMint := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `4`, "sharecrop.mint_collectible",
		`{"name":"Member Medal","kind":"badge","transfer_policy":"transferable_between_users"}`)))
	var second struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(secondMint), &second); err != nil {
		t.Fatalf("decode second mint payload: %v (%s)", err, secondMint)
	}
	if text := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `5`, "sharecrop.transfer_collectible",
		`{"collectible_id":"`+second.ID+`","target_kind":"organization","recipient_id":"`+organizationID+`"}`))); !strings.Contains(text, `"owner_kind":"organization"`) {
		t.Fatalf("second donation payload = %s", text)
	}

	orgCredential := mintOrgCredential(t, server, owner.AccessToken, organizationID, `["collectibles_read","collectibles_manage"]`)
	orgSession := initializeMCPSession(t, server, orgCredential.Secret)

	// A non-member recipient is refused on the org-credential path.
	nonMember := toolFailureText(t, decodeRPC(t, mcpCall(t, server, orgCredential.Secret, orgSession, `6`, "sharecrop.transfer_org_collectible",
		`{"collectible_id":"`+second.ID+`","recipient_id":"`+outsider.SubjectID+`"}`)))
	if !strings.Contains(nonMember, "active member") {
		t.Fatalf("non-member refusal = %s", nonMember)
	}

	viaOrg := toolText(t, decodeRPC(t, mcpCall(t, server, orgCredential.Secret, orgSession, `7`, "sharecrop.transfer_org_collectible",
		`{"collectible_id":"`+second.ID+`","recipient_id":"`+owner.SubjectID+`"}`)))
	if !strings.Contains(viaOrg, `"owner_id":"`+owner.SubjectID+`"`) {
		t.Fatalf("org-credential transfer payload = %s", viaOrg)
	}
}

// TestMCPMalformedIDMessage pins the sweep finding end to end: a malformed
// id argument returns the uniform domain-shaped invalid-params message, not
// the raw UUID library error.
func TestMCPMalformedIDMessage(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	user := registerUser(t, server, "mcp-bad-id")
	agentToken := createAgentCredential(t, server, user.AccessToken, []string{"webhooks_read"})
	session := initializeMCPSession(t, server, agentToken)

	response := decodeRPC(t, mcpCall(t, server, agentToken, session, `1`, "sharecrop.list_webhook_deliveries", `{"subscription_id":"abcd"}`))
	if response.Error == nil {
		t.Fatalf("expected an invalid-params protocol error")
	}
	if !strings.Contains(response.Error.Message, "subscription_id must be a valid id") {
		t.Fatalf("message = %q, want the uniform id message", response.Error.Message)
	}
	if strings.Contains(response.Error.Message, "invalid UUID") {
		t.Fatalf("raw UUID library error leaked: %q", response.Error.Message)
	}
}
