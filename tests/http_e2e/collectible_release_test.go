//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// hasAuditAction reports whether the platform audit log carries an event with
// the given action and subject id.
func hasAuditAction(t *testing.T, server *httptest.Server, adminToken string, action string, subjectID string) bool {
	t.Helper()
	response := getWithBearer(t, server.URL+"/api/admin/audit-events?action="+action, adminToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var body struct {
		Events []struct {
			Action    string `json:"action"`
			SubjectID string `json:"subject_id"`
		} `json:"events"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode audit events: %v", err)
	}
	for _, event := range body.Events {
		if event.Action == action && event.SubjectID == subjectID {
			return true
		}
	}
	return false
}

// TestAdminCatalogEntryReleaseHTTP drives the entry release surface: a
// withdrawn entry becomes awardable again through POST .../release, releasing
// a not-withdrawn entry conflicts, non-admins are refused, and the action is
// audited.
func TestAdminCatalogEntryReleaseHTTP(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "entry-release-admin")
	holder := registerUser(t, bootstrap, "entry-release-holder")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	slug := "second-season-" + uniqueTestSuffix(t)
	addResponse := postJSONWithBearer(t, server.URL+"/api/admin/collectible-catalog",
		[]byte(`{"slug":"`+slug+`","name":"Second Season","kind":"badge","transfer_policy":"transferable_between_users","art":"pumpkin"}`), admin.AccessToken)
	defer addResponse.Body.Close()
	assertStatus(t, addResponse, http.StatusCreated)

	// Releasing an entry that is not withdrawn conflicts.
	releaseAvailable := postJSONWithBearer(t, server.URL+"/api/admin/collectible-catalog/"+slug+"/release", []byte(`{}`), admin.AccessToken)
	defer releaseAvailable.Body.Close()
	assertStatus(t, releaseAvailable, http.StatusConflict)

	withdrawEntry := postJSONWithBearer(t, server.URL+"/api/admin/collectible-catalog/"+slug+"/withdraw", []byte(`{}`), admin.AccessToken)
	defer withdrawEntry.Body.Close()
	assertStatus(t, withdrawEntry, http.StatusOK)
	refusedAward := awardFromCatalog(t, server, admin.AccessToken, slug, holder.SubjectID)
	defer refusedAward.Body.Close()
	assertStatus(t, refusedAward, http.StatusConflict)

	// A non-admin cannot release the entry.
	forbidden := postJSONWithBearer(t, server.URL+"/api/admin/collectible-catalog/"+slug+"/release", []byte(`{}`), holder.AccessToken)
	defer forbidden.Body.Close()
	assertStatus(t, forbidden, http.StatusForbidden)

	release := postJSONWithBearer(t, server.URL+"/api/admin/collectible-catalog/"+slug+"/release", []byte(`{}`), admin.AccessToken)
	defer release.Body.Close()
	assertStatus(t, release, http.StatusOK)
	var released catalogEntryHTTPResponse
	if err := json.NewDecoder(release.Body).Decode(&released); err != nil {
		t.Fatalf("decode released entry: %v", err)
	}
	if released.State != "available" {
		t.Fatalf("released entry state = %q, want available", released.State)
	}

	// The entry is awardable again.
	award := awardFromCatalog(t, server, admin.AccessToken, slug, holder.SubjectID)
	defer award.Body.Close()
	assertStatus(t, award, http.StatusCreated)

	if !hasAuditAction(t, server, admin.AccessToken, "admin_catalog_entry_released", slug) {
		t.Fatalf("audit log has no admin_catalog_entry_released row for %s", slug)
	}
}

// TestAdminCollectibleReleaseHTTP drives the instance release surface: a
// withdrawn instance returns to its holder (who is notified through
// collectible_released), releasing a live instance conflicts, a unique whose
// slot was re-minted conflicts, non-admins are refused, the action is
// audited, and owner labels are visible in the listing and catalog payloads.
func TestAdminCollectibleReleaseHTTP(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "instance-release-admin")
	holder := registerUser(t, bootstrap, "instance-release-holder")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	slug := "lone-lantern-" + uniqueTestSuffix(t)
	addResponse := postJSONWithBearer(t, server.URL+"/api/admin/collectible-catalog",
		[]byte(`{"slug":"`+slug+`","name":"Lone Lantern","kind":"unique","transfer_policy":"transferable_between_users","art":"full-moon-harvest","max_editions":1}`), admin.AccessToken)
	defer addResponse.Body.Close()
	assertStatus(t, addResponse, http.StatusCreated)

	firstAward := awardFromCatalog(t, server, admin.AccessToken, slug, holder.SubjectID)
	defer firstAward.Body.Close()
	assertStatus(t, firstAward, http.StatusCreated)
	first := decodeCollectibleHTTPResponse(t, firstAward)

	// Releasing a live (never-withdrawn) instance conflicts.
	releaseLive := postJSONWithBearer(t, server.URL+"/api/admin/collectibles/"+first.ID+"/release", []byte(`{}`), admin.AccessToken)
	defer releaseLive.Body.Close()
	assertStatus(t, releaseLive, http.StatusConflict)

	withdraw := postJSONWithBearer(t, server.URL+"/api/admin/collectibles/"+first.ID+"/withdraw", []byte(`{}`), admin.AccessToken)
	defer withdraw.Body.Close()
	assertStatus(t, withdraw, http.StatusOK)

	// A non-admin cannot release the instance.
	forbidden := postJSONWithBearer(t, server.URL+"/api/admin/collectibles/"+first.ID+"/release", []byte(`{}`), holder.AccessToken)
	defer forbidden.Body.Close()
	assertStatus(t, forbidden, http.StatusForbidden)

	release := postJSONWithBearer(t, server.URL+"/api/admin/collectibles/"+first.ID+"/release", []byte(`{}`), admin.AccessToken)
	defer release.Body.Close()
	assertStatus(t, release, http.StatusOK)
	released := decodeCollectibleHTTPResponse(t, release)
	if released.State != "minted" || released.OwnerID != holder.SubjectID {
		t.Fatalf("released instance = %+v, want minted and still holder-owned", released)
	}
	if !hasNotificationKind(t, server, holder.AccessToken, "collectible_released") {
		t.Fatalf("holder inbox has no collectible_released notification")
	}
	if !hasAuditAction(t, server, admin.AccessToken, "admin_collectible_released", first.ID) {
		t.Fatalf("audit log has no admin_collectible_released row for %s", first.ID)
	}

	// Back in the holder's inventory as minted, with the owner label resolved.
	held := listCollectibles(t, server, holder.AccessToken)
	if len(held) != 1 || held[0].ID != first.ID || held[0].State != "minted" {
		t.Fatalf("holder holdings after release = %+v, want the released instance minted", held)
	}
	if held[0].OwnerDisplayName == "" {
		t.Fatalf("held collectible is missing its owner display label")
	}

	// The catalog's unique-entry ownership reflects the live holder.
	entry := fetchCatalogEntry(t, server, holder.AccessToken, slug)
	if entry.LiveOwnerCount != 1 || entry.OwnerDisplayName == "" {
		t.Fatalf("catalog ownership = %+v, want one live owner with a label", entry)
	}

	// Withdraw again and re-mint the unique slot: the old instance can no
	// longer be released.
	secondWithdraw := postJSONWithBearer(t, server.URL+"/api/admin/collectibles/"+first.ID+"/withdraw", []byte(`{}`), admin.AccessToken)
	defer secondWithdraw.Body.Close()
	assertStatus(t, secondWithdraw, http.StatusOK)
	replacementAward := awardFromCatalog(t, server, admin.AccessToken, slug, holder.SubjectID)
	defer replacementAward.Body.Close()
	assertStatus(t, replacementAward, http.StatusCreated)

	conflicted := postJSONWithBearer(t, server.URL+"/api/admin/collectibles/"+first.ID+"/release", []byte(`{}`), admin.AccessToken)
	defer conflicted.Body.Close()
	assertStatus(t, conflicted, http.StatusConflict)

	// Deleting the stuck withdrawn instance is still allowed.
	deleteOld := deleteWithBearer(t, server.URL+"/api/admin/collectibles/"+first.ID, admin.AccessToken)
	defer deleteOld.Body.Close()
	assertStatus(t, deleteOld, http.StatusOK)
}

// TestMCPReleaseCatalogEntryAndCollectible drives the two release tools over
// MCP: the admin double-check gates them, a withdrawn entry and instance are
// released, and the catalog payload carries the ownership fields.
func TestMCPReleaseCatalogEntryAndCollectible(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "mcp-release-admin")
	holder := registerUser(t, bootstrap, "mcp-release-holder")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	adminAgent := createAgentCredential(t, server, admin.AccessToken, []string{"platform_admin", "collectibles_read"})
	session := initializeMCPSession(t, server, adminAgent)

	slug := "mcp-release-" + uniqueTestSuffix(t)
	added := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `1`, "sharecrop.add_catalog_entry",
		`{"slug":"`+slug+`","name":"MCP Release Badge","kind":"badge","transfer_policy":"transferable_between_users","art":"scarecrow"}`)))
	if !strings.Contains(added, slug) {
		t.Fatalf("add_catalog_entry missing the slug: %s", added)
	}
	toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `2`, "sharecrop.withdraw_catalog_entry", `{"slug":"`+slug+`"}`)))

	releasedEntry := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `3`, "sharecrop.release_catalog_entry", `{"slug":"`+slug+`"}`)))
	if !strings.Contains(releasedEntry, `"state":"available"`) {
		t.Fatalf("release_catalog_entry did not report an available entry: %s", releasedEntry)
	}

	awarded := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `4`, "sharecrop.award_collectible",
		`{"slug":"`+slug+`","recipient_id":"`+holder.SubjectID+`"}`)))
	var instance struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(awarded), &instance); err != nil {
		t.Fatalf("decode award_collectible: %v (%s)", err, awarded)
	}

	toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `5`, "sharecrop.withdraw_collectible", `{"collectible_id":"`+instance.ID+`"}`)))
	releasedInstance := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `6`, "sharecrop.release_collectible", `{"collectible_id":"`+instance.ID+`"}`)))
	if !strings.Contains(releasedInstance, `"state":"minted"`) || !strings.Contains(releasedInstance, holder.SubjectID) {
		t.Fatalf("release_collectible did not return the minted holder-owned instance: %s", releasedInstance)
	}

	catalog := toolText(t, decodeRPC(t, mcpCall(t, server, adminAgent, session, `7`, "sharecrop.collectible_catalog", `{}`)))
	if !strings.Contains(catalog, `"live_owner_count"`) || !strings.Contains(catalog, `"owner_display_name"`) {
		t.Fatalf("collectible_catalog payload is missing the ownership fields: %s", catalog)
	}

	// A non-admin user's credential is refused by the double-check even with
	// the platform_admin scope.
	holderAgent := createAgentCredential(t, server, holder.AccessToken, []string{"platform_admin"})
	holderSession := initializeMCPSession(t, server, holderAgent)
	denied := decodeRPC(t, mcpCall(t, server, holderAgent, holderSession, `8`, "sharecrop.release_collectible", `{"collectible_id":"`+instance.ID+`"}`))
	var deniedResult struct {
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(denied.Result, &deniedResult); err != nil {
		t.Fatalf("decode denied result: %v", err)
	}
	if !deniedResult.IsError {
		t.Fatalf("release_collectible was not denied for a non-admin user")
	}
}
