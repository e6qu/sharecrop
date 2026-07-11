package httpserver

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/e6qu/sharecrop/internal/auth"
)

const accountTokenDeliveryAPI = "api"
const accountTokenDeliveryLog = "log"

type accountTokenDelivery struct {
	mode string
}

func newAccountTokenDeliveryFromEnv() accountTokenDelivery {
	mode := os.Getenv("SHARECROP_ACCOUNT_TOKEN_DELIVERY")
	// Default to log delivery (fail closed). In api mode `write` returns the
	// account token in the HTTP response body, and the password-reset request
	// is unauthenticated - so an api-mode default would hand a password-reset
	// token to anyone who knows a victim's email, an account-takeover
	// primitive. api mode is opt-in for the browser demo (browser-local, no
	// real accounts) and for tests; production `serve` keeps tokens in the
	// server log for admin-driven delivery.
	if mode != accountTokenDeliveryAPI && mode != accountTokenDeliveryLog {
		mode = accountTokenDeliveryLog
	}
	return accountTokenDelivery{mode: mode}
}

func (delivery accountTokenDelivery) write(w http.ResponseWriter, kind auth.AccountTokenKind, recipient string, issued auth.AccountTokenIssued) {
	if delivery.mode == accountTokenDeliveryLog {
		slog.Info("account token issued", "kind", kind.String(), "recipient", recipient, "token", issued.Token.String())
		writeJSON(w, http.StatusCreated, accountTokenSentResponse{Status: "sent"})
		return
	}
	writeJSON(w, http.StatusCreated, accountTokenResponse{Token: issued.Token.String()})
}
