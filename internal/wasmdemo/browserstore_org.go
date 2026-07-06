package wasmdemo

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
)

// OrgBrowserStore implements org.Store against BrowserStorage, so the real
// org.Service (the same code cmd/sharecrop runs against Postgres) can serve
// the browser demo directly.
type OrgBrowserStore struct {
	storage BrowserStorage
	ids     InteractionIDSource
}

func NewOrgBrowserStore(storage BrowserStorage, ids InteractionIDSource) OrgBrowserStore {
	return OrgBrowserStore{storage: storage, ids: ids}
}

type storedOrganization struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
}

type storedMembership struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	UserID         string   `json:"user_id"`
	Status         string   `json:"status"`
	Roles          []string `json:"roles"`
}

type storedTeam struct {
	ID             string `json:"id"`
	OwnerKind      string `json:"owner_kind"`
	OrganizationID string `json:"organization_id,omitempty"`
	OwnerUserID    string `json:"owner_user_id,omitempty"`
	Name           string `json:"name"`
	CreatedBy      string `json:"created_by"`
}

func orgRecordKey(id string) string { return "org:record:" + id }
func orgUserOrgsIndexKey(userID string) string {
	return "org:user_orgs:" + userID
}
func orgMembershipKey(id string) string { return "org:membership:" + id }
func orgMembershipIndexKey(organizationID string) string {
	return "org:org_memberships:" + organizationID
}
func orgActiveMembershipKey(organizationID string, userID string) string {
	return "org:active_membership:" + organizationID + ":" + userID
}

// orgAnyMembershipKey points to a (org, user) pair's membership id
// regardless of its current status, mirroring the real store's
// `organization_memberships_unique_user` constraint (one membership row per
// (organization_id, user_id) ever, active or not) - unlike
// orgActiveMembershipKey, this is never cleared by deactivation.
func orgAnyMembershipKey(organizationID string, userID string) string {
	return "org:any_membership:" + organizationID + ":" + userID
}

// isActiveOrgMember reports whether userID currently holds an active
// membership in organizationID. Shared by any browser store that needs to
// verify org membership without going through org.Service (e.g. a
// within-organization collectible transfer/award).
func isActiveOrgMember(storage BrowserStorage, organizationID string, userID string) (bool, bool) {
	membershipID, found, ok := getStorageString(storage, orgActiveMembershipKey(organizationID, userID))
	if !ok {
		return false, false
	}
	if !found {
		return false, true
	}
	membership, membershipFound, membershipOK := getStoredMembershipJSON(storage, orgMembershipKey(membershipID))
	if !membershipOK {
		return false, false
	}
	return membershipFound && membership.Status == org.MembershipStatusActive.String(), true
}
func teamRecordKey(id string) string { return "team:record:" + id }
func teamOrgIndexKey(organizationID string) string {
	return "team:org_index:" + organizationID
}
func teamUserIndexKey(userID string) string { return "team:user_index:" + userID }
func teamMembersKey(teamID string) string   { return "team:members:" + teamID }

func putStoredOrganizationJSON(storage BrowserStorage, rawKey string, record storedOrganization) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredOrganizationJSON(storage BrowserStorage, rawKey string) (storedOrganization, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedOrganization{}, found, ok
	}
	var record storedOrganization
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedOrganization{}, false, false
	}
	return record, true, true
}

// putStoredMembershipJSON/getStoredMembershipJSON are package-level (not
// OrgBrowserStore methods) since AssetBrowserStore.AwardOrganizationCollectible
// also reads membership records directly, sharing this one storage format.
func putStoredMembershipJSON(storage BrowserStorage, rawKey string, record storedMembership) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredMembershipJSON(storage BrowserStorage, rawKey string) (storedMembership, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedMembership{}, found, ok
	}
	var record storedMembership
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedMembership{}, false, false
	}
	return record, true, true
}

