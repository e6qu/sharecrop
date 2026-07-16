package httpserver

import (
	"testing"
	"time"
)

func TestSHAUTHTransactionIsAuthenticated(t *testing.T) {
	config := shauthConfig{clientSecret: "test-client-secret"}
	want := shauthTransaction{State: "state", Nonce: "nonce", Verifier: "verifier", Expires: time.Now().Add(time.Minute).Unix()}
	encoded, err := config.encodeTransaction(want)
	if err != nil {
		t.Fatalf("encode transaction: %v", err)
	}
	got, err := config.decodeTransaction(encoded)
	if err != nil {
		t.Fatalf("decode transaction: %v", err)
	}
	if got != want {
		t.Fatalf("transaction = %#v, want %#v", got, want)
	}
	if _, err := config.decodeTransaction(encoded[:len(encoded)-1] + "A"); err == nil {
		t.Fatal("tampered transaction was accepted")
	}
}

func TestSHAUTHConfigRequiresCompleteHTTPSCoordinates(t *testing.T) {
	for _, config := range []shauthConfig{
		{issuer: "https://auth.dev.e6qu.dev", clientID: "client"},
		{issuer: "http://auth.dev.e6qu.dev", clientID: "client", clientSecret: "secret", publicURL: "https://sharecrop.dev.e6qu.dev"},
		{issuer: "https://auth.dev.e6qu.dev", clientID: "client", clientSecret: "secret", publicURL: "http://sharecrop.dev.e6qu.dev"},
	} {
		if err := config.validate(); err == nil {
			t.Fatalf("config %#v was accepted", config)
		}
	}
	if err := (shauthConfig{}).validate(); err != nil {
		t.Fatalf("disabled config: %v", err)
	}
	if err := (shauthConfig{issuer: "https://auth.dev.e6qu.dev", clientID: "client", clientSecret: "secret", publicURL: "https://sharecrop.dev.e6qu.dev"}).validate(); err != nil {
		t.Fatalf("valid config: %v", err)
	}
}
