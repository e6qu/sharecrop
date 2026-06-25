package agent

import "testing"

// FuzzAgentValueParsers feeds arbitrary strings to the agent credential value
// parsers (scope enum, label, and the opaque secret which base64-decodes
// untrusted input). None may panic, and each must return a declared variant.
func FuzzAgentValueParsers(f *testing.F) {
	seeds := []string{
		"", "tasks_read", "submissions_review", "bogus",
		"sck_", "sck_abc", "sck_!!!!", "sck_" + "AAAA",
		"   ", "a label", string(make([]byte, 200)),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		switch ParseScope(raw).(type) {
		case ScopeAccepted, ScopeRejected:
		default:
			t.Fatalf("ParseScope returned an unexpected type for %q", raw)
		}

		switch NewLabel(raw).(type) {
		case LabelAccepted, LabelRejected:
		default:
			t.Fatalf("NewLabel returned an unexpected type for %q", raw)
		}

		switch typed := ParseSecretPlain(raw).(type) {
		case SecretPlainAccepted:
			// An accepted secret must round-trip its string form and hash.
			if typed.Value.String() != raw {
				t.Fatalf("accepted secret did not round-trip: %q", raw)
			}
			_ = typed.Value.Hash()
		case SecretPlainRejected:
		default:
			t.Fatalf("ParseSecretPlain returned an unexpected type for %q", raw)
		}
	})
}
