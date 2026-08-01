package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
)

func testOrgSubject(t *testing.T) auth.OrgSubject {
	t.Helper()
	created, matched := core.NewOrganizationID().(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id rejected")
	}
	return auth.OrgSubject{ID: created.Value}
}

func ledgerWriteScopes() agent.ScopeSet {
	return agent.NewScopeSet([]agent.Scope{agent.ScopeLedgerRead, agent.ScopeLedgerWrite})
}

// TestSendCreditsThreadsArguments pins the send_credits boundary: the
// source/target/amount/note/idempotency arguments map onto the ledger
// service's transfer values exactly like REST's POST /api/credits/transfers.
func TestSendCreditsThreadsArguments(t *testing.T) {
	services := newCapturingServices()
	server := NewServer(services)
	targetUserID := core.NewUserID().(core.UserIDCreated).Value
	organizationID := testOrganizationID(t)

	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: ledgerWriteScopes()}, request(`1`, "tools/call",
		`{"name":"sharecrop.send_credits","arguments":{"source_kind":"self","target_kind":"user","target_id":"`+targetUserID.String()+`","amount":7,"note":"Thanks for the review","idempotency_key":"send-1"}}`)))
	var payload creditTransferPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode transfer payload: %v (%s)", err, content)
	}
	if payload.EntryID == "" || payload.Amount != 7 {
		t.Fatalf("transfer payload = %+v", payload)
	}
	if _, matched := services.sendCommand.source.(ledger.TransferFromSelf); !matched {
		t.Fatalf("source = %#v, want TransferFromSelf", services.sendCommand.source)
	}
	target, targetMatched := services.sendCommand.target.(ledger.TransferToUser)
	if !targetMatched || target.ID != targetUserID {
		t.Fatalf("target = %#v, want TransferToUser %s", services.sendCommand.target, targetUserID)
	}
	if services.sendCommand.amount != 7 || services.sendCommand.key != "send-1" {
		t.Fatalf("amount/key = %d/%q", services.sendCommand.amount, services.sendCommand.key)
	}
	if _, matched := services.sendCommand.note.(ledger.TransferNoteProvided); !matched {
		t.Fatalf("note = %#v, want TransferNoteProvided", services.sendCommand.note)
	}

	// An organization source and organization target thread through, and an
	// absent note is explicitly no note.
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: ledgerWriteScopes()}, request(`2`, "tools/call",
		`{"name":"sharecrop.send_credits","arguments":{"source_kind":"organization","source_organization_id":"`+organizationID+`","target_kind":"organization","target_id":"`+organizationID+`","amount":3,"idempotency_key":"send-2"}}`))
	if response.Error != nil {
		t.Fatalf("organization send error: %s", response.Error.Message)
	}
	if _, matched := services.sendCommand.source.(ledger.TransferFromOrganization); !matched {
		t.Fatalf("source = %#v, want TransferFromOrganization", services.sendCommand.source)
	}
	if _, matched := services.sendCommand.target.(ledger.TransferToOrganization); !matched {
		t.Fatalf("target = %#v, want TransferToOrganization", services.sendCommand.target)
	}
	if _, matched := services.sendCommand.note.(ledger.NoTransferNote); !matched {
		t.Fatalf("note = %#v, want NoTransferNote", services.sendCommand.note)
	}
}

// TestSendCreditsInputShapeViolations pins the invalid-params layer for the
// send tool: unknown enum values and malformed ids are JSON-RPC protocol
// errors, phrased in the uniform "<argument> must be ..." shape.
func TestSendCreditsInputShapeViolations(t *testing.T) {
	server := NewServer(fakeServices{})
	targetUserID := core.NewUserID().(core.UserIDCreated).Value.String()

	badSource := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: ledgerWriteScopes()}, request(`1`, "tools/call",
		`{"name":"sharecrop.send_credits","arguments":{"source_kind":"treasury","target_kind":"user","target_id":"`+targetUserID+`","amount":1,"idempotency_key":"k"}}`))
	if badSource.Error == nil || badSource.Error.Code != codeInvalidParams || !strings.Contains(badSource.Error.Message, "source_kind must be self or organization") {
		t.Fatalf("bad source_kind = %+v", badSource.Error)
	}

	badTarget := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: ledgerWriteScopes()}, request(`2`, "tools/call",
		`{"name":"sharecrop.send_credits","arguments":{"source_kind":"self","target_kind":"user","target_id":"abcd","amount":1,"idempotency_key":"k"}}`))
	if badTarget.Error == nil || badTarget.Error.Code != codeInvalidParams || !strings.Contains(badTarget.Error.Message, "target_id must be a valid id") {
		t.Fatalf("bad target_id = %+v", badTarget.Error)
	}
}

