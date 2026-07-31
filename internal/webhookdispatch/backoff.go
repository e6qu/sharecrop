//go:build !wasip1

package webhookdispatch

import (
	"math/rand/v2"
	"time"
)

// The bounded retry walk: after a failed attempt N (1-based) the delivery is
// retried after retryDelays[N-1] (with jitter); after the sixth failed
// attempt there is no further slot and the delivery is dead.
var retryDelays = []time.Duration{
	30 * time.Second,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	8 * time.Hour,
}

// MaxAttempts is the total number of delivery attempts before a delivery is
// marked dead.
const MaxAttempts = 6

// jitterFraction spreads retries ±20% around the schedule so a burst of
// failures does not come back as one synchronized burst of retries.
const jitterFraction = 0.2

// retryDecision is where a failed attempt leaves the walk.
type retryDecision interface {
	retryDecision()
}

type retryAt struct {
	at time.Time
}

type retriesExhausted struct{}

func (retryAt) retryDecision() {}

func (retriesExhausted) retryDecision() {}

// decideRetry maps a just-failed attempt number (1-based) onto the next step
// of the walk.
func decideRetry(attempt int64, now time.Time, jitter *rand.Rand) retryDecision {
	if attempt >= MaxAttempts {
		return retriesExhausted{}
	}
	index := attempt - 1
	if index < 0 {
		index = 0
	}
	if index >= int64(len(retryDelays)) {
		index = int64(len(retryDelays)) - 1
	}
	return retryAt{at: now.Add(jitterDelay(retryDelays[index], jitter))}
}

// jitterDelay scales base by a uniform factor in [1-jitterFraction,
// 1+jitterFraction].
func jitterDelay(base time.Duration, jitter *rand.Rand) time.Duration {
	factor := 1 - jitterFraction + 2*jitterFraction*jitter.Float64()
	return time.Duration(float64(base) * factor)
}
