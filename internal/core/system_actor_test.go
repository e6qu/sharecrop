package core

import "testing"

func TestSystemUserIDMatchesSeededRow(t *testing.T) {
	if got := SystemUserID().String(); got != systemUserIDValue {
		t.Fatalf("SystemUserID() = %q, want %q (constant failed to parse)", got, systemUserIDValue)
	}
}

func TestSystemUserEmailIsReserved(t *testing.T) {
	if SystemUserEmail != "system@sharecrop.invalid" {
		t.Fatalf("SystemUserEmail = %q", SystemUserEmail)
	}
}
