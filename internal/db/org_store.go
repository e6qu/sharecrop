package db

import (
	"context"
	"errors"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrgStore struct {
	pool *pgxpool.Pool
}

func NewOrgStore(pool *pgxpool.Pool) OrgStore {
	return OrgStore{pool: pool}
}

func (store OrgStore) CreateOrganization(ctx context.Context, organizationID core.OrganizationID, name org.OrganizationName, createdBy core.UserID, membershipID core.OrganizationMembershipID) org.CreateOrganizationStoreResult {
	tx, err := store.pool.Begin(ctx)
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

	if err := tx.Commit(ctx); err != nil {
		return org.CreateOrganizationStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit create organization transaction failed")}
	}

	return org.CreateOrganizationStoreAccepted{}
}

func (store OrgStore) ListOrganizationsForUser(ctx context.Context, userID core.UserID) org.ListOrganizationsResult {
	rows, err := store.pool.Query(ctx, `
		select organizations.id::text, organizations.name, organizations.created_by_user_id::text
		from organizations
		join organization_memberships on organization_memberships.organization_id = organizations.id
		where organization_memberships.user_id = $1
			and organization_memberships.status = $2
		order by organizations.name
	`, userID.String(), org.MembershipStatusActive.String())
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
	rows, err := store.pool.Query(ctx, `
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
	tx, err := store.pool.Begin(ctx)
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

func (store OrgStore) DeactivateMember(ctx context.Context, organizationID core.OrganizationID, userID core.UserID) org.DeactivateMemberStoreResult {
	commandTag, err := store.pool.Exec(ctx, `
		update organization_memberships
		set status = $3, status_recorded_at = now()
		where organization_id = $1
			and user_id = $2
			and status = $4
	`, organizationID.String(), userID.String(), org.MembershipStatusDeactivated.String(), org.MembershipStatusActive.String())
	if err != nil {
		return org.DeactivateMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "deactivate member failed")}
	}
	if commandTag.RowsAffected() == 0 {
		return org.DeactivateMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "active organization member was not found")}
	}
	return org.MemberDeactivated{}
}

func (store OrgStore) CreateOrganizationTeam(ctx context.Context, teamID core.TeamID, organizationID core.OrganizationID, name org.TeamName, createdBy core.UserID) org.CreateTeamStoreResult {
	_, err := store.pool.Exec(ctx, "insert into teams (id, name, owner_kind, organization_id, created_by_user_id) values ($1, $2, 'organization', $3, $4)", teamID.String(), name.String(), organizationID.String(), createdBy.String())
	if err != nil {
		return org.CreateTeamStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert organization team failed")}
	}
	return org.CreateTeamStoreAccepted{}
}

func (store OrgStore) AddTeamMember(ctx context.Context, teamID core.TeamID, userID core.UserID) org.AddTeamMemberStoreResult {
	_, err := store.pool.Exec(ctx, "insert into team_members (team_id, user_id) values ($1, $2) on conflict do nothing", teamID.String(), userID.String())
	if err != nil {
		return org.AddTeamMemberStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert team member failed")}
	}
	return org.TeamMemberAdded{}
}

func (store OrgStore) ListOrganizationTeams(ctx context.Context, organizationID core.OrganizationID, userID core.UserID) org.TeamListResult {
	rolesResult := store.FindMemberRoles(ctx, organizationID, userID)
	if _, matched := rolesResult.(org.MemberRolesFound); !matched {
		return org.TeamListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "organization team access denied")}
	}

	rows, err := store.pool.Query(ctx, `
		select id::text, organization_id::text, name, created_by_user_id::text
		from teams
		where organization_id = $1
		order by name
	`, organizationID.String())
	if err != nil {
		return org.TeamListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list organization teams failed")}
	}
	defer rows.Close()

	values := make([]org.Team, 0)
	for rows.Next() {
		var rawID string
		var rawOrganizationID string
		var rawName string
		var rawCreatedBy string
		if err := rows.Scan(&rawID, &rawOrganizationID, &rawName, &rawCreatedBy); err != nil {
			return org.TeamListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan team failed")}
		}
		parsed := parseTeamRow(rawID, rawOrganizationID, rawName, rawCreatedBy)
		accepted, matched := parsed.(teamRowAccepted)
		if !matched {
			rejected := parsed.(teamRowRejected)
			return org.TeamListRejected{Reason: rejected.reason}
		}
		values = append(values, accepted.value)
	}

	if err := rows.Err(); err != nil {
		return org.TeamListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read organization teams failed")}
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

func parseTeamRow(rawID string, rawOrganizationID string, rawName string, rawCreatedBy string) teamRowResult {
	idResult := core.ParseTeamID(rawID)
	idCreated, idMatched := idResult.(core.TeamIDCreated)
	if !idMatched {
		rejected := idResult.(core.TeamIDRejected)
		return teamRowRejected{reason: rejected.Reason}
	}

	organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
	organizationIDCreated, organizationIDMatched := organizationIDResult.(core.OrganizationIDCreated)
	if !organizationIDMatched {
		rejected := organizationIDResult.(core.OrganizationIDRejected)
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

	return teamRowAccepted{value: org.Team{ID: idCreated.Value, OrganizationID: organizationIDCreated.Value, Name: nameAccepted.Value, CreatedBy: userCreated.Value}}
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

func lookupUserIDByEmail(ctx context.Context, tx pgx.Tx, email auth.EmailAddress) userIDLookupResult {
	var rawUserID string
	err := tx.QueryRow(ctx, "select id::text from users where email = $1", email.String()).Scan(&rawUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return userIDLookupRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "user was not found for email address")}
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
