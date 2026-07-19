//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestGuestHonorsAccountTokenDeliveryEnv covers request-shaping configuration
// that internal/http's newServer reads straight from the environment. The guest
// runs that same newServer, so such a variable the host forgets to forward
// into the guest is silently dropped and the guest falls back to a default -
// exactly what happened to SHARECROP_ACCOUNT_TOKEN_DELIVERY, which stayed on its
// fail-closed "log" default no matter what the operator configured, because it
// was never forwarded to the guest pool.
//
// The check drives the account-token endpoint through the guest with and
// without the env set and asserts the delivery mode actually changes: "api"
// returns the token in the body, the default only reports it was sent.
func TestGuestHonorsAccountTokenDeliveryEnv(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-app-guest")
	if err != nil {
		t.Fatalf("compile app guest: %v", err)
	}

	type verificationResponse struct {
		Token  string `json:"token"`
		Status string `json:"status"`
	}

	// issueVerification registers a fresh account through a guest pool built with
	// the given extra env, then requests an email-verification token and returns
	// the decoded response body.
	issueVerification := func(t *testing.T, env map[string]string) verificationResponse {
		t.Helper()
		guestEnv := map[string]string{"SHARECROP_ACCESS_TOKEN_SECRET": appRouteSecret}
		for key, value := range env {
			guestEnv[key] = value
		}
		guestPool, err := rpc.NewPool(ctx, guestWASM, storehost.Dispatcher(pool), 2)
		if err != nil {
			t.Fatalf("new pool: %v", err)
		}
		guestPool.WithGuestEnv(guestEnv)
		t.Cleanup(func() { _ = guestPool.Close(ctx) })
		guest := httpbridge.Handler(guestPool)

		do := func(method, path, authorization, body string) *http.Response {
			req := httptest.NewRequest(method, path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if authorization != "" {
				req.Header.Set("Authorization", "Bearer "+authorization)
			}
			rec := httptest.NewRecorder()
			guest.ServeHTTP(rec, req)
			return rec.Result()
		}

		email := uniqueIntegrationEmail(t, "token-delivery")
		registerResp := do("POST", "/api/auth/register", "",
			`{"email":"`+email+`","password":"correct horse battery staple"}`)
		if registerResp.StatusCode != http.StatusCreated {
			t.Fatalf("register: status %d, want 201", registerResp.StatusCode)
		}
		var registered struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.NewDecoder(registerResp.Body).Decode(&registered); err != nil {
			t.Fatalf("decode register response: %v", err)
		}

		verifyResp := do("POST", "/api/account/email-verification", registered.AccessToken, `{}`)
		if verifyResp.StatusCode != http.StatusCreated {
			t.Fatalf("email verification: status %d, want 201", verifyResp.StatusCode)
		}
		var decoded verificationResponse
		if err := json.NewDecoder(verifyResp.Body).Decode(&decoded); err != nil {
			t.Fatalf("decode verification response: %v", err)
		}
		return decoded
	}

	t.Run("api mode returns the token", func(t *testing.T) {
		body := issueVerification(t, map[string]string{"SHARECROP_ACCOUNT_TOKEN_DELIVERY": "api"})
		if body.Token == "" {
			t.Errorf("api delivery must return a token in the body, got %+v", body)
		}
	})

	t.Run("default log mode only reports sent", func(t *testing.T) {
		body := issueVerification(t, nil)
		if body.Token != "" {
			t.Errorf("default delivery must not return a token in the body, got %+v", body)
		}
		if body.Status != "sent" {
			t.Errorf("default delivery status = %q, want \"sent\"", body.Status)
		}
	})
}
