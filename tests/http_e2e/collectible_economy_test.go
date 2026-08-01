//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type catalogEntryHTTPResponse struct {
	Slug             string `json:"slug"`
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	TransferPolicy   string `json:"transfer_policy"`
	Art              string `json:"art"`
	State            string `json:"state"`
	MaxEditions      int64  `json:"max_editions"`
	MintedCount      int64  `json:"minted_count"`
	LiveOwnerCount   int64  `json:"live_owner_count"`
	OwnerDisplayName string `json:"owner_display_name"`
}

func fetchCatalogEntry(t *testing.T, server *httptest.Server, accessToken string, slug string) catalogEntryHTTPResponse {
	t.Helper()
	response := getWithBearer(t, server.URL+"/api/collectibles/catalog", accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var catalog struct {
		Entries []catalogEntryHTTPResponse `json:"entries"`
	}
	if err := json.NewDecoder(response.Body).Decode(&catalog); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	for _, entry := range catalog.Entries {
		if entry.Slug == slug {
			return entry
		}
	}
	t.Fatalf("catalog has no entry %q", slug)
	return catalogEntryHTTPResponse{}
}

func awardFromCatalog(t *testing.T, server *httptest.Server, adminToken string, slug string, recipientID string) *http.Response {
	t.Helper()
	return postJSONWithBearer(t, server.URL+"/api/collectibles/award",
		[]byte(`{"slug":"`+slug+`","recipient_kind":"user","recipient_id":"`+recipientID+`"}`), adminToken)
}

// TestAdminCatalogEntryLifecycle drives the full catalog administration
// surface: add an edition entry, award numbered instances, withdraw the
// entry (award refused afterward), withdraw and delete an instance, and
// finally delete the withdrawn entry once no live instance remains.
func TestAdminCatalogEntryLifecycle(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "catalog-admin")
	holder := registerUser(t, bootstrap, "catalog-holder")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	slug := "spring-run-" + uniqueTestSuffix(t)

	// A non-admin cannot manage the catalog.
	forbidden := postJSONWithBearer(t, server.URL+"/api/admin/collectible-catalog",
		[]byte(`{"slug":"`+slug+`","name":"Spring Run","kind":"edition","transfer_policy":"transferable_between_users","art":"seedling","max_editions":2}`), holder.AccessToken)
	defer forbidden.Body.Close()
	assertStatus(t, forbidden, http.StatusForbidden)

	// The admin adds an edition entry capped at 2 instances.
	addResponse := postJSONWithBearer(t, server.URL+"/api/admin/collectible-catalog",
		[]byte(`{"slug":"`+slug+`","name":"Spring Run","kind":"edition","transfer_policy":"transferable_between_users","art":"seedling","max_editions":2}`), admin.AccessToken)
	defer addResponse.Body.Close()
	assertStatus(t, addResponse, http.StatusCreated)

	// The art must come from the fixed sprite registry.
	badArt := postJSONWithBearer(t, server.URL+"/api/admin/collectible-catalog",
		[]byte(`{"slug":"`+slug+`-bad","name":"Bad Art","kind":"badge","transfer_policy":"transferable_between_users","art":"not-a-sprite"}`), admin.AccessToken)
	defer badArt.Body.Close()
	assertStatus(t, badArt, http.StatusBadRequest)

	entry := fetchCatalogEntry(t, server, admin.AccessToken, slug)
	if entry.State != "available" || entry.MaxEditions != 2 || entry.MintedCount != 0 {
		t.Fatalf("fresh entry = %+v, want available, max 2, 0 minted", entry)
	}

	// Award two numbered instances; the cap refuses a third.
	firstAward := awardFromCatalog(t, server, admin.AccessToken, slug, holder.SubjectID)
	defer firstAward.Body.Close()
	assertStatus(t, firstAward, http.StatusCreated)
	first := decodeCollectibleHTTPResponse(t, firstAward)
	if first.CatalogSlug != slug || first.EditionNumber != 1 {
		t.Fatalf("first instance = %+v, want catalog slug %q edition 1", first, slug)
	}
	secondAward := awardFromCatalog(t, server, admin.AccessToken, slug, holder.SubjectID)
	defer secondAward.Body.Close()
	assertStatus(t, secondAward, http.StatusCreated)
	second := decodeCollectibleHTTPResponse(t, secondAward)
	if second.EditionNumber != 2 {
		t.Fatalf("second instance edition = %d, want 2", second.EditionNumber)
	}
	capped := awardFromCatalog(t, server, admin.AccessToken, slug, holder.SubjectID)
	defer capped.Body.Close()
	assertStatus(t, capped, http.StatusConflict)

	// The holder's listing shows provenance: catalog slug, edition number,
	// and the awarding admin's display name.
	held := listCollectibles(t, server, holder.AccessToken)
	if len(held) != 2 {
		t.Fatalf("holder collectible count = %d, want 2", len(held))
	}
	for _, collectible := range held {
		if collectible.CatalogSlug != slug || collectible.EditionNumber == 0 || collectible.IssuerDisplayName == "" {
			t.Fatalf("held collectible %+v is missing provenance (slug/edition/issuer)", collectible)
		}
	}

	// Deleting the entry is refused while it is available and has live
	// instances.
	deleteAvailable := deleteWithBearer(t, server.URL+"/api/admin/collectible-catalog/"+slug, admin.AccessToken)
	defer deleteAvailable.Body.Close()
	assertStatus(t, deleteAvailable, http.StatusConflict)

	// Withdrawing the entry stops further awards; existing instances stay.
	withdrawEntry := postJSONWithBearer(t, server.URL+"/api/admin/collectible-catalog/"+slug+"/withdraw", []byte(`{}`), admin.AccessToken)
	defer withdrawEntry.Body.Close()
	assertStatus(t, withdrawEntry, http.StatusOK)
	refused := awardFromCatalog(t, server, admin.AccessToken, slug, holder.SubjectID)
	defer refused.Body.Close()
	assertStatus(t, refused, http.StatusConflict)
	if entry := fetchCatalogEntry(t, server, holder.AccessToken, slug); entry.State != "withdrawn" || entry.MintedCount != 2 {
		t.Fatalf("withdrawn entry = %+v, want state withdrawn with 2 live instances", entry)
	}

	// Deleting an instance requires withdrawing it first.
	deleteLive := deleteWithBearer(t, server.URL+"/api/admin/collectibles/"+first.ID, admin.AccessToken)
	defer deleteLive.Body.Close()
	assertStatus(t, deleteLive, http.StatusConflict)

	// Withdraw both instances; the holder is notified and no longer lists them.
	for _, id := range []string{first.ID, second.ID} {
		withdrawInstance := postJSONWithBearer(t, server.URL+"/api/admin/collectibles/"+id+"/withdraw", []byte(`{}`), admin.AccessToken)
		defer withdrawInstance.Body.Close()
		assertStatus(t, withdrawInstance, http.StatusOK)
		withdrawn := decodeCollectibleHTTPResponse(t, withdrawInstance)
		if withdrawn.State != "withdrawn" {
			t.Fatalf("instance state after withdraw = %q, want withdrawn", withdrawn.State)
		}
	}
	if !hasNotificationKind(t, server, holder.AccessToken, "collectible_withdrawn") {
		t.Fatalf("holder inbox has no collectible_withdrawn notification")
	}

	// With no live instances, the instances can be hard-deleted and then the
	// withdrawn entry itself.
	for _, id := range []string{first.ID, second.ID} {
		deleteInstance := deleteWithBearer(t, server.URL+"/api/admin/collectibles/"+id, admin.AccessToken)
		defer deleteInstance.Body.Close()
		assertStatus(t, deleteInstance, http.StatusOK)
	}
	deleteEntry := deleteWithBearer(t, server.URL+"/api/admin/collectible-catalog/"+slug, admin.AccessToken)
	defer deleteEntry.Body.Close()
	assertStatus(t, deleteEntry, http.StatusOK)
}

