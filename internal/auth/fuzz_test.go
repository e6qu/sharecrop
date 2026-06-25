package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

func mustUserID(t testing.TB) core.UserID {
	t.Helper()
	result := core.ParseUserID("00000000-0000-7000-8000-000000000000")
	created, ok := result.(core.UserIDCreated)
	if !ok {
		t.Fatalf("test user id was rejected")
	}
	return created.Value
}

func fuzzSecret(t *testing.T) AccessTokenSecret {
	t.Helper()
	result := NewAccessTokenSecret("0123456789abcdef0123456789abcdef")
	accepted, ok := result.(AccessTokenSecretAccepted)
	if !ok {
		t.Fatalf("test secret was rejected")
	}
	return accepted.Value
}

// FuzzVerifyAccessToken feeds arbitrary token strings to the verifier. The
// verifier must never panic, and it must never accept a token whose HMAC the
// fuzzer cannot have produced (it does not know the secret), so every fuzz
// input must be rejected.
func FuzzVerifyAccessToken(f *testing.F) {
	now := time.Unix(1_700_000_000, 0)
	secret := fuzzSecret(&testing.T{})

	// A genuine, correctly-signed token as a seed, plus malformed shapes.
	signed := SignAccessToken(secret, UserSubject{ID: mustUserID(f)}, now)
	if accepted, ok := signed.(AccessTokenAccepted); ok {
		f.Add(accepted.Value.String())
	}
	f.Add("")
	f.Add("a.b.c")
	f.Add("...")
	f.Add(strings.Repeat("A", 4096))

	f.Fuzz(func(t *testing.T, raw string) {
		result := VerifyAccessToken(secret, AccessToken{value: raw}, now)
		verified, ok := result.(SubjectVerified)
		if !ok {
			if _, rejected := result.(SubjectVerifyRejected); !rejected {
				t.Fatalf("verify returned an unexpected result type")
			}
			return
		}
		// The fuzzer does not know the secret, so every input it accepts can only
		// be one this test signed and re-fed. Re-signing the verified subject
		// must reproduce the exact token; otherwise verification accepted a
		// forgery.
		resigned := SignAccessToken(secret, verified.Value, now)
		accepted, signedOK := resigned.(AccessTokenAccepted)
		if !signedOK {
			t.Fatalf("verified subject failed to re-sign")
		}
		if accepted.Value.String() != raw {
			t.Fatalf("verifier accepted a token it did not produce: %q", raw)
		}
	})
}
