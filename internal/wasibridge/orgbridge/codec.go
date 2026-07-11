// Package orgbridge is the WASI bridge for internal/org's Store (organizations,
// members, and teams): hand-written per-type codecs (this file) plus a generated
// dispatcher and guest client (bridge_gen.go). Shared core types (ids, page) are
// serialized by internal/wasibridge/corewire; the domain error by
// internal/wasibridge/domainwire.
package orgbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- value types (string wrappers) ----

func encodeOrganizationName(name org.OrganizationName) string { return name.String() }

func decodeOrganizationName(raw string) (org.OrganizationName, error) {
	accepted, matched := org.NewOrganizationName(raw).(org.OrganizationNameAccepted)
	if !matched {
		return org.OrganizationName{}, fmt.Errorf("invalid organization name")
	}
	return accepted.Value, nil
}

func encodeTeamName(name org.TeamName) string { return name.String() }

func decodeTeamName(raw string) (org.TeamName, error) {
	accepted, matched := org.NewTeamName(raw).(org.TeamNameAccepted)
	if !matched {
		return org.TeamName{}, fmt.Errorf("invalid team name")
	}
	return accepted.Value, nil
}

func encodeEmail(email auth.EmailAddress) string { return email.String() }

func decodeEmail(raw string) (auth.EmailAddress, error) {
	accepted, matched := auth.NewEmailAddress(raw).(auth.EmailAddressAccepted)
	if !matched {
		return auth.EmailAddress{}, fmt.Errorf("invalid member email address %q", raw)
	}
	return accepted.Value, nil
}

func encodeRole(role org.Role) string { return role.String() }

func decodeRole(raw string) (org.Role, error) {
	accepted, matched := org.ParseRole(raw).(org.RoleAccepted)
	if !matched {
		return org.Role{}, fmt.Errorf("invalid organization role %q", raw)
	}
	return accepted.Value, nil
}

func encodeRoles(roles []org.Role) []string {
	encoded := make([]string, 0, len(roles))
	for index := range roles {
		encoded = append(encoded, encodeRole(roles[index]))
	}
	return encoded
}

