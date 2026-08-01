package httpserver

import "testing"

// TestValidModerationReasonMatchesContractEnum pins the sealed report
// category set against the contracts ModerationReason enum
// (internal/contracts/definitions.go): the stage-3 dispute category is a
// valid reason, submission is a valid subject kind for it, and unknown
// values stay rejected.
func TestValidModerationReasonMatchesContractEnum(t *testing.T) {
	for _, reason := range []string{"spam", "abuse", "pii", "policy", "dispute", "other"} {
		if !validModerationReason(reason) {
			t.Fatalf("reason %q should be valid", reason)
		}
	}
	for _, reason := range []string{"", "disputed", "appeal", "DISPUTE"} {
		if validModerationReason(reason) {
			t.Fatalf("reason %q should be invalid", reason)
		}
	}
	if !validModerationSubjectKind("submission") {
		t.Fatalf("submission should be a valid moderation subject kind")
	}
}
