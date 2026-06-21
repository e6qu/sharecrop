package org

import "testing"

func TestOwnerRoleGrantsPublicPublishing(t *testing.T) {
	result := CheckPermission([]Role{RoleOwner}, PermissionPublishPublicTask)
	if _, matched := result.(PermissionGranted); !matched {
		t.Fatalf("result = %T, want PermissionGranted", result)
	}
}

func TestMemberRoleDoesNotGrantPublicPublishing(t *testing.T) {
	result := CheckPermission([]Role{RoleMember}, PermissionPublishPublicTask)
	if _, matched := result.(PermissionDenied); !matched {
		t.Fatalf("result = %T, want PermissionDenied", result)
	}
}

func TestPublicPublisherRoleIsSeparateFromReviewer(t *testing.T) {
	publishResult := CheckPermission([]Role{RolePublicPublisher}, PermissionPublishPublicTask)
	if _, matched := publishResult.(PermissionGranted); !matched {
		t.Fatalf("publish result = %T, want PermissionGranted", publishResult)
	}

	reviewResult := CheckPermission([]Role{RolePublicPublisher}, PermissionReviewSubmissions)
	if _, matched := reviewResult.(PermissionDenied); !matched {
		t.Fatalf("review result = %T, want PermissionDenied", reviewResult)
	}
}

func TestOrganizationNameRejectsBlankValue(t *testing.T) {
	result := NewOrganizationName(" ")
	if _, matched := result.(OrganizationNameRejected); !matched {
		t.Fatalf("result = %T, want OrganizationNameRejected", result)
	}
}

func TestPermissionStringReturnsStableValue(t *testing.T) {
	if PermissionPublishPublicTask.String() != "publish_public_task" {
		t.Fatalf("permission = %q, want publish_public_task", PermissionPublishPublicTask.String())
	}
}

func TestParseMembershipStatusAcceptsActive(t *testing.T) {
	result := ParseMembershipStatus("active")
	accepted, matched := result.(MembershipStatusAccepted)
	if !matched {
		t.Fatalf("result = %T, want MembershipStatusAccepted", result)
	}
	if accepted.Value.String() != MembershipStatusActive.String() {
		t.Fatalf("status = %q, want %q", accepted.Value.String(), MembershipStatusActive.String())
	}
}