// TestSendCreditsScopeAndSubjectGates pins who can send: only a
// ledger_write-scoped credential sees or calls the tool, and only a personal
// agent credential (the transfer needs an acting user) may use it.
func TestSendCreditsScopeAndSubjectGates(t *testing.T) {
	server := NewServer(fakeServices{})

	// tools/list: shown only with ledger_write.
	granted := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: ledgerWriteScopes()}, request(`1`, "tools/list", `{}`))
	if !strings.Contains(string(mustResult(t, granted)), toolSendCredits) {
		t.Fatalf("ledger_write credential missing send_credits in tools/list")
	}
	readOnly := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeLedgerRead})}, request(`2`, "tools/list", `{}`))
	if strings.Contains(string(mustResult(t, readOnly)), toolSendCredits) {
		t.Fatalf("ledger_read-only credential should not see send_credits")
	}

	// Calling without the scope fails the scope gate.
	targetUserID := core.NewUserID().(core.UserIDCreated).Value.String()
	denied := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeLedgerRead})}, request(`3`, "tools/call",
		`{"name":"sharecrop.send_credits","arguments":{"source_kind":"self","target_kind":"user","target_id":"`+targetUserID+`","amount":1,"idempotency_key":"k"}}`))
	if denied.Error == nil || denied.Error.Code != codeScopeDenied {
		t.Fatalf("expected scope-denied without ledger_write, got %+v", denied.Error)
	}

	// An organization credential has no personal balance to send from.
	orgDenied := server.Handle(context.Background(), testOrgSubject(t), CallerCredential{Scopes: ledgerWriteScopes()}, request(`4`, "tools/call",
		`{"name":"sharecrop.send_credits","arguments":{"source_kind":"self","target_kind":"user","target_id":"`+targetUserID+`","amount":1,"idempotency_key":"k"}}`))
	var orgResult toolCallResult
	if err := json.Unmarshal(mustResult(t, orgDenied), &orgResult); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !orgResult.IsError || !strings.Contains(orgResult.Content[0].Text, "personal agent credential") {
		t.Fatalf("expected a personal-credential tool failure, got %+v", orgResult)
	}
}

// TestTransferCollectibleTargetKinds pins the widened transfer tool: the
// default and "user" target gift to a user, "organization" donates to an
// organization's trophy case, and other kinds are input-shape violations.
func TestTransferCollectibleTargetKinds(t *testing.T) {
	server := NewServer(fakeServices{})
	collectibleID := core.NewCollectibleID().(core.CollectibleIDCreated).Value.String()
	recipientID := core.NewUserID().(core.UserIDCreated).Value.String()
	organizationID := testOrganizationID(t)

	toUser := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeCollectiblesManage})}, request(`1`, "tools/call",
		`{"name":"sharecrop.transfer_collectible","arguments":{"collectible_id":"`+collectibleID+`","recipient_id":"`+recipientID+`"}}`)))
	if !strings.Contains(toUser, collectibleID) {
		t.Fatalf("user transfer payload missing collectible: %s", toUser)
	}

	toOrg := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeCollectiblesManage})}, request(`2`, "tools/call",
		`{"name":"sharecrop.transfer_collectible","arguments":{"collectible_id":"`+collectibleID+`","target_kind":"organization","recipient_id":"`+organizationID+`"}}`)))
	if !strings.Contains(toOrg, `"owner_kind":"organization"`) {
		t.Fatalf("organization transfer payload owner kind wrong: %s", toOrg)
	}

	badKind := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeCollectiblesManage})}, request(`3`, "tools/call",
		`{"name":"sharecrop.transfer_collectible","arguments":{"collectible_id":"`+collectibleID+`","target_kind":"team","recipient_id":"`+recipientID+`"}}`))
	if badKind.Error == nil || badKind.Error.Code != codeInvalidParams || !strings.Contains(badKind.Error.Message, "target_kind must be user or organization") {
		t.Fatalf("bad target_kind = %+v", badKind.Error)
	}
}

