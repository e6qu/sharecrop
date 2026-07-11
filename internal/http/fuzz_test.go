package httpserver

import (
	"net/http"
	"net/url"
	"testing"
)

// FuzzParsePageStrict feeds arbitrary raw query strings to parsePageStrict.
// However the attacker-controlled limit/offset are spelled, the parser must
// either reject the request outright or return a Page whose limit stays
// within the configured bounds and whose offset is never negative, so a
// malformed query can never reach SQL as an out-of-range LIMIT or a negative
// OFFSET. It must also never panic.
func FuzzParsePageStrict(f *testing.F) {
	seeds := []string{
		"",
		"limit=10&offset=5",
		"limit=0",
		"limit=-1",
		"offset=-100",
		"limit=999999999999999999999999",
		"limit=abc&offset=xyz",
		"limit=200&offset=0",
		"limit=201",
		"offset=2147483648",
		"limit=%ff&offset=%00",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, rawQuery string) {
		request := &http.Request{URL: &url.URL{RawQuery: rawQuery}}
		result := parsePageStrict(request)
		accepted, matched := result.(pageParseAccepted)
		if !matched {
			if result.(pageParseRejected).reason == "" {
				t.Fatalf("rejected page for %q must carry a reason", rawQuery)
			}
			return
		}
		if accepted.value.Limit() < 1 || accepted.value.Limit() > 200 {
			t.Fatalf("limit out of bounds for %q: %d", rawQuery, accepted.value.Limit())
		}
		if accepted.value.Offset() < 0 {
			t.Fatalf("negative offset for %q: %d", rawQuery, accepted.value.Offset())
		}
	})
}
