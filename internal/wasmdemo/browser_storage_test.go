package wasmdemo

import "testing"

type testBrowserStorage struct {
	values map[string]string
}

func newTestBrowserStorage() *testBrowserStorage {
	return &testBrowserStorage{values: map[string]string{}}
}

func (storage *testBrowserStorage) Put(key StorageKey, value string) StorageWriteResult {
	storage.values[key.String()] = value
	return StorageWritten{}
}

func (storage *testBrowserStorage) Get(key StorageKey) StorageReadResult {
	value, ok := storage.values[key.String()]
	if !ok {
		return StorageMissing{Reason: "storage key was not found"}
	}
	return StorageRead{Value: value}
}

func TestModerationTriageBrowserStorageRoundTrip(t *testing.T) {
	storage := newTestBrowserStorage()
	triage := StoredModerationTriage{
		ReportID:       "audit-1",
		State:          "resolved",
		ResolutionNote: "handled",
		UpdatedBy:      "user-admin",
		UpdatedAt:      "2026-06-30T10:00:00Z",
	}

	saveResult := SaveModerationTriage(storage, triage)
	if _, matched := saveResult.(ModerationTriageStored); !matched {
		t.Fatalf("save result = %T, want ModerationTriageStored", saveResult)
	}

	loadResult := LoadModerationTriage(storage, "audit-1")
	loaded, matched := loadResult.(ModerationTriageStored)
	if !matched {
		t.Fatalf("load result = %T, want ModerationTriageStored", loadResult)
	}
	if loaded.Value.State != "resolved" {
		t.Fatalf("state = %q, want resolved", loaded.Value.State)
	}
	if loaded.Value.ResolutionNote != "handled" {
		t.Fatalf("resolution note = %q, want handled", loaded.Value.ResolutionNote)
	}
}

func TestModerationTriageBrowserStorageRejectsMissingRecord(t *testing.T) {
	result := LoadModerationTriage(newTestBrowserStorage(), "audit-missing")
	rejected, matched := result.(ModerationTriageStorageRejected)
	if !matched {
		t.Fatalf("result = %T, want ModerationTriageStorageRejected", result)
	}
	if rejected.Reason != "storage key was not found" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestModerationTriageBrowserStorageRejectsInvalidState(t *testing.T) {
	storage := newTestBrowserStorage()
	result := SaveModerationTriage(storage, StoredModerationTriage{ReportID: "audit-1", State: "unknown"})
	rejected, matched := result.(ModerationTriageStorageRejected)
	if !matched {
		t.Fatalf("result = %T, want ModerationTriageStorageRejected", result)
	}
	if rejected.Reason != "moderation triage state is invalid" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestPrivacyRequestBrowserStorageListRoundTrip(t *testing.T) {
	storage := newTestBrowserStorage()
	request := StoredPrivacyRequest{
		ID:                 "privacy-1",
		Kind:               "data_export",
		Status:             "queued",
		RequestedBy:        "user-1",
		CreatedAt:          "2026-06-30T10:00:00Z",
		RedactedFieldCount: 0,
	}

	saveResult := SavePrivacyRequest(storage, request)
	if _, matched := saveResult.(PrivacyRequestStored); !matched {
		t.Fatalf("save result = %T, want PrivacyRequestStored", saveResult)
	}

	listResult := ListPrivacyRequests(storage)
	listed, matched := listResult.(PrivacyRequestsStored)
	if !matched {
		t.Fatalf("list result = %T, want PrivacyRequestsStored", listResult)
	}
	if len(listed.Values) != 1 {
		t.Fatalf("privacy request count = %d, want 1", len(listed.Values))
	}
	if listed.Values[0].ID != "privacy-1" || listed.Values[0].Kind != "data_export" || listed.Values[0].RequestedBy != "user-1" {
		t.Fatalf("listed request = %#v", listed.Values[0])
	}
}

func TestPrivacyRequestBrowserStorageRejectsInvalidKind(t *testing.T) {
	result := SavePrivacyRequest(newTestBrowserStorage(), StoredPrivacyRequest{
		ID:          "privacy-1",
		Kind:        "unknown",
		Status:      "queued",
		RequestedBy: "user-1",
	})
	rejected, matched := result.(PrivacyRequestStorageRejected)
	if !matched {
		t.Fatalf("result = %T, want PrivacyRequestStorageRejected", result)
	}
	if rejected.Reason != "privacy request kind is invalid" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}
