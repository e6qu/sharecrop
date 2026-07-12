package db

import (
	"context"
	"errors"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrgStore struct {
	db Beginner
}

func NewOrgStore(pool *pgxpool.Pool) OrgStore {
	return OrgStore{db: NewPGX(pool)}
}

func (store OrgStore) CreateOrganization(ctx context.Context, organizationID core.OrganizationID, name org.OrganizationName, createdBy core.UserID, membershipID core.OrganizationMembershipID) org.CreateOrganizationStoreResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return org.CreateOrganizationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin create organization transaction failed")}
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, "insert into organizations (id, name, created_by_user_id) values ($1, $2, $3)", organizationID.String(), name.String(), createdBy.String())
	if err != nil {
		return org.CreateOrganizationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert organization failed")}
	}

	_, err = tx.Exec(ctx, "insert into organization_memberships (id, organization_id, user_id, status) values ($1, $2, $3, $4)", membershipID.String(), organizationID.String(), createdBy.String(), org.MembershipStatusActive.String())
	if err != nil {
		return org.CreateOrganizationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert organization owner membership failed")}
	}

	_, err = tx.Exec(ctx, "insert into organization_membership_roles (membership_id, role) values ($1, $2)", membershipID.String(), org.RoleOwner.String())
	if err != nil {
		return org.CreateOrganizationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert organization owner role failed")}
	}

	grantResult := insertOrganizationCreditGrant(ctx, tx, organizationID)
	if rejected, matched := grantResult.(signupGrantRejected); matched {
		return org.CreateOrganizationStoreRejected{Reason: rejected.reason}
	}

	if err := tx.Commit(ctx); err != nil {
		return org.CreateOrganizationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit create organization transaction failed")}
	}

	return org.CreateOrganizationStoreAccepted{}
}

func (store OrgStore) ListOrganizationsForUser(ctx context.Context, userID core.UserID, query string, page core.Page) org.ListOrganizationsResult {
	rows, err := store.db.Query(ctx, `
		select organizations.id::text, organizations.name, organizations.created_by_user_id::text
		from organizations
		join organization_memberships on organization_memberships.organization_id = organizations.id
		where organization_memberships.user_id = $1
			and organization_memberships.status = $2
			and ($3 = '' or lower(organizations.name) like '%' || lower($3) || '%')
		order by organizations.name
		limit $4 offset $5
	`, userID.String(), org.MembershipStatusActive.String(), query, page.Limit(), page.Offset())
	if err != nil {
		return org.ListOrganizationsRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list organizations failed")}
	}
	defer rows.Close()

	values := make([]org.Organization, 0)
	for rows.Next() {
		var rawID string
		var rawName string
		var rawCreatedBy string
		if err := rows.Scan(&rawID, &rawName, &rawCreatedBy); err != nil {
			return org.ListOrganizationsRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan organization failed")}
		}
		parsed := parseOrganizationRow(rawID, rawName, rawCreatedBy)
		accepted, matched := parsed.(organizationRowAccepted)
		if !matched {
			rejected := parsed.(organizationRowRejected)
			return org.ListOrganizationsRejected{Reason: rejected.reason}
		}
		values = append(values, accepted.value)
	}

	if err := rows.Err(); err != nil {
		return org.ListOrganizationsRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read organizations failed")}
	}

	return org.OrganizationsListed{Values: values}
}

func (store OrgStore) FindMemberRoles(ctx context.Context, organizationID core.OrganizationID, userID core.UserID) org.MemberRolesResult {
	rows, err := store.db.Query(ctx, `
		select organization_membership_roles.role
		from organization_memberships
		join organization_membership_roles on organization_membership_roles.membership_id = organization_memberships.id
		where organization_memberships.organization_id = $1
			and organization_memberships.user_id = $2
			and organization_memberships.status = $3
		order by organization_membership_roles.role
	`, organizationID.String(), userID.String(), org.MembershipStatusActive.String())
	if err != nil {
		return org.MemberRolesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "find member roles failed")}
	}
	defer rows.Close()

	roles := make([]org.Role, 0)
	for rows.Next() {
		var rawRole string
		if err := rows.Scan(&rawRole); err != nil {
			return org.MemberRolesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan member role failed")}
		}
		roleResult := org.ParseRole(rawRole)
		roleAccepted, roleMatched := roleResult.(org.RoleAccepted)
		if !roleMatched {
			rejected := roleResult.(org.RoleRejected)
			return org.MemberRolesRejected{Reason: rejected.Reason}
		}
		roles = append(roles, roleAccepted.Value)
	}

	if err := rows.Err(); err != nil {
		return org.MemberRolesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read member roles failed")}
	}

	if len(roles) == 0 {
		return org.MemberRolesMissing{}
	}

	return org.MemberRolesFound{Roles: roles}
}