func putStoredTeamJSON(storage BrowserStorage, rawKey string, record storedTeam) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredTeamJSON(storage BrowserStorage, rawKey string) (storedTeam, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedTeam{}, found, ok
	}
	var record storedTeam
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedTeam{}, false, false
	}
	return record, true, true
}

// lookupUserIDByEmail reuses AuthBrowserStore's own email index directly
// (same underlying BrowserStorage, same package) rather than duplicating a
// second user-directory concept.
func (store OrgBrowserStore) lookupUserIDByEmail(email string) (string, bool, bool) {
	return getStorageString(store.storage, authUserEmailKey(email))
}

func (store OrgBrowserStore) CreateOrganization(_ context.Context, organizationID core.OrganizationID, name org.OrganizationName, createdBy core.UserID, membershipID core.OrganizationMembershipID) org.CreateOrganizationStoreResult {
	record := storedOrganization{ID: organizationID.String(), Name: name.String(), CreatedBy: createdBy.String()}
	if !putStoredOrganizationJSON(store.storage, orgRecordKey(record.ID), record) {
		return org.CreateOrganizationStoreRejected{Reason: invalidState("insert organization failed")}
	}

	membership := storedMembership{
		ID:             membershipID.String(),
		OrganizationID: organizationID.String(),
		UserID:         createdBy.String(),
		Status:         org.MembershipStatusActive.String(),
		Roles:          []string{org.RoleOwner.String()},
	}
	if !store.saveMembership(membership) {
		return org.CreateOrganizationStoreRejected{Reason: invalidState("insert organization owner membership failed")}
	}

	if !store.insertOrganizationCreditGrant(organizationID.String()) {
		return org.CreateOrganizationStoreRejected{Reason: invalidState("insert organization credit grant failed")}
	}

	return org.CreateOrganizationStoreAccepted{}
}

// insertOrganizationCreditGrant mirrors internal/db's
// insertOrganizationCreditGrant, reusing the same SaveLedgerEntry helper
// AuthBrowserStore's signup grant already uses.
func (store OrgBrowserStore) insertOrganizationCreditGrant(organizationID string) bool {
	entryID := strings.TrimSpace(store.ids.NextLedgerEntryID())
	result := SaveLedgerEntry(store.storage, StoredLedgerEntry{
		ID:        entryID,
		OwnerKind: "organization",
		OwnerID:   organizationID,
		Kind:      "signup_grant",
		Amount:    ledger.SignupGrantAmount().Int64(),
	})
	_, matched := result.(LedgerEntryStored)
	return matched
}

func (store OrgBrowserStore) saveMembership(membership storedMembership) bool {
	if !putStoredMembershipJSON(store.storage, orgMembershipKey(membership.ID), membership) {
		return false
	}
	if _, matched := appendStringIndex(store.storage, orgMembershipIndexKey(membership.OrganizationID), membership.ID, "organization membership").(stringIndexStored); !matched {
		return false
	}
	if _, matched := appendStringIndex(store.storage, orgUserOrgsIndexKey(membership.UserID), membership.OrganizationID, "user organization").(stringIndexStored); !matched {
		return false
	}
	if !putStorageString(store.storage, orgActiveMembershipKey(membership.OrganizationID, membership.UserID), membership.ID) {
		return false
	}
	if !putStorageString(store.storage, orgAnyMembershipKey(membership.OrganizationID, membership.UserID), membership.ID) {
		return false
	}
	return true
}

