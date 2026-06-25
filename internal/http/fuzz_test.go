package httpserver

import (
	"net/http"
	"net/url"
	"testing"
)

// FuzzParsePage feeds arbitrary raw query strings to parsePage. However the
// attacker-controlled limit/offset are spelled, parsePage must return a Page
// whose limit stays within the configured bounds and whose offset is never
// negative, so a malformed query can never reach SQL as an out-of-range LIMIT
// or a negative OFFSET. It must also never panic.
func FuzzParsePage(f *testing.F) {
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
		page := parsePage(request)
		if page.Limit() < 1 || page.Limit() > 200 {
			t.Fatalf("limit out of bounds for %q: %d", rawQuery, page.Limit())
		}
		if page.Offset() < 0 {
			t.Fatalf("negative offset for %q: %d", rawQuery, page.Offset())
		}
	})
}