// TestTransferOrgCollectibleSubjectPaths pins the org-transfer tool: a
// personal credential names the organization and acts as its user, an
// organization credential acts for its own organization only.
func TestTransferOrgCollectibleSubjectPaths(t *testing.T) {
	server := NewServer(fakeServices{})
	scopes := agent.NewScopeSet([]agent.Scope{agent.ScopeCollectiblesManage})
	collectibleID := core.NewCollectibleID().(core.CollectibleIDCreated).Value.String()
	recipientID := core.NewUserID().(core.UserIDCreated).Value.String()
	organizationID := testOrganizationID(t)

	asUser := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: scopes}, request(`1`, "tools/call",
		`{"name":"sharecrop.transfer_org_collectible","arguments":{"organization_id":"`+organizationID+`","collectible_id":"`+collectibleID+`","recipient_id":"`+recipientID+`"}}`)))
	if !strings.Contains(asUser, `"owner_id":"`+recipientID+`"`) {
		t.Fatalf("member transfer payload owner wrong: %s", asUser)
	}

	orgSubject := testOrgSubject(t)
	asOrg := decodeToolText(t, server.Handle(context.Background(), orgSubject, CallerCredential{Scopes: scopes}, request(`2`, "tools/call",
		`{"name":"sharecrop.transfer_org_collectible","arguments":{"collectible_id":"`+collectibleID+`","recipient_id":"`+recipientID+`"}}`)))
	if !strings.Contains(asOrg, `"owner_id":"`+recipientID+`"`) {
		t.Fatalf("org-credential transfer payload owner wrong: %s", asOrg)
	}

	// An organization credential cannot act for a different organization.
	foreign := server.Handle(context.Background(), orgSubject, CallerCredential{Scopes: scopes}, request(`3`, "tools/call",
		`{"name":"sharecrop.transfer_org_collectible","arguments":{"organization_id":"`+testOrganizationID(t)+`","collectible_id":"`+collectibleID+`","recipient_id":"`+recipientID+`"}}`))
	var foreignResult toolCallResult
	if err := json.Unmarshal(mustResult(t, foreign), &foreignResult); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !foreignResult.IsError || !strings.Contains(foreignResult.Content[0].Text, "own organization") {
		t.Fatalf("expected an own-organization tool failure, got %+v", foreignResult)
	}
}

// TestCatalogAdminToolsRequireLiveAdmin pins the admin gate on every catalog
// mutation tool: a platform_admin-scoped credential whose user is not
// currently an admin is refused with a tool-level failure, and a live admin
// passes through to the service.
func TestCatalogAdminToolsRequireLiveAdmin(t *testing.T) {
	collectibleID := core.NewCollectibleID().(core.CollectibleIDCreated).Value.String()
	calls := []string{
		`{"name":"sharecrop.add_catalog_entry","arguments":{"slug":"golden-plow","name":"Golden Plow","kind":"edition","transfer_policy":"transferable_between_users","art":"seedling","max_editions":10}}`,
		`{"name":"sharecrop.withdraw_catalog_entry","arguments":{"slug":"golden-plow"}}`,
		`{"name":"sharecrop.delete_catalog_entry","arguments":{"slug":"golden-plow"}}`,
		`{"name":"sharecrop.withdraw_collectible","arguments":{"collectible_id":"` + collectibleID + `"}}`,
		`{"name":"sharecrop.delete_withdrawn_collectible","arguments":{"collectible_id":"` + collectibleID + `"}}`,
	}

	demoted := NewServer(fakeServices{isAdmin: false})
	adminScopes := agent.NewScopeSet([]agent.Scope{agent.ScopePlatformAdmin})
	for _, call := range calls {
		response := demoted.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: adminScopes}, request(`1`, "tools/call", call))
		var result toolCallResult
		if err := json.Unmarshal(mustResult(t, response), &result); err != nil {
			t.Fatalf("decode result: %v", err)
		}
		if !result.IsError || !strings.Contains(result.Content[0].Text, "platform admin access is required") {
			t.Fatalf("expected the live admin re-check for %s, got %+v", call, result)
		}
	}

	admin := NewServer(fakeServices{isAdmin: true})
	added := decodeToolText(t, admin.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: adminScopes}, request(`2`, "tools/call", calls[0])))
	var entry catalogEntryPayload
	if err := json.Unmarshal([]byte(added), &entry); err != nil {
		t.Fatalf("decode entry payload: %v (%s)", err, added)
	}
	if entry.Slug != "golden-plow" || entry.State != "available" || entry.MaxEditions != 10 {
		t.Fatalf("added entry = %+v", entry)
	}
	deleted := decodeToolText(t, admin.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: adminScopes}, request(`3`, "tools/call", calls[4])))
	if !strings.Contains(deleted, `"status":"deleted"`) {
		t.Fatalf("delete payload = %s", deleted)
	}
}