func (store OrgBrowserStore) ListOrganizationsForUser(_ context.Context, userID core.UserID, query string, page core.Page) org.ListOrganizationsResult {
	indexResult := loadStringIndex(store.storage, orgUserOrgsIndexKey(userID.String()), "user organization")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return org.ListOrganizationsRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}

	cleanQuery := strings.ToLower(strings.TrimSpace(query))
	matching := make([]org.Organization, 0, len(loaded.values))
	seen := make(map[string]bool, len(loaded.values))
	for _, organizationID := range loaded.values {
		if seen[organizationID] {
			continue
		}
		seen[organizationID] = true

		membershipID, found, ok := getStorageString(store.storage, orgActiveMembershipKey(organizationID, userID.String()))
		if !ok {
			return org.ListOrganizationsRejected{Reason: invalidState("find active membership failed")}
		}
		if !found {
			continue
		}
		membership, membershipFound, membershipOK := getStoredMembershipJSON(store.storage, orgMembershipKey(membershipID))
		if !membershipOK {
			return org.ListOrganizationsRejected{Reason: invalidState("read membership failed")}
		}
		if !membershipFound || membership.Status != org.MembershipStatusActive.String() {
			continue
		}

		record, orgFound, orgOK := getStoredOrganizationJSON(store.storage, orgRecordKey(organizationID))
		if !orgOK {
			return org.ListOrganizationsRejected{Reason: invalidState("read organization failed")}
		}
		if !orgFound {
			continue
		}
		if cleanQuery != "" && !strings.Contains(strings.ToLower(record.Name), cleanQuery) {
			continue
		}

		organization, parseErr := parseStoredOrganization(record)
		if parseErr != nil {
			return org.ListOrganizationsRejected{Reason: *parseErr}
		}
		matching = append(matching, organization)
	}

	sort.Slice(matching, func(i, j int) bool { return matching[i].Name.String() < matching[j].Name.String() })
	return org.OrganizationsListed{Values: paginateOrganizations(matching, page)}
}

func paginateOrganizations(values []org.Organization, page core.Page) []org.Organization {
	start := page.Offset()
	if start > len(values) {
		start = len(values)
	}
	end := start + page.Limit()
	if end > len(values) {
		end = len(values)
	}
	result := make([]org.Organization, end-start)
	copy(result, values[start:end])
	return result
}

func parseStoredOrganization(record storedOrganization) (org.Organization, *core.DomainError) {
	idResult := core.ParseOrganizationID(record.ID)
	id, idMatched := idResult.(core.OrganizationIDCreated)
	if !idMatched {
		reason := idResult.(core.OrganizationIDRejected).Reason
		return org.Organization{}, &reason
	}
	nameResult := org.NewOrganizationName(record.Name)
	name, nameMatched := nameResult.(org.OrganizationNameAccepted)
	if !nameMatched {
		reason := nameResult.(org.OrganizationNameRejected).Reason
		return org.Organization{}, &reason
	}
	createdByResult := core.ParseUserID(record.CreatedBy)
	createdBy, createdByMatched := createdByResult.(core.UserIDCreated)
	if !createdByMatched {
		reason := createdByResult.(core.UserIDRejected).Reason
		return org.Organization{}, &reason
	}
	return org.Organization{ID: id.Value, Name: name.Value, CreatedBy: createdBy.Value}, nil
}

func parseStoredMembership(record storedMembership) (org.OrganizationMember, *core.DomainError) {
	idResult := core.ParseOrganizationMembershipID(record.ID)
	id, idMatched := idResult.(core.OrganizationMembershipIDCreated)
	if !idMatched {
		reason := idResult.(core.OrganizationMembershipIDRejected).Reason
		return org.OrganizationMember{}, &reason
	}
	organizationResult := core.ParseOrganizationID(record.OrganizationID)
	organizationID, organizationMatched := organizationResult.(core.OrganizationIDCreated)
	if !organizationMatched {
		reason := organizationResult.(core.OrganizationIDRejected).Reason
		return org.OrganizationMember{}, &reason
	}
	userResult := core.ParseUserID(record.UserID)
	userID, userMatched := userResult.(core.UserIDCreated)
	if !userMatched {
		reason := userResult.(core.UserIDRejected).Reason
		return org.OrganizationMember{}, &reason
	}
	statusResult := org.ParseMembershipStatus(record.Status)
	status, statusMatched := statusResult.(org.MembershipStatusAccepted)
	if !statusMatched {
		reason := statusResult.(org.MembershipStatusRejected).Reason
		return org.OrganizationMember{}, &reason
	}
	roles := make([]org.Role, 0, len(record.Roles))
	for _, raw := range record.Roles {
		roleResult := org.ParseRole(raw)
		roleAccepted, roleMatched := roleResult.(org.RoleAccepted)
		if !roleMatched {
			reason := roleResult.(org.RoleRejected).Reason
			return org.OrganizationMember{}, &reason
		}
		roles = append(roles, roleAccepted.Value)
	}
	return org.OrganizationMember{ID: id.Value, OrganizationID: organizationID.Value, UserID: userID.Value, Status: status.Value, Roles: roles}, nil
}

