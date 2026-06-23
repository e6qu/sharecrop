package core

import "testing"

func mustAcceptPage(t *testing.T, result PageResult) Page {
	t.Helper()
	accepted, matched := result.(PageAccepted)
	if !matched {
		t.Fatalf("result = %T, want PageAccepted", result)
	}
	return accepted.Value
}

func mustRejectPage(t *testing.T, result PageResult) {
	t.Helper()
	rejected, matched := result.(PageRejected)
	if !matched {
		t.Fatalf("result = %T, want PageRejected", result)
	}
	if rejected.Reason.Code().String() != ErrorCodeInvalidArgument.String() {
		t.Fatalf("code = %q, want %q", rejected.Reason.Code().String(), ErrorCodeInvalidArgument.String())
	}
}

func TestDefaultPageUsesDefaults(t *testing.T) {
	page := DefaultPage()
	if page.Limit() != 100 {
		t.Fatalf("limit = %d, want 100", page.Limit())
	}
	if page.Offset() != 0 {
		t.Fatalf("offset = %d, want 0", page.Offset())
	}
}

func TestNewPageAcceptsValidValues(t *testing.T) {
	page := mustAcceptPage(t, NewPage(25, 50))
	if page.Limit() != 25 {
		t.Fatalf("limit = %d, want 25", page.Limit())
	}
	if page.Offset() != 50 {
		t.Fatalf("offset = %d, want 50", page.Offset())
	}
}

func TestNewPageClampsLimitToMaximum(t *testing.T) {
	page := mustAcceptPage(t, NewPage(5000, 0))
	if page.Limit() != 200 {
		t.Fatalf("limit = %d, want 200", page.Limit())
	}
}

func TestNewPageRejectsNonPositiveLimit(t *testing.T) {
	mustRejectPage(t, NewPage(0, 0))
}

func TestNewPageRejectsNegativeOffset(t *testing.T) {
	mustRejectPage(t, NewPage(10, -1))
}
