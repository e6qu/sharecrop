package httpserver

import (
	"context"
	"errors"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
)

type fakeSensitiveFieldRedactor struct {
	count int
	err   error
}

func (redactor fakeSensitiveFieldRedactor) RedactSensitiveFields(context.Context, core.UserID) (int, error) {
	return redactor.count, redactor.err
}

func testUserIDForPrivacy(t *testing.T) core.UserID {
	t.Helper()
	result := core.NewUserID()
	created, matched := result.(core.UserIDCreated)
	if !matched {
		t.Fatalf("new user id failed: %#v", result)
	}
	return created.Value
}

func TestMemoryPrivacyServiceResolveSensitiveFieldDeletionUsesRedactor(t *testing.T) {
	service := NewMemoryPrivacyService(fakeSensitiveFieldRedactor{count: 3})
	ctx := context.Background()
	userID := testUserIDForPrivacy(t)

	createResult := service.Create(ctx, userID, "sensitive_field_deletion")
	created, matched := createResult.(PrivacyRequestSaved)
	if !matched {
		t.Fatalf("create privacy request: want PrivacyRequestSaved, got %#v", createResult)
	}

	resolveResult := service.Resolve(ctx, created.Value.ID, "redact away")
	resolved, matched := resolveResult.(PrivacyRequestSaved)
	if !matched {
		t.Fatalf("resolve privacy request: want PrivacyRequestSaved, got %#v", resolveResult)
	}
	if resolved.Value.RedactedFieldCount != 3 {
		t.Fatalf("redacted field count = %d, want 3", resolved.Value.RedactedFieldCount)
	}
}

func TestMemoryPrivacyServiceResolveWithoutRedactorLeavesCountZero(t *testing.T) {
	service := NewMemoryPrivacyService(nil)
	ctx := context.Background()
	userID := testUserIDForPrivacy(t)

	createResult := service.Create(ctx, userID, "sensitive_field_deletion")
	created := createResult.(PrivacyRequestSaved)

	resolveResult := service.Resolve(ctx, created.Value.ID, "note")
	resolved, matched := resolveResult.(PrivacyRequestSaved)
	if !matched {
		t.Fatalf("resolve privacy request: want PrivacyRequestSaved, got %#v", resolveResult)
	}
	if resolved.Value.RedactedFieldCount != 0 {
		t.Fatalf("redacted field count = %d, want 0 (no redactor configured)", resolved.Value.RedactedFieldCount)
	}
}

func TestMemoryPrivacyServiceResolvePropagatesRedactorError(t *testing.T) {
	service := NewMemoryPrivacyService(fakeSensitiveFieldRedactor{err: errors.New("redaction storage failed")})
	ctx := context.Background()
	userID := testUserIDForPrivacy(t)

	createResult := service.Create(ctx, userID, "sensitive_field_deletion")
	created := createResult.(PrivacyRequestSaved)

	resolveResult := service.Resolve(ctx, created.Value.ID, "note")
	if _, matched := resolveResult.(PrivacyRequestMutationRejected); !matched {
		t.Fatalf("resolve privacy request with failing redactor: want PrivacyRequestMutationRejected, got %#v", resolveResult)
	}
}

func TestMemoryPrivacyServiceRunRetentionUsesRedactor(t *testing.T) {
	service := NewMemoryPrivacyService(fakeSensitiveFieldRedactor{count: 5})
	ctx := context.Background()
	userID := testUserIDForPrivacy(t)

	result := service.RunRetention(ctx, userID)
	run, matched := result.(PrivacyRetentionRun)
	if !matched {
		t.Fatalf("run retention: want PrivacyRetentionRun, got %#v", result)
	}
	if run.RedactedFieldCount != 5 {
		t.Fatalf("redacted field count = %d, want 5", run.RedactedFieldCount)
	}
}

func TestMemoryPrivacyServiceDataExportIgnoresRedactor(t *testing.T) {
	service := NewMemoryPrivacyService(fakeSensitiveFieldRedactor{count: 9})
	ctx := context.Background()
	userID := testUserIDForPrivacy(t)

	createResult := service.Create(ctx, userID, "data_export")
	created := createResult.(PrivacyRequestSaved)

	resolveResult := service.Resolve(ctx, created.Value.ID, "note")
	resolved, matched := resolveResult.(PrivacyRequestSaved)
	if !matched {
		t.Fatalf("resolve privacy request: want PrivacyRequestSaved, got %#v", resolveResult)
	}
	if resolved.Value.RedactedFieldCount != 0 {
		t.Fatalf("redacted field count = %d, want 0 (data export doesn't redact)", resolved.Value.RedactedFieldCount)
	}
	if resolved.Value.ExportJSON == "" {
		t.Fatalf("data export result has no export JSON")
	}
}