// TestCollectibleCatalogListsEveryEntryWithState pins the honest catalog
// read: withdrawn entries stay listed with an explicit state marker, and
// every row carries max_editions and minted_count.
func TestCollectibleCatalogListsEveryEntryWithState(t *testing.T) {
	available := assets.CatalogEntry{
		Slug:   assets.NewCatalogSlug("harvest-star").(assets.CatalogSlugAccepted).Value,
		Name:   assets.NewCollectibleName("Harvest Star").(assets.CollectibleNameAccepted).Value,
		Kind:   assets.ParseCollectibleKind("edition").(assets.CollectibleKindAccepted).Value,
		Policy: assets.ParseTransferPolicy("transferable_between_users").(assets.TransferPolicyAccepted).Value,
		State:  assets.CatalogEntryStateAvailable,
		Cap:    assets.EditionCapOf{Limit: 50},
	}
	withdrawn := available
	withdrawn.Slug = assets.NewCatalogSlug("retired-rooster").(assets.CatalogSlugAccepted).Value
	withdrawn.State = assets.CatalogEntryStateWithdrawn
	server := NewServer(fakeServices{catalogListings: []assets.CatalogListing{
		{Entry: available, LiveInstanceCount: 7},
		{Entry: withdrawn, LiveInstanceCount: 2},
	}})

	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeCollectiblesRead})}, request(`1`, "tools/call", `{"name":"sharecrop.collectible_catalog","arguments":{}}`)))
	var payload catalogPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode catalog payload: %v (%s)", err, content)
	}
	if len(payload.Entries) != 2 {
		t.Fatalf("entries = %d, want both (including the withdrawn one)", len(payload.Entries))
	}
	if payload.Entries[0].State != "available" || payload.Entries[0].MaxEditions != 50 || payload.Entries[0].MintedCount != 7 {
		t.Fatalf("available entry = %+v", payload.Entries[0])
	}
	if payload.Entries[1].State != "withdrawn" || payload.Entries[1].MintedCount != 2 {
		t.Fatalf("withdrawn entry = %+v", payload.Entries[1])
	}
}

