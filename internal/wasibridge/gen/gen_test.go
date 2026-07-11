package gen

import (
	"strings"
	"testing"
)

func sources(body string) map[string][]byte {
	return map[string][]byte{"store.go": []byte(body)}
}

func TestGenerateEmitsDispatchAndClient(t *testing.T) {
	source, err := Generate(sources(`package audit
import "context"
type Store interface {
	Get(context.Context, core.AuditEventID) GetResult
	List(context.Context, ListFilters, core.Page) ListResult
}
`), "audit")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	for _, want := range []string{
		"func Dispatch(ctx context.Context, store audit.Store,",
		"type GuestStore struct",
		"func (g GuestStore) Get(ctx context.Context, argID core.AuditEventID) audit.GetResult",
		"func (g GuestStore) List(ctx context.Context, argFilters audit.ListFilters, argPage core.Page) audit.ListResult",
		`methodGet  = "audit.Get"`,
		"var _ audit.Store = GuestStore{}",
	} {
		if !strings.Contains(source, want) {
			t.Errorf("generated source is missing %q", want)
		}
	}
}

func TestGenerateRejectsUnregisteredArgumentType(t *testing.T) {
	_, err := Generate(sources(`package audit
import "context"
type Store interface {
	Weird(context.Context, MysteryValue) GetResult
}
`), "audit")
	if err == nil {
		t.Fatalf("expected an error for an unregistered argument type")
	}
	if !strings.Contains(err.Error(), "audit.MysteryValue") {
		t.Errorf("error should name the unregistered type, got: %v", err)
	}
}

func TestGenerateRejectsUnregisteredResultType(t *testing.T) {
	_, err := Generate(sources(`package audit
import "context"
type Store interface {
	Get(context.Context, core.AuditEventID) MysteryResult
}
`), "audit")
	if err == nil {
		t.Fatalf("expected an error for an unregistered result type")
	}
	if !strings.Contains(err.Error(), "audit.MysteryResult") {
		t.Errorf("error should name the unregistered result type, got: %v", err)
	}
}

func TestGenerateRequiresTheInterface(t *testing.T) {
	_, err := Generate(sources(`package audit
type NotStore struct{}
`), "audit")
	if err == nil || !strings.Contains(err.Error(), `interface "Store" not found`) {
		t.Fatalf("expected a not-found error, got: %v", err)
	}
}