// TestUniqueCatalogEntryAllowsOneLiveInstance pins the unique-kind engine
// rule end to end: a second award of a unique conflicts while the first
// instance is live.
func TestUniqueCatalogEntryAllowsOneLiveInstance(t *testing.T) {
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "unique-admin")
	holder := registerUser(t, bootstrap, "unique-holder")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	slug := "one-of-one-" + uniqueTestSuffix(t)
	addResponse := postJSONWithBearer(t, server.URL+"/api/admin/collectible-catalog",
		[]byte(`{"slug":"`+slug+`","name":"One Of One","kind":"unique","transfer_policy":"transferable_between_users","art":"golden-egg","max_editions":1}`), admin.AccessToken)
	defer addResponse.Body.Close()
	assertStatus(t, addResponse, http.StatusCreated)

	firstAward := awardFromCatalog(t, server, admin.AccessToken, slug, holder.SubjectID)
	defer firstAward.Body.Close()
	assertStatus(t, firstAward, http.StatusCreated)

	conflict := awardFromCatalog(t, server, admin.AccessToken, slug, holder.SubjectID)
	defer conflict.Body.Close()
	assertStatus(t, conflict, http.StatusConflict)
}

// TestCollectibleTransfersBetweenUsersAndOrganizations covers the peer
// collectible economy: user-to-user gifting under the default tradeable
// policy, donating to an organization, and the organization sending one of
// its collectibles to a user (gated by manage_collectibles).
func TestCollectibleTransfersBetweenUsersAndOrganizations(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "collectible-econ-owner")
	friend := registerUser(t, server, "collectible-econ-friend")
	organizationID := createOrganization(t, server, owner, "Collectible Economy Org")

	memberEmail := "collectible-econ-member-" + uniqueTestSuffix(t) + "@example.com"
	member := registerUserWithEmail(t, server, memberEmail)
	provisionOrganizationMember(t, server, owner.AccessToken, organizationID, memberEmail, `["member"]`)

	// The default mint policy is freely tradeable, so a user-to-user
	// transfer works without declaring a policy.
	mintResponse := postJSONWithBearer(t, server.URL+"/api/collectibles", []byte(`{"name":"Trade Token","kind":"badge"}`), owner.AccessToken)
	defer mintResponse.Body.Close()
	assertStatus(t, mintResponse, http.StatusCreated)
	traded := decodeCollectibleHTTPResponse(t, mintResponse)

	toFriend := postJSONWithBearer(t, server.URL+"/api/collectibles/"+traded.ID+"/transfer",
		[]byte(`{"recipient_id":"`+friend.SubjectID+`"}`), owner.AccessToken)
	defer toFriend.Body.Close()
	assertStatus(t, toFriend, http.StatusOK)
	if held := listCollectibles(t, server, friend.AccessToken); len(held) != 1 || held[0].ID != traded.ID {
		t.Fatalf("friend holdings = %+v, want the traded collectible", held)
	}

	// The friend donates it to the organization's trophy case.
	toOrg := postJSONWithBearer(t, server.URL+"/api/collectibles/"+traded.ID+"/transfer",
		[]byte(`{"target_kind":"organization","recipient_id":"`+organizationID+`"}`), friend.AccessToken)
	defer toOrg.Body.Close()
	assertStatus(t, toOrg, http.StatusOK)
	donated := decodeCollectibleHTTPResponse(t, toOrg)
	if donated.OwnerKind != "organization" || donated.OwnerID != organizationID {
		t.Fatalf("donated collectible = %+v, want organization-owned", donated)
	}

	// An unknown target kind is rejected as an enum error.
	badKind := postJSONWithBearer(t, server.URL+"/api/collectibles/"+traded.ID+"/transfer",
		[]byte(`{"target_kind":"team","recipient_id":"`+organizationID+`"}`), friend.AccessToken)
	defer badKind.Body.Close()
	assertStatus(t, badKind, http.StatusBadRequest)

	// A plain member cannot move the organization's collectible out.
	memberDenied := postJSONWithBearer(t, server.URL+"/api/organizations/"+organizationID+"/collectibles/"+traded.ID+"/transfer",
		[]byte(`{"recipient_id":"`+member.SubjectID+`"}`), member.AccessToken)
	defer memberDenied.Body.Close()
	assertStatus(t, memberDenied, http.StatusForbidden)

	// The owner (manage_collectibles via the owner role) sends it to a
	// user — unlike the member award route, the recipient need not be a
	// member, so send it back to the outside friend.
	fromOrg := postJSONWithBearer(t, server.URL+"/api/organizations/"+organizationID+"/collectibles/"+traded.ID+"/transfer",
		[]byte(`{"recipient_id":"`+friend.SubjectID+`"}`), owner.AccessToken)
	defer fromOrg.Body.Close()
	assertStatus(t, fromOrg, http.StatusOK)
	returned := decodeCollectibleHTTPResponse(t, fromOrg)
	if returned.OwnerKind != "user" || returned.OwnerID != friend.SubjectID {
		t.Fatalf("returned collectible = %+v, want owned by the friend", returned)
	}
	if held := listCollectibles(t, server, friend.AccessToken); len(held) != 1 || held[0].ID != traded.ID {
		t.Fatalf("friend holdings after org send = %+v, want the collectible back", held)
	}
}

func deleteWithBearer(t *testing.T, url string, accessToken string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("build DELETE request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("DELETE %s: %v", url, err)
	}
	return response
}