func (store OrgBrowserStore) FindMemberRoles(_ context.Context, organizationID core.OrganizationID, userID core.UserID) org.MemberRolesResult {
	membershipID, found, ok := getStorageString(store.storage, orgActiveMembershipKey(organizationID.String(), userID.String()))
	if !ok {
		return org.MemberRolesRejected{Reason: invalidState("find member roles failed")}
	}
	if !found {
		return org.MemberRolesMissing{}
	}
	membership, membershipFound, membershipOK := getStoredMembershipJSON(store.storage, orgMembershipKey(membershipID))
	if !membershipOK {
		return org.MemberRolesRejected{Reason: invalidState("read membership failed")}
	}
	if !membershipFound || membership.Status != org.MembershipStatusActive.String() {
		return org.MemberRolesMissing{}
	}
	roles := make([]org.Role, 0, len(membership.Roles))
	for _, raw := range membership.Roles {
		roleResult := org.ParseRole(raw)
		roleAccepted, roleMatched := roleResult.(org.RoleAccepted)
		if !roleMatched {
			return org.MemberRolesRejected{Reason: roleResult.(org.RoleRejected).Reason}
		}
		roles = append(roles, roleAccepted.Value)
	}
	return org.MemberRolesFound{Roles: roles}
}

func (store OrgBrowserStore) ListMembers(_ context.Context, organizationID core.OrganizationID, page core.Page) org.ListMembersResult {
	indexResult := loadStringIndex(store.storage, orgMembershipIndexKey(organizationID.String()), "organization membership")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return org.ListMembersRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}

	values := make([]org.OrganizationMember, 0, len(loaded.values))
	for _, membershipID := range loaded.values {
		record, found, ok := getStoredMembershipJSON(store.storage, orgMembershipKey(membershipID))
		if !ok {
			return org.ListMembersRejected{Reason: invalidState("read membership failed")}
		}
		if !found || record.Status == "removed" {
			continue
		}
		member, parseErr := parseStoredMembership(record)
		if parseErr != nil {
			return org.ListMembersRejected{Reason: *parseErr}
		}
		values = append(values, member)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].UserID.String() < values[j].UserID.String() })

	start := page.Offset()
	if start > len(values) {
		start = len(values)
	}
	end := start + page.Limit()
	if end > len(values) {
		end = len(values)
	}
	return org.MembersListed{Values: values[start:end]}
}