func (store OrgStore) ProvisionMember(ctx context.Context, membershipID core.OrganizationMembershipID, organizationID core.OrganizationID, email auth.EmailAddress, roles []org.Role) org.ProvisionMemberStoreResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return org.ProvisionMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin provision member transaction failed")}
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	userIDResult := lookupUserIDByEmail(ctx, tx, email)
	userIDFound, userIDMatched := userIDResult.(userIDLookupFound)
	if !userIDMatched {
		rejected := userIDResult.(userIDLookupRejected)
		return org.ProvisionMemberStoreRejected{Reason: rejected.reason}
	}

	_, err = tx.Exec(ctx, "insert into organization_memberships (id, organization_id, user_id, status) values ($1, $2, $3, $4)", membershipID.String(), organizationID.String(), userIDFound.value.String(), org.MembershipStatusActive.String())
	if err != nil {
		if isUniqueViolation(err) {
			return org.ProvisionMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "user is already a member of this organization")}
		}
		return org.ProvisionMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert organization membership failed")}
	}

	for _, role := range roles {
		if _, err := tx.Exec(ctx, "insert into organization_membership_roles (membership_id, role) values ($1, $2)", membershipID.String(), role.String()); err != nil {
			return org.ProvisionMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert organization member role failed")}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return org.ProvisionMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit provision member transaction failed")}
	}

	return org.MemberProvisioned{
		Value: org.OrganizationMember{
			ID:             membershipID,
			OrganizationID: organizationID,
			UserID:         userIDFound.value,
			Status:         org.MembershipStatusActive,
			Roles:          roles,
		},
	}
}

func (store OrgStore) ListMembers(ctx context.Context, organizationID core.OrganizationID, page core.Page) org.ListMembersResult {
	rows, err := store.db.Query(ctx, `
		select organization_memberships.id::text, organization_memberships.user_id::text, organization_memberships.status,
			coalesce(array_agg(organization_membership_roles.role) filter (where organization_membership_roles.role is not null), '{}')
		from organization_memberships
		left join organization_membership_roles on organization_membership_roles.membership_id = organization_memberships.id
		where organization_memberships.organization_id = $1 and organization_memberships.status <> $2
		group by organization_memberships.id, organization_memberships.user_id, organization_memberships.status
		order by organization_memberships.user_id
		limit $3 offset $4
	`, organizationID.String(), org.MembershipStatusRemoved.String(), page.Limit(), page.Offset())
	if err != nil {
		return org.ListMembersRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list organization members failed")}
	}
	defer rows.Close()

	values := make([]org.OrganizationMember, 0)
	for rows.Next() {
		var rawID string
		var rawUserID string
		var rawStatus string
		var rawRoles []string
		if err := rows.Scan(&rawID, &rawUserID, &rawStatus, &rawRoles); err != nil {
			return org.ListMembersRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan organization member failed")}
		}
		memberResult := parseMemberRow(rawID, organizationID, rawUserID, rawStatus, rawRoles)
		member, matched := memberResult.(memberRowAccepted)
		if !matched {
			return org.ListMembersRejected{Reason: memberResult.(memberRowRejected).reason}
		}
		values = append(values, member.value)
	}
	if err := rows.Err(); err != nil {
		return org.ListMembersRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read organization members failed")}
	}
	return org.MembersListed{Values: values}
}

type memberRowResult interface {
	memberRowResult()
}

type memberRowAccepted struct {
	value org.OrganizationMember
}

type memberRowRejected struct {
	reason core.DomainError
}

func (memberRowAccepted) memberRowResult() {}

func (memberRowRejected) memberRowResult() {}

