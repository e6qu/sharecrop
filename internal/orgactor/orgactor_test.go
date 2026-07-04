package orgactor

import (
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

func TestCheck(t *testing.T) {
	organizationID := testOrganizationID(t)
	otherOrganizationID := testOrganizationID(t)

	cases := []struct {
		name  string
		actor auth.Subject
		want  Result
	}{
		{"matching org subject", auth.OrgSubject{ID: organizationID}, Match},
		{"mismatched org subject", auth.OrgSubject{ID: otherOrganizationID}, Mismatch},
		{"user subject", auth.UserSubject{ID: testUserID(t)}, NotApplicable},
		{"guest subject", auth.GuestSubject{ID: testGuestID(t)}, NotApplicable},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := Check(testCase.actor, organizationID); got != testCase.want {
				t.Fatalf("Check() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func testOrganizationID(t *testing.T) core.OrganizationID {
	t.Helper()
	result := core.NewOrganizationID()
	created, matched := result.(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("new organization id: %+v", result)
	}
	return created.Value
}

func testUserID(t *testing.T) core.UserID {
	t.Helper()
	result := core.NewUserID()
	created, matched := result.(core.UserIDCreated)
	if !matched {
		t.Fatalf("new user id: %+v", result)
	}
	return created.Value
}

func testGuestID(t *testing.T) core.GuestID {
	t.Helper()
	result := core.NewGuestID()
	created, matched := result.(core.GuestIDCreated)
	if !matched {
		t.Fatalf("new guest id: %+v", result)
	}
	return created.Value
}