func decodeRoles(raw []string) ([]org.Role, error) {
	roles := make([]org.Role, 0, len(raw))
	for index := range raw {
		role, err := decodeRole(raw[index])
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func decodeMembershipStatus(raw string) (org.MembershipStatus, error) {
	accepted, matched := org.ParseMembershipStatus(raw).(org.MembershipStatusAccepted)
	if !matched {
		return org.MembershipStatus{}, fmt.Errorf("invalid membership status %q", raw)
	}
	return accepted.Value, nil
}

func encodeUserIDs(ids []core.UserID) []string {
	encoded := make([]string, 0, len(ids))
	for index := range ids {
		encoded = append(encoded, corewire.EncodeUserID(ids[index]))
	}
	return encoded
}

func decodeUserIDs(raw []string) ([]core.UserID, error) {
	ids := make([]core.UserID, 0, len(raw))
	for index := range raw {
		id, err := corewire.DecodeUserID(raw[index])
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// ---- org.Organization ----

type organizationWire struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
}

func encodeOrganization(value org.Organization) organizationWire {
	return organizationWire{
		ID:        corewire.EncodeOrganizationID(value.ID),
		Name:      encodeOrganizationName(value.Name),
		CreatedBy: corewire.EncodeUserID(value.CreatedBy),
	}
}

func decodeOrganization(wire organizationWire) (org.Organization, error) {
	id, err := corewire.DecodeOrganizationID(wire.ID)
	if err != nil {
		return org.Organization{}, err
	}
	name, err := decodeOrganizationName(wire.Name)
	if err != nil {
		return org.Organization{}, err
	}
	createdBy, err := corewire.DecodeUserID(wire.CreatedBy)
	if err != nil {
		return org.Organization{}, err
	}
	return org.Organization{ID: id, Name: name, CreatedBy: createdBy}, nil
}

func encodeOrganizations(values []org.Organization) []organizationWire {
	encoded := make([]organizationWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeOrganization(values[index]))
	}
	return encoded
}

func decodeOrganizations(wires []organizationWire) ([]org.Organization, error) {
	values := make([]org.Organization, 0, len(wires))
	for index := range wires {
		value, err := decodeOrganization(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// ---- org.OrganizationMember ----

type organizationMemberWire struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	UserID         string   `json:"user_id"`
	Status         string   `json:"status"`
	Roles          []string `json:"roles,omitempty"`
}

func encodeMember(member org.OrganizationMember) organizationMemberWire {
	return organizationMemberWire{
		ID:             corewire.EncodeOrganizationMembershipID(member.ID),
		OrganizationID: corewire.EncodeOrganizationID(member.OrganizationID),
		UserID:         corewire.EncodeUserID(member.UserID),
		Status:         member.Status.String(),
		Roles:          encodeRoles(member.Roles),
	}
}

func decodeMember(wire organizationMemberWire) (org.OrganizationMember, error) {
	id, err := corewire.DecodeOrganizationMembershipID(wire.ID)
	if err != nil {
		return org.OrganizationMember{}, err
	}
	organizationID, err := corewire.DecodeOrganizationID(wire.OrganizationID)
	if err != nil {
		return org.OrganizationMember{}, err
	}
	userID, err := corewire.DecodeUserID(wire.UserID)
	if err != nil {
		return org.OrganizationMember{}, err
	}
	status, err := decodeMembershipStatus(wire.Status)
	if err != nil {
		return org.OrganizationMember{}, err
	}
	roles, err := decodeRoles(wire.Roles)
	if err != nil {
		return org.OrganizationMember{}, err
	}
	return org.OrganizationMember{ID: id, OrganizationID: organizationID, UserID: userID, Status: status, Roles: roles}, nil
}

func encodeMembers(values []org.OrganizationMember) []organizationMemberWire {
	encoded := make([]organizationMemberWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeMember(values[index]))
	}
	return encoded
}

func decodeMembers(wires []organizationMemberWire) ([]org.OrganizationMember, error) {
	values := make([]org.OrganizationMember, 0, len(wires))
	for index := range wires {
		value, err := decodeMember(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func decodeMemberPayload(wire *organizationMemberWire) (org.OrganizationMember, error) {
	if wire == nil {
		return org.OrganizationMember{}, fmt.Errorf("result is missing its member")
	}
	return decodeMember(*wire)
}

// ---- org.Team ----

type teamOwnerWire struct {
	Variant        string `json:"variant"`
	OrganizationID string `json:"organization_id,omitempty"`
	OwnerUserID    string `json:"owner_user_id,omitempty"`
}

func encodeTeamOwner(owner org.TeamOwner) teamOwnerWire {
	switch typed := owner.(type) {
	case org.OrganizationOwnedTeam:
		return teamOwnerWire{Variant: "organization", OrganizationID: corewire.EncodeOrganizationID(typed.OrganizationID)}
	case org.UserOwnedTeam:
		return teamOwnerWire{Variant: "user", OwnerUserID: corewire.EncodeUserID(typed.OwnerUserID)}
	default:
		return teamOwnerWire{Variant: "user"}
	}
}

func decodeTeamOwner(wire teamOwnerWire) (org.TeamOwner, error) {
	switch wire.Variant {
	case "organization":
		id, err := corewire.DecodeOrganizationID(wire.OrganizationID)
		if err != nil {
			return nil, err
		}
		return org.OrganizationOwnedTeam{OrganizationID: id}, nil
	case "user":
		id, err := corewire.DecodeUserID(wire.OwnerUserID)
		if err != nil {
			return nil, err
		}
		return org.UserOwnedTeam{OwnerUserID: id}, nil
	default:
		return nil, fmt.Errorf("unknown team owner variant %q", wire.Variant)
	}
}

type teamValueWire struct {
	ID        string        `json:"id"`
	Owner     teamOwnerWire `json:"owner"`
	Name      string        `json:"name"`
	CreatedBy string        `json:"created_by"`
}

func encodeTeam(team org.Team) teamValueWire {
	return teamValueWire{
		ID:        corewire.EncodeTeamID(team.ID),
		Owner:     encodeTeamOwner(team.Owner),
		Name:      encodeTeamName(team.Name),
		CreatedBy: corewire.EncodeUserID(team.CreatedBy),
	}
}

func decodeTeam(wire teamValueWire) (org.Team, error) {
	id, err := corewire.DecodeTeamID(wire.ID)
	if err != nil {
		return org.Team{}, err
	}
	owner, err := decodeTeamOwner(wire.Owner)
	if err != nil {
		return org.Team{}, err
	}
	name, err := decodeTeamName(wire.Name)
	if err != nil {
		return org.Team{}, err
	}
	createdBy, err := corewire.DecodeUserID(wire.CreatedBy)
	if err != nil {
		return org.Team{}, err
	}
	return org.Team{ID: id, Owner: owner, Name: name, CreatedBy: createdBy}, nil
}

func encodeTeams(values []org.Team) []teamValueWire {
	encoded := make([]teamValueWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeTeam(values[index]))
	}
	return encoded
}

func decodeTeams(wires []teamValueWire) ([]org.Team, error) {
	values := make([]org.Team, 0, len(wires))
	for index := range wires {
		value, err := decodeTeam(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// ---- accept/reject result unions ----

// acceptedRejectedWire backs every union that is just an accept/reject pair; the
// success arm carries no payload.
type acceptedRejectedWire struct {
	Variant string                  `json:"variant"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCreateOrganizationResult(result org.CreateOrganizationStoreResult) acceptedRejectedWire {
	rejected, matched := result.(org.CreateOrganizationStoreRejected)
	if !matched {
		return acceptedRejectedWire{Variant: "accepted"}
	}
	reason := domainwire.EncodeDomainError(rejected.Reason)
	return acceptedRejectedWire{Variant: "rejected", Error: &reason}
}

func decodeCreateOrganizationResult(wire acceptedRejectedWire) (org.CreateOrganizationStoreResult, error) {
	if wire.Variant == "rejected" {
		return org.CreateOrganizationStoreRejected{Reason: decodeReason(wire.Error)}, nil
	}
	return org.CreateOrganizationStoreAccepted{}, nil
}

func encodeDeactivateMemberResult(result org.DeactivateMemberStoreResult) acceptedRejectedWire {
	rejected, matched := result.(org.DeactivateMemberStoreRejected)
	if !matched {
		return acceptedRejectedWire{Variant: "accepted"}
	}
	reason := domainwire.EncodeDomainError(rejected.Reason)
	return acceptedRejectedWire{Variant: "rejected", Error: &reason}
}

func decodeDeactivateMemberResult(wire acceptedRejectedWire) (org.DeactivateMemberStoreResult, error) {
	if wire.Variant == "rejected" {
		return org.DeactivateMemberStoreRejected{Reason: decodeReason(wire.Error)}, nil
	}
	return org.MemberDeactivated{}, nil
}

func encodeCreateTeamResult(result org.CreateTeamStoreResult) acceptedRejectedWire {
	rejected, matched := result.(org.CreateTeamStoreRejected)
	if !matched {
		return acceptedRejectedWire{Variant: "accepted"}
	}
	reason := domainwire.EncodeDomainError(rejected.Reason)
	return acceptedRejectedWire{Variant: "rejected", Error: &reason}
}

func decodeCreateTeamResult(wire acceptedRejectedWire) (org.CreateTeamStoreResult, error) {
	if wire.Variant == "rejected" {
		return org.CreateTeamStoreRejected{Reason: decodeReason(wire.Error)}, nil
	}
	return org.CreateTeamStoreAccepted{}, nil
}

func encodeAddTeamMemberResult(result org.AddTeamMemberStoreResult) acceptedRejectedWire {
	rejected, matched := result.(org.AddTeamMemberStoreRejected)
	if !matched {
		return acceptedRejectedWire{Variant: "accepted"}
	}
	reason := domainwire.EncodeDomainError(rejected.Reason)
	return acceptedRejectedWire{Variant: "rejected", Error: &reason}
}

func decodeAddTeamMemberResult(wire acceptedRejectedWire) (org.AddTeamMemberStoreResult, error) {
	if wire.Variant == "rejected" {
		return org.AddTeamMemberStoreRejected{Reason: decodeReason(wire.Error)}, nil
	}
	return org.TeamMemberAdded{}, nil
}

// ---- payload-carrying result unions ----

type organizationsResultWire struct {
	Variant       string                  `json:"variant"`
	Organizations []organizationWire      `json:"organizations,omitempty"`
	Error         *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListOrganizationsResult(result org.ListOrganizationsResult) organizationsResultWire {
	switch typed := result.(type) {
	case org.OrganizationsListed:
		return organizationsResultWire{Variant: "listed", Organizations: encodeOrganizations(typed.Values)}
	case org.ListOrganizationsRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return organizationsResultWire{Variant: "rejected", Error: &reason}
	default:
		return organizationsResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown org result %T", result))}
	}
}

func decodeListOrganizationsResult(wire organizationsResultWire) (org.ListOrganizationsResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeOrganizations(wire.Organizations)
		if err != nil {
			return nil, err
		}
		return org.OrganizationsListed{Values: values}, nil
	case "rejected":
		return org.ListOrganizationsRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list organizations result variant %q", wire.Variant)
	}
}

type memberRolesResultWire struct {
	Variant string                  `json:"variant"`
	Roles   []string                `json:"roles,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeMemberRolesResult(result org.MemberRolesResult) memberRolesResultWire {
	switch typed := result.(type) {
	case org.MemberRolesFound:
		return memberRolesResultWire{Variant: "found", Roles: encodeRoles(typed.Roles)}
	case org.MemberRolesMissing:
		return memberRolesResultWire{Variant: "missing"}
	case org.MemberRolesRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return memberRolesResultWire{Variant: "rejected", Error: &reason}
	default:
		return memberRolesResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown org result %T", result))}
	}
}

func decodeMemberRolesResult(wire memberRolesResultWire) (org.MemberRolesResult, error) {
	switch wire.Variant {
	case "found":
		roles, err := decodeRoles(wire.Roles)
		if err != nil {
			return nil, err
		}
		return org.MemberRolesFound{Roles: roles}, nil
	case "missing":
		return org.MemberRolesMissing{}, nil
	case "rejected":
		return org.MemberRolesRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown member roles result variant %q", wire.Variant)
	}
}

type membersResultWire struct {
	Variant string                   `json:"variant"`
	Members []organizationMemberWire `json:"members,omitempty"`
	Error   *domainwire.DomainError  `json:"error,omitempty"`
}

func encodeListMembersResult(result org.ListMembersResult) membersResultWire {
	switch typed := result.(type) {
	case org.MembersListed:
		return membersResultWire{Variant: "listed", Members: encodeMembers(typed.Values)}
	case org.ListMembersRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return membersResultWire{Variant: "rejected", Error: &reason}
	default:
		return membersResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown org result %T", result))}
	}
}

func decodeListMembersResult(wire membersResultWire) (org.ListMembersResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeMembers(wire.Members)
		if err != nil {
			return nil, err
		}
		return org.MembersListed{Values: values}, nil
	case "rejected":
		return org.ListMembersRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list members result variant %q", wire.Variant)
	}
}

// memberResultWire backs the provision and update-roles results, which each
// carry one organization member on success.
type memberResultWire struct {
	Variant string                  `json:"variant"`
	Member  *organizationMemberWire `json:"member,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeProvisionMemberResult(result org.ProvisionMemberStoreResult) memberResultWire {
	switch typed := result.(type) {
	case org.MemberProvisioned:
		member := encodeMember(typed.Value)
		return memberResultWire{Variant: "provisioned", Member: &member}
	case org.ProvisionMemberStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return memberResultWire{Variant: "rejected", Error: &reason}
	default:
		return memberResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown org result %T", result))}
	}
}

func decodeProvisionMemberResult(wire memberResultWire) (org.ProvisionMemberStoreResult, error) {
	switch wire.Variant {
	case "provisioned":
		member, err := decodeMemberPayload(wire.Member)
		if err != nil {
			return nil, err
		}
		return org.MemberProvisioned{Value: member}, nil
	case "rejected":
		return org.ProvisionMemberStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown provision member result variant %q", wire.Variant)
	}
}

func encodeUpdateMemberRolesResult(result org.UpdateMemberRolesStoreResult) memberResultWire {
	switch typed := result.(type) {
	case org.MemberRolesUpdated:
		member := encodeMember(typed.Value)
		return memberResultWire{Variant: "updated", Member: &member}
	case org.UpdateMemberRolesStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return memberResultWire{Variant: "rejected", Error: &reason}
	default:
		return memberResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown org result %T", result))}
	}
}

func decodeUpdateMemberRolesResult(wire memberResultWire) (org.UpdateMemberRolesStoreResult, error) {
	switch wire.Variant {
	case "updated":
		member, err := decodeMemberPayload(wire.Member)
		if err != nil {
			return nil, err
		}
		return org.MemberRolesUpdated{Value: member}, nil
	case "rejected":
		return org.UpdateMemberRolesStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown update member roles result variant %q", wire.Variant)
	}
}

type teamsResultWire struct {
	Variant string                  `json:"variant"`
	Teams   []teamValueWire         `json:"teams,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeTeamListResult(result org.TeamListResult) teamsResultWire {
	switch typed := result.(type) {
	case org.TeamsListed:
		return teamsResultWire{Variant: "listed", Teams: encodeTeams(typed.Values)}
	case org.TeamListRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return teamsResultWire{Variant: "rejected", Error: &reason}
	default:
		return teamsResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown org result %T", result))}
	}
}

func decodeTeamListResult(wire teamsResultWire) (org.TeamListResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeTeams(wire.Teams)
		if err != nil {
			return nil, err
		}
		return org.TeamsListed{Values: values}, nil
	case "rejected":
		return org.TeamListRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown team list result variant %q", wire.Variant)
	}
}

type findTeamResultWire struct {
	Variant string                  `json:"variant"`
	Team    *teamValueWire          `json:"team,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeFindTeamResult(result org.FindTeamResult) findTeamResultWire {
	switch typed := result.(type) {
	case org.TeamFound:
		team := encodeTeam(typed.Value)
		return findTeamResultWire{Variant: "found", Team: &team}
	case org.TeamMissing:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return findTeamResultWire{Variant: "missing", Error: &reason}
	default:
		return findTeamResultWire{Variant: "missing", Error: rejectionError(fmt.Sprintf("unknown org result %T", result))}
	}
}

func decodeFindTeamResult(wire findTeamResultWire) (org.FindTeamResult, error) {
	switch wire.Variant {
	case "found":
		if wire.Team == nil {
			return nil, fmt.Errorf("find team result is missing its team")
		}
		team, err := decodeTeam(*wire.Team)
		if err != nil {
			return nil, err
		}
		return org.TeamFound{Value: team}, nil
	case "missing":
		return org.TeamMissing{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown find team result variant %q", wire.Variant)
	}
}

type teamMembersResultWire struct {
	Variant string                  `json:"variant"`
	UserIDs []string                `json:"user_ids,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeTeamMembersResult(result org.TeamMembersResult) teamMembersResultWire {
	switch typed := result.(type) {
	case org.TeamMembersListed:
		return teamMembersResultWire{Variant: "listed", UserIDs: encodeUserIDs(typed.Values)}
	case org.TeamMembersRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return teamMembersResultWire{Variant: "rejected", Error: &reason}
	default:
		return teamMembersResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown org result %T", result))}
	}
}

func decodeTeamMembersResult(wire teamMembersResultWire) (org.TeamMembersResult, error) {
	switch wire.Variant {
	case "listed":
		ids, err := decodeUserIDs(wire.UserIDs)
		if err != nil {
			return nil, err
		}
		return org.TeamMembersListed{Values: ids}, nil
	case "rejected":
		return org.TeamMembersRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown team members result variant %q", wire.Variant)
	}
}

// ---- shared helpers ----

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "org bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}