func parseMemberRow(rawID string, organizationID core.OrganizationID, rawUserID string, rawStatus string, rawRoles []string) memberRowResult {
	idResult := core.ParseOrganizationMembershipID(rawID)
	idCreated, idMatched := idResult.(core.OrganizationMembershipIDCreated)
	if !idMatched {
		return memberRowRejected{reason: idResult.(core.OrganizationMembershipIDRejected).Reason}
	}

	userResult := core.ParseUserID(rawUserID)
	userCreated, userMatched := userResult.(core.UserIDCreated)
	if !userMatched {
		return memberRowRejected{reason: userResult.(core.UserIDRejected).Reason}
	}

	statusResult := org.ParseMembershipStatus(rawStatus)
	statusAccepted, statusMatched := statusResult.(org.MembershipStatusAccepted)
	if !statusMatched {
		return memberRowRejected{reason: statusResult.(org.MembershipStatusRejected).Reason}
	}

	roles := make([]org.Role, 0, len(rawRoles))
	for _, rawRole := range rawRoles {
		roleResult := org.ParseRole(rawRole)
		roleAccepted, roleMatched := roleResult.(org.RoleAccepted)
		if !roleMatched {
			return memberRowRejected{reason: roleResult.(org.RoleRejected).Reason}
		}
		roles = append(roles, roleAccepted.Value)
	}

	return memberRowAccepted{value: org.OrganizationMember{ID: idCreated.Value, OrganizationID: organizationID, UserID: userCreated.Value, Status: statusAccepted.Value, Roles: roles}}
}

func (store OrgStore) DeactivateMember(ctx context.Context, organizationID core.OrganizationID, userID core.UserID) org.DeactivateMemberStoreResult {
	commandTag, err := store.db.Exec(ctx, `
		update organization_memberships
		set status = $3, status_recorded_at = now()
		where organization_id = $1
			and user_id = $2
			and status = $4
	`, organizationID.String(), userID.String(), org.MembershipStatusDeactivated.String(), org.MembershipStatusActive.String())
	if err != nil {
		return org.DeactivateMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "deactivate member failed")}
	}
	if commandTag == 0 {
		return org.DeactivateMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "active organization member was not found")}
	}
	return org.MemberDeactivated{}
}

func (store OrgStore) UpdateMemberRoles(ctx context.Context, organizationID core.OrganizationID, userID core.UserID, roles []org.Role) org.UpdateMemberRolesStoreResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return org.UpdateMemberRolesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin update member roles transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var membershipID string
	err = tx.QueryRow(ctx, `
		select id::text
		from organization_memberships
		where organization_id = $1 and user_id = $2 and status = $3
		for update
	`, organizationID.String(), userID.String(), org.MembershipStatusActive.String()).Scan(&membershipID)
	if err != nil {
		if errors.Is(err, ErrNoRows) {
			return org.UpdateMemberRolesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "active organization member was not found")}
		}
		return org.UpdateMemberRolesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "find organization member failed")}
	}

	if _, err := tx.Exec(ctx, "delete from organization_membership_roles where membership_id = $1", membershipID); err != nil {
		return org.UpdateMemberRolesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "delete organization member roles failed")}
	}
	for _, role := range roles {
		if _, err := tx.Exec(ctx, "insert into organization_membership_roles (membership_id, role) values ($1, $2)", membershipID, role.String()); err != nil {
			return org.UpdateMemberRolesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert organization member role failed")}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return org.UpdateMemberRolesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit update member roles transaction failed")}
	}

	return store.findActiveMember(ctx, organizationID, userID)
}

func (store OrgStore) findActiveMember(ctx context.Context, organizationID core.OrganizationID, userID core.UserID) org.UpdateMemberRolesStoreResult {
	var rawID string
	var rawUserID string
	var rawStatus string
	var rawRoles []string
	err := store.db.QueryRow(ctx, `
		select organization_memberships.id::text, organization_memberships.user_id::text, organization_memberships.status,
			coalesce(array_agg(organization_membership_roles.role) filter (where organization_membership_roles.role is not null), '{}')
		from organization_memberships
		left join organization_membership_roles on organization_membership_roles.membership_id = organization_memberships.id
		where organization_memberships.organization_id = $1
			and organization_memberships.user_id = $2
			and organization_memberships.status = $3
		group by organization_memberships.id, organization_memberships.user_id, organization_memberships.status
	`, organizationID.String(), userID.String(), org.MembershipStatusActive.String()).Scan(&rawID, &rawUserID, &rawStatus, &rawRoles)
	if err != nil {
		if errors.Is(err, ErrNoRows) {
			return org.UpdateMemberRolesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "active organization member was not found")}
		}
		return org.UpdateMemberRolesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "find updated organization member failed")}
	}
	memberResult := parseMemberRow(rawID, organizationID, rawUserID, rawStatus, rawRoles)
	member, matched := memberResult.(memberRowAccepted)
	if !matched {
		return org.UpdateMemberRolesStoreRejected{Reason: memberResult.(memberRowRejected).reason}
	}
	return org.MemberRolesUpdated{Value: member.value}
}