// TestCollectibleRowsCarryProvenance pins the provenance serialization:
// catalog_slug, edition_number, issuer_display_name, and the withdrawn state
// all reach the tool payload.
func TestCollectibleRowsCarryProvenance(t *testing.T) {
	collectibleID := core.NewCollectibleID().(core.CollectibleIDCreated).Value
	server := NewServer(fakeServices{collectibles: []assets.Collectible{{
		ID:                collectibleID,
		Name:              assets.NewCollectibleName("Harvest Star #3").(assets.CollectibleNameAccepted).Value,
		Kind:              assets.ParseCollectibleKind("edition").(assets.CollectibleKindAccepted).Value,
		State:             assets.CollectibleStateWithdrawn,
		Policy:            assets.ParseTransferPolicy("transferable_between_users").(assets.TransferPolicyAccepted).Value,
		OwnerKind:         assets.CollectibleOwnerKindUser,
		Catalog:           assets.FromCatalog{Slug: assets.NewCatalogSlug("harvest-star").(assets.CatalogSlugAccepted).Value},
		Edition:           assets.EditionNumbered{Number: 3},
		IssuerDisplayName: auth.NewDisplayName("mara").(auth.DisplayNameAccepted).Value,
	}}})

	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeCollectiblesRead})}, request(`1`, "tools/call", `{"name":"sharecrop.list_collectibles","arguments":{}}`)))
	var payload collectiblesListPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode collectibles payload: %v (%s)", err, content)
	}
	if len(payload.Collectibles) != 1 {
		t.Fatalf("collectibles = %d, want 1", len(payload.Collectibles))
	}
	row := payload.Collectibles[0]
	if row.CatalogSlug != "harvest-star" || row.EditionNumber != 3 || row.IssuerDisplayName != "mara" || row.State != "withdrawn" {
		t.Fatalf("provenance row = %+v", row)
	}
}

// TestOrgCredentialReviewsAsOrganizationReviewer pins the reviewer union
// threading: an organization credential's accept reaches the ledger service
// as OrganizationReviewer, a personal credential's as UserReviewer, and the
// read-side review tools accept the organization subject instead of failing
// the personal-credential gate.
func TestOrgCredentialReviewsAsOrganizationReviewer(t *testing.T) {
	services := newCapturingServices()
	server := NewServer(services)
	scopes := agent.NewScopeSet([]agent.Scope{agent.ScopeSubmissionsRead, agent.ScopeSubmissionsReview})
	taskID := testTaskID(t)
	submissionID := core.NewSubmissionID().(core.SubmissionIDCreated).Value.String()
	orgSubject := testOrgSubject(t)

	accept := server.Handle(context.Background(), orgSubject, CallerCredential{Scopes: scopes}, request(`1`, "tools/call",
		`{"name":"sharecrop.accept_submission","arguments":{"task_id":"`+taskID+`","submission_id":"`+submissionID+`","idempotency_key":"org-accept"}}`))
	if accept.Error != nil {
		t.Fatalf("org accept error: %s", accept.Error.Message)
	}
	reviewer, matched := (*services.reviewer).(ledger.OrganizationReviewer)
	if !matched || reviewer.ID != orgSubject.ID {
		t.Fatalf("reviewer = %#v, want OrganizationReviewer %s", *services.reviewer, orgSubject.ID)
	}

	userSubject := testSubject(t)
	userAccept := server.Handle(context.Background(), userSubject, CallerCredential{Scopes: scopes}, request(`2`, "tools/call",
		`{"name":"sharecrop.accept_submission","arguments":{"task_id":"`+taskID+`","submission_id":"`+submissionID+`","idempotency_key":"user-accept"}}`))
	if userAccept.Error != nil {
		t.Fatalf("user accept error: %s", userAccept.Error.Message)
	}
	if _, matched := (*services.reviewer).(ledger.UserReviewer); !matched {
		t.Fatalf("reviewer = %#v, want UserReviewer", *services.reviewer)
	}

	// The review read/settle tools no longer refuse the org subject with the
	// personal-credential message.
	for _, call := range []string{
		`{"name":"sharecrop.list_task_submissions","arguments":{"task_id":"` + taskID + `"}}`,
		`{"name":"sharecrop.get_submission","arguments":{"submission_id":"` + submissionID + `"}}`,
		`{"name":"sharecrop.request_submission_changes","arguments":{"task_id":"` + taskID + `","submission_id":"` + submissionID + `","idempotency_key":"k","review_note":"Please split the patch."}}`,
		`{"name":"sharecrop.reject_submission","arguments":{"task_id":"` + taskID + `","submission_id":"` + submissionID + `","idempotency_key":"k","review_note":"Off-topic submission."}}`,
	} {
		response := server.Handle(context.Background(), orgSubject, CallerCredential{Scopes: scopes}, request(`3`, "tools/call", call))
		text := decodeToolText(t, response)
		if strings.Contains(text, "requires a personal agent credential") {
			t.Fatalf("org subject refused for %s: %s", call, text)
		}
	}
}

