// Package ratelimitbridge is the WASI bridge for internal/http's RateLimiter (a
// RuntimeState service). Unlike the domain and other infra bridges it is hand-
// written, not generated: RateLimiter.Allow takes no context and returns a bare
// bool, which the code generator (built for ctx + single-union-result methods)
// does not model. There are two limiters - one keyed by client IP, one by MCP
// agent subject - so the wire method carries the prefix ("ip"/"subject") that
// selects which. Bridging keeps the token buckets in one shared Postgres store
// instead of a per-instance in-memory copy, so a pooled guest rate-limits
// consistently across instances. internal/http is package httpserver.
package ratelimitbridge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	httpserver "github.com/e6qu/sharecrop/internal/http"
)

// Invoker sends a call to the host and returns the serialized result (rpc.Invoke
// on the guest; a test stand-in otherwise).
type Invoker func(method string, args []byte) ([]byte, error)

// GuestRateLimiter implements httpserver.RateLimiter by forwarding each call to
// the host's shared, Postgres-backed limiter. prefix selects the limiter.
type GuestRateLimiter struct {
	invoke Invoker
	prefix string
}

// NewGuestRateLimiter builds a guest limiter for the "ip" or "subject" limiter.
func NewGuestRateLimiter(invoke Invoker, prefix string) GuestRateLimiter {
	return GuestRateLimiter{invoke: invoke, prefix: prefix}
}

// Allow reports whether the key is under its rate budget. On a transport failure
// it fails open (allows): a broken bridge must not lock every client out.
func (g GuestRateLimiter) Allow(key string) bool {
	raw, err := g.call("Allow", key)
	if err != nil {
		return true
	}
	var allowed bool
	if err := json.Unmarshal(raw, &allowed); err != nil {
		return true
	}
	return allowed
}

// ActiveBuckets reports the number of live buckets (used by the metrics route).
func (g GuestRateLimiter) ActiveBuckets() int {
	raw, err := g.call("ActiveBuckets", "")
	if err != nil {
		return 0
	}
	var count int
	if err := json.Unmarshal(raw, &count); err != nil {
		return 0
	}
	return count
}

// StorageKind names the limiter's backing store (used by the metrics route).
func (g GuestRateLimiter) StorageKind() string {
	raw, err := g.call("StorageKind", "")
	if err != nil {
		return ""
	}
	var kind string
	if err := json.Unmarshal(raw, &kind); err != nil {
		return ""
	}
	return kind
}

func (g GuestRateLimiter) call(op, key string) ([]byte, error) {
	args, err := json.Marshal(key)
	if err != nil {
		return nil, err
	}
	return g.invoke("ratelimit."+g.prefix+"."+op, args)
}

var _ httpserver.RateLimiter = GuestRateLimiter{}

// Dispatch services one rate-limit call against the host's real limiters. method
// is "ratelimit.<prefix>.<op>"; the prefix picks ip vs subject. The context is
// unused (RateLimiter has none) but kept for a uniform dispatcher signature.
func Dispatch(_ context.Context, ipLimiter, subjectLimiter httpserver.RateLimiter, method string, args []byte) ([]byte, error) {
	parts := strings.SplitN(method, ".", 3)
	if len(parts) != 3 || parts[0] != "ratelimit" {
		return nil, fmt.Errorf("ratelimit bridge: unknown method %q", method)
	}
	limiter := ipLimiter
	if parts[1] == "subject" {
		limiter = subjectLimiter
	}
	switch parts[2] {
	case "Allow":
		var key string
		if err := json.Unmarshal(args, &key); err != nil {
			return nil, fmt.Errorf("ratelimit bridge: decode key: %w", err)
		}
		return json.Marshal(limiter.Allow(key))
	case "ActiveBuckets":
		return json.Marshal(limiter.ActiveBuckets())
	case "StorageKind":
		return json.Marshal(limiter.StorageKind())
	default:
		return nil, fmt.Errorf("ratelimit bridge: unknown op %q", parts[2])
	}
}