func (store OrgStore) CreateOrganizationTeam(ctx context.Context, teamID core.TeamID, organizationID core.OrganizationID, name org.TeamName, createdBy core.UserID) org.CreateTeamStoreResult {
	_, err := store.db.Exec(ctx, "insert into teams (id, name, owner_kind, organization_id, created_by_user_id) values ($1, $2, 'organization', $3, $4)", teamID.String(), name.String(), organizationID.String(), createdBy.String())
	if err != nil {
		return org.CreateTeamStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert organization team failed")}
	}
	return org.CreateTeamStoreAccepted{}
}

func (store OrgStore) AddTeamMember(ctx context.Context, teamID core.TeamID, userID core.UserID) org.AddTeamMemberStoreResult {
	_, err := store.db.Exec(ctx, "insert into team_members (team_id, user_id) values ($1, $2) on conflict do nothing", teamID.String(), userID.String())
	if err != nil {
		return org.AddTeamMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert team member failed")}
	}
	return org.TeamMemberAdded{}
}

func (store OrgStore) AddTeamMemberByEmail(ctx context.Context, teamID core.TeamID, email auth.EmailAddress) org.AddTeamMemberStoreResult {
	var rawUserID string
	err := store.db.QueryRow(ctx, "select id::text from users where email = $1", email.String()).Scan(&rawUserID)
	if err != nil {
		if errors.Is(err, ErrNoRows) {
			return org.AddTeamMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "user was not found for email address")}
		}
		return org.AddTeamMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "lookup user by email failed")}
	}

	userIDResult := core.ParseUserID(rawUserID)
	userIDCreated, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		return org.AddTeamMemberStoreRejected{Reason: userIDResult.(core.UserIDRejected).Reason}
	}
	return store.AddTeamMember(ctx, teamID, userIDCreated.Value)
}

func (store OrgStore) ListOrganizationTeams(ctx context.Context, organizationID core.OrganizationID, userID core.UserID, query string, page core.Page) org.TeamListResult {
	rolesResult := store.FindMemberRoles(ctx, organizationID, userID)
	if _, matched := rolesResult.(org.MemberRolesFound); !matched {
		return org.TeamListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "organization team access denied")}
	}

	rows, err := store.db.Query(ctx, `
		select id::text, owner_kind, coalesce(organization_id::text, ''), coalesce(owner_user_id::text, ''), name, created_by_user_id::text
		from teams
		where organization_id = $1
		and ($2 = '' or lower(name) like '%' || lower($2) || '%')
		order by name
		limit $3 offset $4
	`, organizationID.String(), query, page.Limit(), page.Offset())
	if err != nil {
		return org.TeamListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list organization teams failed")}
	}
	return scanTeamRows(rows, "read organization teams failed")
}

// CreateStandaloneTeam stores a team owned directly by a user.
func (store OrgStore) CreateStandaloneTeam(ctx context.Context, teamID core.TeamID, ownerUserID core.UserID, name org.TeamName) org.CreateTeamStoreResult {
	_, err := store.db.Exec(ctx, "insert into teams (id, name, owner_kind, owner_user_id, created_by_user_id) values ($1, $2, 'user', $3, $3)", teamID.String(), name.String(), ownerUserID.String())
	if err != nil {
		return org.CreateTeamStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert standalone team failed")}
	}
	return org.CreateTeamStoreAccepted{}
}

// ListStandaloneTeams lists the user-owned teams for the given user.
func (store OrgStore) ListStandaloneTeams(ctx context.Context, ownerUserID core.UserID, query string, page core.Page) org.TeamListResult {
	rows, err := store.db.Query(ctx, `
		select id::text, owner_kind, coalesce(organization_id::text, ''), coalesce(owner_user_id::text, ''), name, created_by_user_id::text
		from teams
		where owner_kind = 'user' and owner_user_id = $1
		and ($2 = '' or lower(name) like '%' || lower($2) || '%')
		order by name
		limit $3 offset $4
	`, ownerUserID.String(), query, page.Limit(), page.Offset())
	if err != nil {
		return org.TeamListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list standalone teams failed")}
	}
	return scanTeamRows(rows, "read standalone teams failed")
}