func (store OrgBrowserStore) ProvisionMember(_ context.Context, membershipID core.OrganizationMembershipID, organizationID core.OrganizationID, email auth.EmailAddress, roles []org.Role) org.ProvisionMemberStoreResult {
	userID, found, ok := store.lookupUserIDByEmail(email.String())
	if !ok {
		return org.ProvisionMemberStoreRejected{Reason: invalidState("lookup user by email failed")}
	}
	if !found {
		return org.ProvisionMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "user was not found for email address")}
	}
	_, alreadyMember, ok := getStorageString(store.storage, orgAnyMembershipKey(organizationID.String(), userID))
	if !ok {
		return org.ProvisionMemberStoreRejected{Reason: invalidState("check existing organization membership failed")}
	}
	if alreadyMember {
		return org.ProvisionMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "user is already a member of this organization")}
	}

	rawRoles := make([]string, len(roles))
	for index, role := range roles {
		rawRoles[index] = role.String()
	}
	membership := storedMembership{
		ID:             membershipID.String(),
		OrganizationID: organizationID.String(),
		UserID:         userID,
		Status:         org.MembershipStatusActive.String(),
		Roles:          rawRoles,
	}
	if !store.saveMembership(membership) {
		return org.ProvisionMemberStoreRejected{Reason: invalidState("insert organization membership failed")}
	}
	member, parseErr := parseStoredMembership(membership)
	if parseErr != nil {
		return org.ProvisionMemberStoreRejected{Reason: *parseErr}
	}
	return org.MemberProvisioned{Value: member}
}

func (store OrgBrowserStore) DeactivateMember(_ context.Context, organizationID core.OrganizationID, userID core.UserID) org.DeactivateMemberStoreResult {
	membershipID, found, ok := getStorageString(store.storage, orgActiveMembershipKey(organizationID.String(), userID.String()))
	if !ok {
		return org.DeactivateMemberStoreRejected{Reason: invalidState("deactivate member failed")}
	}
	if !found {
		return org.DeactivateMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "active organization member was not found")}
	}
	membership, membershipFound, membershipOK := getStoredMembershipJSON(store.storage, orgMembershipKey(membershipID))
	if !membershipOK {
		return org.DeactivateMemberStoreRejected{Reason: invalidState("read membership failed")}
	}
	if !membershipFound || membership.Status != org.MembershipStatusActive.String() {
		return org.DeactivateMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "active organization member was not found")}
	}
	membership.Status = org.MembershipStatusDeactivated.String()
	if !putStoredMembershipJSON(store.storage, orgMembershipKey(membershipID), membership) {
		return org.DeactivateMemberStoreRejected{Reason: invalidState("deactivate member failed")}
	}
	return org.MemberDeactivated{}
}

func (store OrgBrowserStore) UpdateMemberRoles(_ context.Context, organizationID core.OrganizationID, userID core.UserID, roles []org.Role) org.UpdateMemberRolesStoreResult {
	membershipID, found, ok := getStorageString(store.storage, orgActiveMembershipKey(organizationID.String(), userID.String()))
	if !ok {
		return org.UpdateMemberRolesStoreRejected{Reason: invalidState("update member roles failed")}
	}
	if !found {
		return org.UpdateMemberRolesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "active organization member was not found")}
	}
	membership, membershipFound, membershipOK := getStoredMembershipJSON(store.storage, orgMembershipKey(membershipID))
	if !membershipOK {
		return org.UpdateMemberRolesStoreRejected{Reason: invalidState("read membership failed")}
	}
	if !membershipFound || membership.Status != org.MembershipStatusActive.String() {
		return org.UpdateMemberRolesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "active organization member was not found")}
	}
	rawRoles := make([]string, len(roles))
	for index, role := range roles {
		rawRoles[index] = role.String()
	}
	membership.Roles = rawRoles
	if !putStoredMembershipJSON(store.storage, orgMembershipKey(membershipID), membership) {
		return org.UpdateMemberRolesStoreRejected{Reason: invalidState("update member roles failed")}
	}
	member, parseErr := parseStoredMembership(membership)
	if parseErr != nil {
		return org.UpdateMemberRolesStoreRejected{Reason: *parseErr}
	}
	return org.MemberRolesUpdated{Value: member}
}

func (store OrgBrowserStore) saveTeam(record storedTeam) bool {
	if !putStoredTeamJSON(store.storage, teamRecordKey(record.ID), record) {
		return false
	}
	if record.OwnerKind == "organization" {
		if _, matched := appendStringIndex(store.storage, teamOrgIndexKey(record.OrganizationID), record.ID, "team").(stringIndexStored); !matched {
			return false
		}
	} else {
		if _, matched := appendStringIndex(store.storage, teamUserIndexKey(record.OwnerUserID), record.ID, "team").(stringIndexStored); !matched {
			return false
		}
	}
	return true
}

