package org

import "github.com/e6qu/sharecrop/internal/core"

type Role struct {
	value string
}

var (
	RoleOwner           = Role{value: "owner"}
	RoleAdmin           = Role{value: "admin"}
	RoleMember          = Role{value: "member"}
	RoleBilling         = Role{value: "billing"}
	RoleReviewer        = Role{value: "reviewer"}
	RolePublicPublisher = Role{value: "public_publisher"}
)

type RoleResult interface {
	roleResult()
}

type RoleAccepted struct {
	Value Role
}

type RoleRejected struct {
	Reason core.DomainError
}

func (RoleAccepted) roleResult() {}

func (RoleRejected) roleResult() {}

func ParseRole(raw string) RoleResult {
	switch raw {
	case RoleOwner.value:
		return RoleAccepted{Value: RoleOwner}
	case RoleAdmin.value:
		return RoleAccepted{Value: RoleAdmin}
	case RoleMember.value:
		return RoleAccepted{Value: RoleMember}
	case RoleBilling.value:
		return RoleAccepted{Value: RoleBilling}
	case RoleReviewer.value:
		return RoleAccepted{Value: RoleReviewer}
	case RolePublicPublisher.value:
		return RoleAccepted{Value: RolePublicPublisher}
	default:
		return RoleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "organization role is invalid")}
	}
}

func (role Role) String() string {
	return role.value
}

type Permission struct {
	value string
}

var (
	PermissionManageMembers          = Permission{value: "manage_members"}
	PermissionManageTeams            = Permission{value: "manage_teams"}
	PermissionCreateOrganizationTask = Permission{value: "create_organization_task"}
	PermissionReviewSubmissions      = Permission{value: "review_submissions"}
	PermissionManageBilling          = Permission{value: "manage_billing"}
	PermissionManageWallets          = Permission{value: "manage_wallets"}
	PermissionPublishPublicTask      = Permission{value: "publish_public_task"}
	PermissionSwitchTaskVisibility   = Permission{value: "switch_task_visibility"}
)

func (permission Permission) String() string {
	return permission.value
}

type PermissionCheck interface {
	permissionCheck()
}

type PermissionGranted struct{}

type PermissionDenied struct {
	Reason core.DomainError
}

func (PermissionGranted) permissionCheck() {}

func (PermissionDenied) permissionCheck() {}

func CheckPermission(roles []Role, permission Permission) PermissionCheck {
	for _, role := range roles {
		result := roleGrantsPermission(role, permission)
		if _, granted := result.(PermissionGranted); granted {
			return PermissionGranted{}
		}
	}
	return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "organization permission denied")}
}

func roleGrantsPermission(role Role, permission Permission) PermissionCheck {
	switch role {
	case RoleOwner:
		return PermissionGranted{}
	case RoleAdmin:
		return adminPermission(permission)
	case RoleBilling:
		return billingPermission(permission)
	case RoleReviewer:
		return reviewerPermission(permission)
	case RolePublicPublisher:
		return publisherPermission(permission)
	default:
		return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "role does not grant permission")}
	}
}

func adminPermission(permission Permission) PermissionCheck {
	switch permission {
	case PermissionManageMembers, PermissionManageTeams, PermissionCreateOrganizationTask, PermissionReviewSubmissions, PermissionSwitchTaskVisibility:
		return PermissionGranted{}
	default:
		return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "admin role does not grant permission")}
	}
}

func billingPermission(permission Permission) PermissionCheck {
	switch permission {
	case PermissionManageBilling:
		return PermissionGranted{}
	default:
		return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "billing role does not grant permission")}
	}
}

func reviewerPermission(permission Permission) PermissionCheck {
	switch permission {
	case PermissionReviewSubmissions:
		return PermissionGranted{}
	default:
		return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "reviewer role does not grant permission")}
	}
}

func publisherPermission(permission Permission) PermissionCheck {
	switch permission {
	case PermissionPublishPublicTask, PermissionSwitchTaskVisibility:
		return PermissionGranted{}
	default:
		return PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "public publisher role does not grant permission")}
	}
}
