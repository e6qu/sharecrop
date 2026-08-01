//go:build !wasip1

package webhookdispatch

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/webhook"
)

func TestRejectNonPublicAddressTable(t *testing.T) {
	banned := map[string]string{
		"loopback v4":          "127.0.0.1:443",
		"loopback v4 high":     "127.8.9.10:443",
		"loopback v6":          "[::1]:443",
		"private 10/8":         "10.1.2.3:443",
		"private 172.16/12":    "172.16.0.9:443",
		"private 172.31 edge":  "172.31.255.254:443",
		"private 192.168/16":   "192.168.1.1:443",
		"link-local v4":        "169.254.10.20:443",
		"link-local v6":        "[fe80::1]:443",
		"unique-local fc":      "[fc00::1]:443",
		"unique-local fd":      "[fd12:3456:789a::1]:443",
		"unspecified v4":       "0.0.0.0:443",
		"unspecified v6":       "[::]:443",
		"multicast v4":         "224.0.0.1:443",
		"multicast v6":         "[ff02::1]:443",
		"mapped loopback":      "[::ffff:127.0.0.1]:443",
		"mapped private":       "[::ffff:10.0.0.1]:443",
		"mapped link-local":    "[::ffff:169.254.1.1]:443",
		"not an address":       "receiver.example.com:443",
		"missing port":         "203.0.113.7",
		"garbage":              "not-an-address",
		"mapped rfc1918 upper": "[::ffff:192.168.7.7]:443",
	}
	for name, address := range banned {
		if err := RejectNonPublicAddress(address); err == nil {
			t.Fatalf("%s: address %q was allowed", name, address)
		}
	}

	allowed := map[string]string{
		"public v4":          "93.184.216.34:443",
		"public v4 doc":      "203.0.113.7:443",
		"public v6":          "[2606:2800:220:1:248:1893:25c8:1946]:443",
		"mapped public v4":   "[::ffff:93.184.216.34]:443",
		"public cgnat-range": "100.64.0.1:443",
	}
	for name, address := range allowed {
		if err := RejectNonPublicAddress(address); err != nil {
			t.Fatalf("%s: address %q was blocked: %v", name, address, err)
		}
	}
}

func TestComputeSignatureVector(t *testing.T) {
	// Fixed vector: secret scrop_whsec_<43 'A's>, timestamp 1767322800,
	// body {"hello":"world"}. Recomputing the HMAC by construction:
	// v1=hex(HMAC-SHA256(secret, "1767322800.{\"hello\":\"world\"}")).
	secret, matched := webhook.ParseSecret("scrop_whsec_" + strings.Repeat("A", 43)).(webhook.SecretAccepted)
	if !matched {
		t.Fatalf("fixture secret rejected")
	}
	got := ComputeSignature(secret.Value, 1767322800, []byte(`{"hello":"world"}`))
	// Vector computed independently (python hmac over the same inputs), so
	// this pins the algorithm rather than mirroring the implementation.
	want := "v1=0ff84b09791decce871e45fb70b49eef64ccb78a27013fd384fb9685cb4236d5"
	if got != want {
		t.Fatalf("signature = %s, want %s", got, want)
	}
	// The signature binds the timestamp: a shifted timestamp changes it.
	if shifted := ComputeSignature(secret.Value, 1767322801, []byte(`{"hello":"world"}`)); shifted == got {
		t.Fatalf("signature ignored the timestamp")
	}
	// And the body: a changed payload changes it.
	if reBodied := ComputeSignature(secret.Value, 1767322800, []byte(`{"hello":"mars"}`)); reBodied == got {
		t.Fatalf("signature ignored the body")
	}
}

func TestDecideRetryWalksTheScheduleWithJitterBounds(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	jitter := rand.New(rand.NewPCG(1, 2))

	baseByAttempt := map[int64]time.Duration{
		1: 30 * time.Second,
		2: 5 * time.Minute,
		3: 30 * time.Minute,
		4: 2 * time.Hour,
		5: 8 * time.Hour,
	}
	for attempt := int64(1); attempt <= 5; attempt++ {
		// Sample repeatedly so the jitter bounds are exercised, not a
		// single lucky draw.
		for sample := 0; sample < 200; sample++ {
			decision, matched := decideRetry(attempt, now, jitter).(retryAt)
			if !matched {
				t.Fatalf("attempt %d: expected a retry slot", attempt)
			}
			delay := decision.at.Sub(now)
			base := baseByAttempt[attempt]
			lower := time.Duration(float64(base) * 0.8)
			upper := time.Duration(float64(base) * 1.2)
			if delay < lower || delay > upper {
				t.Fatalf("attempt %d: delay %v outside [%v, %v]", attempt, delay, lower, upper)
			}
		}
	}

	if _, matched := decideRetry(6, now, jitter).(retriesExhausted); !matched {
		t.Fatalf("attempt 6 should exhaust the walk")
	}
	if _, matched := decideRetry(9, now, jitter).(retriesExhausted); !matched {
		t.Fatalf("attempt beyond the schedule should exhaust the walk")
	}
}

func TestJitterVaries(t *testing.T) {
	jitter := rand.New(rand.NewPCG(7, 11))
	first := jitterDelay(time.Minute, jitter)
	varied := false
	for sample := 0; sample < 50; sample++ {
		if jitterDelay(time.Minute, jitter) != first {
			varied = true
			break
		}
	}
	if !varied {
		t.Fatalf("jitter produced a constant delay")
	}
}

func TestBoundedStatusTruncates(t *testing.T) {
	long := strings.Repeat("x", 500)
	if got := boundedStatus(long); len(got) != lastStatusLimit {
		t.Fatalf("bounded status length = %d, want %d", len(got), lastStatusLimit)
	}
	if got := boundedStatus("200"); got != "200" {
		t.Fatalf("short status was altered: %q", got)
	}
}

// TestClaimHoldCoversWorstCaseBatch pins the derivation of the claim hold:
// it must cover a whole claimed batch timing out (claimBatchSize deliveries,
// each burning the full requestTimeout) plus positive slack, so a slow batch
// can never be re-claimed — and duplicate-POSTed — by another replica.
func TestClaimHoldCoversWorstCaseBatch(t *testing.T) {
	worstCase := time.Duration(claimBatchSize) * requestTimeout
	if claimHold <= worstCase {
		t.Fatalf("claimHold = %s, must exceed the worst-case batch time %s", claimHold, worstCase)
	}
}
