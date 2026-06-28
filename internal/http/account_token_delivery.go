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
	if mode == "" {
		mode = accountTokenDeliveryAPI
	}
	if mode != accountTokenDeliveryAPI && mode != accountTokenDeliveryLog {
		mode = accountTokenDeliveryAPI
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