func (store OrgBrowserStore) CreateOrganizationTeam(_ context.Context, teamID core.TeamID, organizationID core.OrganizationID, name org.TeamName, createdBy core.UserID) org.CreateTeamStoreResult {
	record := storedTeam{ID: teamID.String(), OwnerKind: "organization", OrganizationID: organizationID.String(), Name: name.String(), CreatedBy: createdBy.String()}
	if !store.saveTeam(record) {
		return org.CreateTeamStoreRejected{Reason: invalidState("insert organization team failed")}
	}
	return org.CreateTeamStoreAccepted{}
}

func (store OrgBrowserStore) CreateStandaloneTeam(_ context.Context, teamID core.TeamID, ownerUserID core.UserID, name org.TeamName) org.CreateTeamStoreResult {
	record := storedTeam{ID: teamID.String(), OwnerKind: "user", OwnerUserID: ownerUserID.String(), Name: name.String(), CreatedBy: ownerUserID.String()}
	if !store.saveTeam(record) {
		return org.CreateTeamStoreRejected{Reason: invalidState("insert standalone team failed")}
	}
	return org.CreateTeamStoreAccepted{}
}

func (store OrgBrowserStore) AddTeamMember(_ context.Context, teamID core.TeamID, userID core.UserID) org.AddTeamMemberStoreResult {
	indexResult := loadStringIndex(store.storage, teamMembersKey(teamID.String()), "team member")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return org.AddTeamMemberStoreRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}
	for _, existing := range loaded.values {
		if existing == userID.String() {
			return org.TeamMemberAdded{}
		}
	}
	if _, matched := appendStringIndex(store.storage, teamMembersKey(teamID.String()), userID.String(), "team member").(stringIndexStored); !matched {
		return org.AddTeamMemberStoreRejected{Reason: invalidState("insert team member failed")}
	}
	return org.TeamMemberAdded{}
}

func (store OrgBrowserStore) AddTeamMemberByEmail(ctx context.Context, teamID core.TeamID, email auth.EmailAddress) org.AddTeamMemberStoreResult {
	userID, found, ok := store.lookupUserIDByEmail(email.String())
	if !ok {
		return org.AddTeamMemberStoreRejected{Reason: invalidState("lookup user by email failed")}
	}
	if !found {
		return org.AddTeamMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "user was not found for email address")}
	}
	userIDResult := core.ParseUserID(userID)
	userIDCreated, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		return org.AddTeamMemberStoreRejected{Reason: userIDResult.(core.UserIDRejected).Reason}
	}
	return store.AddTeamMember(ctx, teamID, userIDCreated.Value)
}

func parseStoredTeam(record storedTeam) (org.Team, *core.DomainError) {
	idResult := core.ParseTeamID(record.ID)
	id, idMatched := idResult.(core.TeamIDCreated)
	if !idMatched {
		reason := idResult.(core.TeamIDRejected).Reason
		return org.Team{}, &reason
	}
	nameResult := org.NewTeamName(record.Name)
	name, nameMatched := nameResult.(org.TeamNameAccepted)
	if !nameMatched {
		reason := nameResult.(org.TeamNameRejected).Reason
		return org.Team{}, &reason
	}
	createdByResult := core.ParseUserID(record.CreatedBy)
	createdBy, createdByMatched := createdByResult.(core.UserIDCreated)
	if !createdByMatched {
		reason := createdByResult.(core.UserIDRejected).Reason
		return org.Team{}, &reason
	}

	var owner org.TeamOwner
	if record.OwnerKind == "organization" {
		organizationResult := core.ParseOrganizationID(record.OrganizationID)
		organizationID, organizationMatched := organizationResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			reason := organizationResult.(core.OrganizationIDRejected).Reason
			return org.Team{}, &reason
		}
		owner = org.OrganizationOwnedTeam{OrganizationID: organizationID.Value}
	} else {
		ownerResult := core.ParseUserID(record.OwnerUserID)
		ownerUserID, ownerMatched := ownerResult.(core.UserIDCreated)
		if !ownerMatched {
			reason := ownerResult.(core.UserIDRejected).Reason
			return org.Team{}, &reason
		}
		owner = org.UserOwnedTeam{OwnerUserID: ownerUserID.Value}
	}

	return org.Team{ID: id.Value, Owner: owner, Name: name.Value, CreatedBy: createdBy.Value}, nil
}