func (store OrgStore) FindTeam(ctx context.Context, teamID core.TeamID) org.FindTeamResult {
	var rawID string
	var rawOwnerKind string
	var rawOrganizationID string
	var rawOwnerUserID string
	var rawName string
	var rawCreatedBy string
	err := store.db.QueryRow(ctx, `
		select id::text, owner_kind, coalesce(organization_id::text, ''), coalesce(owner_user_id::text, ''), name, created_by_user_id::text
		from teams
		where id = $1
	`, teamID.String()).Scan(&rawID, &rawOwnerKind, &rawOrganizationID, &rawOwnerUserID, &rawName, &rawCreatedBy)
	if err != nil {
		if errors.Is(err, ErrNoRows) {
			return org.TeamMissing{Reason: core.NewDomainError(core.ErrorCodeNotFound, "team not found")}
		}
		return org.TeamMissing{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "find team failed")}
	}

	parsed := parseTeamRow(rawID, rawOwnerKind, rawOrganizationID, rawOwnerUserID, rawName, rawCreatedBy)
	accepted, matched := parsed.(teamRowAccepted)
	if !matched {
		return org.TeamMissing{Reason: parsed.(teamRowRejected).reason}
	}
	return org.TeamFound{Value: accepted.value}
}

func (store OrgStore) ListTeamMembers(ctx context.Context, teamID core.TeamID) org.TeamMembersResult {
	rows, err := store.db.Query(ctx, "select user_id::text from team_members where team_id = $1 order by user_id", teamID.String())
	if err != nil {
		return org.TeamMembersRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list team members failed")}
	}
	defer rows.Close()

	values := make([]core.UserID, 0)
	for rows.Next() {
		var rawUserID string
		if err := rows.Scan(&rawUserID); err != nil {
			return org.TeamMembersRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan team member failed")}
		}
		userIDResult := core.ParseUserID(rawUserID)
		userIDCreated, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			return org.TeamMembersRejected{Reason: userIDResult.(core.UserIDRejected).Reason}
		}
		values = append(values, userIDCreated.Value)
	}
	if err := rows.Err(); err != nil {
		return org.TeamMembersRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read team members failed")}
	}
	return org.TeamMembersListed{Values: values}
}

func scanTeamRows(rows Rows, readErrorMessage string) org.TeamListResult {
	defer rows.Close()

	values := make([]org.Team, 0)
	for rows.Next() {
		var rawID string
		var rawOwnerKind string
		var rawOrganizationID string
		var rawOwnerUserID string
		var rawName string
		var rawCreatedBy string
		if err := rows.Scan(&rawID, &rawOwnerKind, &rawOrganizationID, &rawOwnerUserID, &rawName, &rawCreatedBy); err != nil {
			return org.TeamListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan team failed")}
		}
		parsed := parseTeamRow(rawID, rawOwnerKind, rawOrganizationID, rawOwnerUserID, rawName, rawCreatedBy)
		accepted, matched := parsed.(teamRowAccepted)
		if !matched {
			rejected := parsed.(teamRowRejected)
			return org.TeamListRejected{Reason: rejected.reason}
		}
		values = append(values, accepted.value)
	}

	if err := rows.Err(); err != nil {
		return org.TeamListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, readErrorMessage)}
	}

	return org.TeamsListed{Values: values}
}

type organizationRowResult interface {
	organizationRowResult()
}

type organizationRowAccepted struct {
	value org.Organization
}

type organizationRowRejected struct {
	reason core.DomainError
}

func (organizationRowAccepted) organizationRowResult() {}

func (organizationRowRejected) organizationRowResult() {}

func parseOrganizationRow(rawID string, rawName string, rawCreatedBy string) organizationRowResult {
	idResult := core.ParseOrganizationID(rawID)
	idCreated, idMatched := idResult.(core.OrganizationIDCreated)
	if !idMatched {
		rejected := idResult.(core.OrganizationIDRejected)
		return organizationRowRejected{reason: rejected.Reason}
	}

	nameResult := org.NewOrganizationName(rawName)
	nameAccepted, nameMatched := nameResult.(org.OrganizationNameAccepted)
	if !nameMatched {
		rejected := nameResult.(org.OrganizationNameRejected)
		return organizationRowRejected{reason: rejected.Reason}
	}

	userResult := core.ParseUserID(rawCreatedBy)
	userCreated, userMatched := userResult.(core.UserIDCreated)
	if !userMatched {
		rejected := userResult.(core.UserIDRejected)
		return organizationRowRejected{reason: rejected.Reason}
	}

	return organizationRowAccepted{value: org.Organization{ID: idCreated.Value, Name: nameAccepted.Value, CreatedBy: userCreated.Value}}
}