// TestNotificationRowsCarryEnrichment pins the read-model enrichment REST
// already serves: actor_display_name and subject_title reach the MCP rows.
func TestNotificationRowsCarryEnrichment(t *testing.T) {
	server := NewServer(fakeServices{})
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeNotificationsRead})}, request(`1`, "tools/call", `{"name":"sharecrop.list_notifications","arguments":{}}`)))
	var payload notificationsPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode notifications payload: %v (%s)", err, content)
	}
	if len(payload.Notifications) != 1 {
		t.Fatalf("notifications = %d, want 1", len(payload.Notifications))
	}
	row := payload.Notifications[0]
	if row.ActorDisplayName != "mara" || row.SubjectTitle != "Review the release" {
		t.Fatalf("notification enrichment = %+v", row)
	}
}

// TestReservationRowsCarryHolderDisplayName pins the reservation-row
// enrichment REST already serves.
func TestReservationRowsCarryHolderDisplayName(t *testing.T) {
	server := NewServer(fakeServices{})
	content := decodeToolText(t, server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_task_reservations","arguments":{"task_id":"`+testTaskID(t)+`"}}`)))
	var payload reservationsPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("decode reservations payload: %v (%s)", err, content)
	}
	if len(payload.Reservations) != 1 || payload.Reservations[0].HolderDisplayName != "ada" {
		t.Fatalf("reservation rows = %+v", payload.Reservations)
	}
}

// TestMalformedIDsReturnDomainShapedMessages sweeps the id-taking tools with
// a malformed id and pins the uniform message shape: the raw UUID library
// error ("invalid UUID length: 4") never reaches an agent.
func TestMalformedIDsReturnDomainShapedMessages(t *testing.T) {
	server := NewServer(fakeServices{isAdmin: true})
	credential := CallerCredential{Scopes: agent.NewScopeSet(agent.AllScopes())}
	cases := map[string]string{
		`{"name":"sharecrop.get_task","arguments":{"task_id":"abcd"}}`:                                                                                  "task_id must be a valid id",
		`{"name":"sharecrop.get_submission","arguments":{"submission_id":"abcd"}}`:                                                                      "submission_id must be a valid id",
		`{"name":"sharecrop.list_webhook_deliveries","arguments":{"subscription_id":"abcd"}}`:                                                           "subscription_id must be a valid id",
		`{"name":"sharecrop.mark_notification_read","arguments":{"notification_id":"abcd"}}`:                                                            "notification_id must be a valid id",
		`{"name":"sharecrop.transfer_collectible","arguments":{"collectible_id":"abcd","recipient_id":"abcd"}}`:                                         "collectible_id must be a valid id",
		`{"name":"sharecrop.get_task_series","arguments":{"series_id":"abcd"}}`:                                                                         "series_id must be a valid id",
		`{"name":"sharecrop.list_organization_members","arguments":{"organization_id":"abcd"}}`:                                                         "organization_id must be a valid id",
		`{"name":"sharecrop.cancel_task_reservation","arguments":{"task_id":"` + testTaskID(t) + `","reservation_id":"abcd"}}`:                          "reservation_id must be a valid id",
		`{"name":"sharecrop.send_credits","arguments":{"source_kind":"self","target_kind":"user","target_id":"abcd","amount":1,"idempotency_key":"k"}}`: "target_id must be a valid id",
	}
	for call, expected := range cases {
		response := server.Handle(context.Background(), testSubject(t), credential, request(`1`, "tools/call", call))
		if response.Error == nil || response.Error.Code != codeInvalidParams {
			t.Fatalf("expected invalid-params for %s, got %+v", call, response.Error)
		}
		if !strings.Contains(response.Error.Message, expected) {
			t.Fatalf("message for %s = %q, want it to contain %q", call, response.Error.Message, expected)
		}
		if strings.Contains(response.Error.Message, "UUID") && !strings.Contains(response.Error.Message, "a UUID)") {
			t.Fatalf("raw UUID library error leaked for %s: %q", call, response.Error.Message)
		}
	}
}