func (store OrgBrowserStore) listTeams(indexKey string, query string, page core.Page) org.TeamListResult {
	indexResult := loadStringIndex(store.storage, indexKey, "team")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return org.TeamListRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}

	cleanQuery := strings.ToLower(strings.TrimSpace(query))
	values := make([]org.Team, 0, len(loaded.values))
	for _, teamID := range loaded.values {
		record, found, ok := getStoredTeamJSON(store.storage, teamRecordKey(teamID))
		if !ok {
			return org.TeamListRejected{Reason: invalidState("read team failed")}
		}
		if !found {
			continue
		}
		if cleanQuery != "" && !strings.Contains(strings.ToLower(record.Name), cleanQuery) {
			continue
		}
		team, parseErr := parseStoredTeam(record)
		if parseErr != nil {
			return org.TeamListRejected{Reason: *parseErr}
		}
		values = append(values, team)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name.String() < values[j].Name.String() })

	start := page.Offset()
	if start > len(values) {
		start = len(values)
	}
	end := start + page.Limit()
	if end > len(values) {
		end = len(values)
	}
	return org.TeamsListed{Values: values[start:end]}
}

func (store OrgBrowserStore) ListOrganizationTeams(_ context.Context, organizationID core.OrganizationID, userID core.UserID, query string, page core.Page) org.TeamListResult {
	if _, matched := store.FindMemberRoles(context.Background(), organizationID, userID).(org.MemberRolesFound); !matched {
		return org.TeamListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "organization team access denied")}
	}
	return store.listTeams(teamOrgIndexKey(organizationID.String()), query, page)
}

func (store OrgBrowserStore) ListStandaloneTeams(_ context.Context, ownerUserID core.UserID, query string, page core.Page) org.TeamListResult {
	return store.listTeams(teamUserIndexKey(ownerUserID.String()), query, page)
}

func (store OrgBrowserStore) FindTeam(_ context.Context, teamID core.TeamID) org.FindTeamResult {
	record, found, ok := getStoredTeamJSON(store.storage, teamRecordKey(teamID.String()))
	if !ok {
		return org.TeamMissing{Reason: invalidState("find team failed")}
	}
	if !found {
		return org.TeamMissing{Reason: core.NewDomainError(core.ErrorCodeNotFound, "team not found")}
	}
	team, parseErr := parseStoredTeam(record)
	if parseErr != nil {
		return org.TeamMissing{Reason: *parseErr}
	}
	return org.TeamFound{Value: team}
}

func (store OrgBrowserStore) ListTeamMembers(_ context.Context, teamID core.TeamID) org.TeamMembersResult {
	indexResult := loadStringIndex(store.storage, teamMembersKey(teamID.String()), "team member")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return org.TeamMembersRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}
	values := make([]core.UserID, 0, len(loaded.values))
	for _, raw := range loaded.values {
		userResult := core.ParseUserID(raw)
		userCreated, userMatched := userResult.(core.UserIDCreated)
		if !userMatched {
			return org.TeamMembersRejected{Reason: userResult.(core.UserIDRejected).Reason}
		}
		values = append(values, userCreated.Value)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].String() < values[j].String() })
	return org.TeamMembersListed{Values: values}
}