type teamRowResult interface {
	teamRowResult()
}

type teamRowAccepted struct {
	value org.Team
}

type teamRowRejected struct {
	reason core.DomainError
}

func (teamRowAccepted) teamRowResult() {}

func (teamRowRejected) teamRowResult() {}

func parseTeamRow(rawID string, rawOwnerKind string, rawOrganizationID string, rawOwnerUserID string, rawName string, rawCreatedBy string) teamRowResult {
	idResult := core.ParseTeamID(rawID)
	idCreated, idMatched := idResult.(core.TeamIDCreated)
	if !idMatched {
		rejected := idResult.(core.TeamIDRejected)
		return teamRowRejected{reason: rejected.Reason}
	}

	nameResult := org.NewTeamName(rawName)
	nameAccepted, nameMatched := nameResult.(org.TeamNameAccepted)
	if !nameMatched {
		rejected := nameResult.(org.TeamNameRejected)
		return teamRowRejected{reason: rejected.Reason}
	}

	userResult := core.ParseUserID(rawCreatedBy)
	userCreated, userMatched := userResult.(core.UserIDCreated)
	if !userMatched {
		rejected := userResult.(core.UserIDRejected)
		return teamRowRejected{reason: rejected.Reason}
	}

	ownerResult := parseTeamOwner(rawOwnerKind, rawOrganizationID, rawOwnerUserID)
	owner, ownerMatched := ownerResult.(teamOwnerAccepted)
	if !ownerMatched {
		return teamRowRejected{reason: ownerResult.(teamOwnerRejected).reason}
	}

	return teamRowAccepted{value: org.Team{ID: idCreated.Value, Owner: owner.value, Name: nameAccepted.Value, CreatedBy: userCreated.Value}}
}

type teamOwnerResult interface {
	teamOwnerResult()
}

type teamOwnerAccepted struct {
	value org.TeamOwner
}

type teamOwnerRejected struct {
	reason core.DomainError
}

func (teamOwnerAccepted) teamOwnerResult() {}

func (teamOwnerRejected) teamOwnerResult() {}

func parseTeamOwner(rawOwnerKind string, rawOrganizationID string, rawOwnerUserID string) teamOwnerResult {
	switch rawOwnerKind {
	case org.TeamOwnerKindOrganization.String():
		organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
		organizationIDCreated, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			return teamOwnerRejected{reason: organizationIDResult.(core.OrganizationIDRejected).Reason}
		}
		return teamOwnerAccepted{value: org.OrganizationOwnedTeam{OrganizationID: organizationIDCreated.Value}}
	case org.TeamOwnerKindUser.String():
		ownerUserIDResult := core.ParseUserID(rawOwnerUserID)
		ownerUserIDCreated, matched := ownerUserIDResult.(core.UserIDCreated)
		if !matched {
			return teamOwnerRejected{reason: ownerUserIDResult.(core.UserIDRejected).Reason}
		}
		return teamOwnerAccepted{value: org.UserOwnedTeam{OwnerUserID: ownerUserIDCreated.Value}}
	default:
		return teamOwnerRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "team owner kind is invalid")}
	}
}

type userIDLookupResult interface {
	userIDLookupResult()
}

type userIDLookupFound struct {
	value core.UserID
}

type userIDLookupRejected struct {
	reason core.DomainError
}

func (userIDLookupFound) userIDLookupResult() {}

func (userIDLookupRejected) userIDLookupResult() {}

func lookupUserIDByEmail(ctx context.Context, tx Tx, email auth.EmailAddress) userIDLookupResult {
	var rawUserID string
	err := tx.QueryRow(ctx, "select id::text from users where email = $1", email.String()).Scan(&rawUserID)
	if err != nil {
		if errors.Is(err, ErrNoRows) {
			return userIDLookupRejected{reason: core.NewDomainError(core.ErrorCodeNotFound, "user was not found for email address")}
		}
		return userIDLookupRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lookup user by email failed")}
	}

	userIDResult := core.ParseUserID(rawUserID)
	userIDCreated, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		rejected := userIDResult.(core.UserIDRejected)
		return userIDLookupRejected{reason: rejected.Reason}
	}

	return userIDLookupFound{value: userIDCreated.Value}
}
