//go:build !wasip1

package webhookdispatch

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/e6qu/sharecrop/internal/webhook"
)

// Delivery request headers. The signature covers "<timestamp>.<body>" so a
// receiver can verify both integrity and freshness with the shared secret.
const (
	HeaderWebhookID        = "Sharecrop-Webhook-Id"
	HeaderWebhookTimestamp = "Sharecrop-Webhook-Timestamp"
	HeaderWebhookSignature = "Sharecrop-Webhook-Signature"
)

// ComputeSignature renders the Sharecrop-Webhook-Signature value:
// "v1=" + hex(HMAC-SHA256(secret, "<unix-timestamp>.<body>")).
func ComputeSignature(secret webhook.Secret, unixTimestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret.String()))
	mac.Write([]byte(strconv.FormatInt(unixTimestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}
