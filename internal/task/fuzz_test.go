package task

import (
	"net/url"
	"testing"
)

// FuzzTaskValueParsers feeds arbitrary strings to the value parsers that accept
// untrusted input (task type, reference URL, comment body, series state and
// description). None may panic, each must return a declared result variant, and
// an accepted non-empty reference URL must be an absolute http(s) URL.
func FuzzTaskValueParsers(f *testing.F) {
	seeds := []string{
		"", "general", "code_review", "qa_testing", "bogus",
		"https://github.com/a/b/pull/7", "http://x.test", "ftp://x", "javascript:alert(1)",
		"not a url", "   ", "draft", "published", "closed",
		"a comment", "//evil", "https://", "http://例え.テスト/パス",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		switch ParseTaskType(raw).(type) {
		case TaskTypeAccepted, TaskTypeRejected:
		default:
			t.Fatalf("ParseTaskType returned an unexpected type for %q", raw)
		}

		switch ParseSeriesState(raw).(type) {
		case SeriesStateAccepted, SeriesStateRejected:
		default:
			t.Fatalf("ParseSeriesState returned an unexpected type for %q", raw)
		}

		switch NewSeriesDescription(raw).(type) {
		case SeriesDescriptionAccepted, SeriesDescriptionRejected:
		default:
			t.Fatalf("NewSeriesDescription returned an unexpected type for %q", raw)
		}

		switch NewCommentBody(raw).(type) {
		case CommentBodyAccepted, CommentBodyRejected:
		default:
			t.Fatalf("NewCommentBody returned an unexpected type for %q", raw)
		}

		switch typed := NewReferenceURL(raw).(type) {
		case ReferenceURLAccepted:
			// A non-empty accepted reference must be an absolute http(s) URL.
			if value := typed.Value.String(); value != "" {
				parsed, err := url.Parse(value)
				if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
					t.Fatalf("accepted reference URL is not absolute http(s): %q", value)
				}
			}
		case ReferenceURLRejected:
		default:
			t.Fatalf("NewReferenceURL returned an unexpected type for %q", raw)
		}
	})
}
