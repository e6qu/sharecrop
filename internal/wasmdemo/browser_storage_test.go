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

func TestSavedQueueViewBrowserStorageUpsertsAndListsByScope(t *testing.T) {
	storage := newTestBrowserStorage()
	first := StoredSavedQueueView{
		ID:          "saved-view-1",
		UserID:      "user-1",
		Scope:       "team_work",
		Name:        "Ready work",
		Query:       "review",
		StateFilter: "ready",
		TypeFilter:  "code_review",
		Sort:        "title_asc",
	}
	if _, matched := SaveSavedQueueView(storage, first).(SavedQueueViewStored); !matched {
		t.Fatalf("first save was rejected")
	}
	second := first
	second.ID = "saved-view-2"
	second.Query = "updated"
	saveResult := SaveSavedQueueView(storage, second)
	saved, matched := saveResult.(SavedQueueViewStored)
	if !matched {
		t.Fatalf("second save result = %T, want SavedQueueViewStored", saveResult)
	}
	if saved.Value.ID != "saved-view-1" {
		t.Fatalf("upsert id = %q, want saved-view-1", saved.Value.ID)
	}

	listResult := ListSavedQueueViews(storage, "user-1", "team_work")
	listed, listedMatched := listResult.(SavedQueueViewsStored)
	if !listedMatched {
		t.Fatalf("list result = %T, want SavedQueueViewsStored", listResult)
	}
	if len(listed.Values) != 1 || listed.Values[0].Query != "updated" {
		t.Fatalf("listed views = %#v", listed.Values)
	}
}

func TestSavedQueueViewBrowserStorageRejectsInvalidScope(t *testing.T) {
	result := SaveSavedQueueView(newTestBrowserStorage(), StoredSavedQueueView{
		ID:     "saved-view-1",
		UserID: "user-1",
		Scope:  "unknown",
		Name:   "Ready work",
	})
	rejected, matched := result.(SavedQueueViewStorageRejected)
	if !matched {
		t.Fatalf("result = %T, want SavedQueueViewStorageRejected", result)
	}
	if rejected.Reason != "saved queue view scope is invalid" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestSavedQueueViewBrowserStorageRejectsCorruptStoredRecord(t *testing.T) {
	storage := newTestBrowserStorage()
	keyResult := savedQueueViewKey("user-1", "team_work", "Ready work")
	key, matched := keyResult.(StorageKeyAccepted)
	if !matched {
		t.Fatalf("key result = %T, want StorageKeyAccepted", keyResult)
	}
	storage.values[key.Value.String()] = `{"id":"saved-view-1","user_id":"other-user","scope":"team_work","name":"Ready work"}`

	result := SaveSavedQueueView(storage, StoredSavedQueueView{
		ID:     "saved-view-2",
		UserID: "user-1",
		Scope:  "team_work",
		Name:   "Ready work",
	})
	rejected, rejectedMatched := result.(SavedQueueViewStorageRejected)
	if !rejectedMatched {
		t.Fatalf("result = %T, want SavedQueueViewStorageRejected", result)
	}
	if rejected.Reason != "saved queue view storage key contains mismatched record" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestTaskBrowserStorageRoundTrip(t *testing.T) {
	storage := newTestBrowserStorage()
	task := StoredTask{
		ID:                     "task-1",
		OwnerKind:              "user",
		OwnerID:                "user-1",
		Title:                  "Label receipts",
		Description:            "Extract totals.",
		TaskType:               "general",
		RewardKind:             "none",
		ParticipationPolicy:    "open",
		AssigneeScope:          "user",
		ReservationExpiryHours: 48,
		State:                  "draft",
		VisibilityKind:         "public",
		SeriesKind:             "standalone",
		ResponseSchemaJSON:     `{"kind":"freeform"}`,
		PayloadKind:            "none",
		CreatedBy:              "user-1",
	}

	saveResult := SaveTask(storage, task)
	if _, matched := saveResult.(TaskStored); !matched {
		t.Fatalf("save result = %T, want TaskStored", saveResult)
	}
	loadResult := LoadTask(storage, "task-1")
	loaded, matched := loadResult.(TaskStored)
	if !matched {
		t.Fatalf("load result = %T, want TaskStored", loadResult)
	}
	if loaded.Value.Title != "Label receipts" || loaded.Value.State != "draft" {
		t.Fatalf("loaded task = %#v", loaded.Value)
	}
}

func TestTaskBrowserStorageRejectsInvalidState(t *testing.T) {
	result := SaveTask(newTestBrowserStorage(), StoredTask{
		ID:                     "task-1",
		OwnerKind:              "user",
		OwnerID:                "user-1",
		Title:                  "Label receipts",
		Description:            "Extract totals.",
		TaskType:               "general",
		RewardKind:             "none",
		ParticipationPolicy:    "open",
		AssigneeScope:          "user",
		ReservationExpiryHours: 48,
		State:                  "unknown",
		VisibilityKind:         "public",
		SeriesKind:             "standalone",
		ResponseSchemaJSON:     `{"kind":"freeform"}`,
		PayloadKind:            "none",
		CreatedBy:              "user-1",
	})
	rejected, matched := result.(TaskStorageRejected)
	if !matched {
		t.Fatalf("result = %T, want TaskStorageRejected", result)
	}
	if rejected.Reason != "task state is invalid" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestNotificationBrowserStorageListsAndMarksRead(t *testing.T) {
	storage := newTestBrowserStorage()
	first := StoredNotification{
		ID:              "notification-1",
		RecipientUserID: "user-2",
		ActorUserID:     "user-1",
		Kind:            "submission_created",
		SubjectKind:     "submission",
		SubjectID:       "submission-1",
		State:           "unread",
		MetadataJSON:    `{"task_id":"task-1"}`,
		CreatedAt:       "2026-07-01T10:00:00Z",
	}
	second := first
	second.ID = "notification-2"
	second.Kind = "submission_accepted"

	if _, matched := SaveNotification(storage, first).(NotificationStored); !matched {
		t.Fatalf("first notification save was rejected")
	}
	if _, matched := SaveNotification(storage, second).(NotificationStored); !matched {
		t.Fatalf("second notification save was rejected")
	}

	pageResult := NewNotificationPage(1, 0)
	page, pageMatched := pageResult.(NotificationPageAccepted)
	if !pageMatched {
		t.Fatalf("page result = %T, want NotificationPageAccepted", pageResult)
	}
	listResult := ListNotifications(storage, "user-2", page.Value)
	listed, listedMatched := listResult.(NotificationsStored)
	if !listedMatched {
		t.Fatalf("list result = %T, want NotificationsStored", listResult)
	}
	if len(listed.Values) != 1 || listed.Values[0].ID != "notification-2" {
		t.Fatalf("first page = %#v", listed.Values)
	}

	markResult := MarkNotificationRead(storage, "notification-2", "user-2")
	marked, markedMatched := markResult.(NotificationStored)
	if !markedMatched {
		t.Fatalf("mark result = %T, want NotificationStored", markResult)
	}
	if marked.Value.State != "read" {
		t.Fatalf("state = %q, want read", marked.Value.State)
	}
}

func TestNotificationPageRejectsInvalidValues(t *testing.T) {
	result := NewNotificationPage(0, 0)
	rejected, matched := result.(NotificationPageRejected)
	if !matched {
		t.Fatalf("result = %T, want NotificationPageRejected", result)
	}
	if rejected.Reason != "notification page limit is invalid" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestNotificationBrowserStorageRejectsMismatchedActor(t *testing.T) {
	storage := newTestBrowserStorage()
	if _, matched := SaveNotification(storage, StoredNotification{
		ID:              "notification-1",
		RecipientUserID: "user-2",
		ActorUserID:     "user-1",
		Kind:            "submission_created",
		SubjectKind:     "submission",
		SubjectID:       "submission-1",
		State:           "unread",
		MetadataJSON:    `{}`,
		CreatedAt:       "2026-07-01T10:00:00Z",
	}).(NotificationStored); !matched {
		t.Fatalf("notification save was rejected")
	}

	result := MarkNotificationRead(storage, "notification-1", "user-3")
	rejected, matched := result.(NotificationStorageRejected)
	if !matched {
		t.Fatalf("result = %T, want NotificationStorageRejected", result)
	}
	if rejected.Reason != "notification does not belong to actor" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestAttachmentBrowserStorageRoundTrip(t *testing.T) {
	storage := newTestBrowserStorage()
	attachment := StoredAttachment{
		Name:        "brief.txt",
		ContentType: "text/plain",
		SizeBytes:   5,
		DataURL:     "data:text/plain;base64,aGVsbG8=",
	}

	saveResult := SaveAttachments(storage, "task", "task-1", []StoredAttachment{attachment})
	saved, savedMatched := saveResult.(AttachmentsStored)
	if !savedMatched {
		t.Fatalf("save result = %T, want AttachmentsStored", saveResult)
	}
	if saved.Values[0].ParentKind != "task" || saved.Values[0].ParentID != "task-1" {
		t.Fatalf("saved parent = %s/%s", saved.Values[0].ParentKind, saved.Values[0].ParentID)
	}

	listResult := ListAttachments(storage, "task", "task-1")
	listed, listedMatched := listResult.(AttachmentsStored)
	if !listedMatched {
		t.Fatalf("list result = %T, want AttachmentsStored", listResult)
	}
	if len(listed.Values) != 1 {
		t.Fatalf("listed attachment count = %d, want 1", len(listed.Values))
	}
	if listed.Values[0].Name != "brief.txt" {
		t.Fatalf("attachment name = %q", listed.Values[0].Name)
	}
}

func TestAttachmentBrowserStorageRejectsInvalidParentKind(t *testing.T) {
	result := SaveAttachments(newTestBrowserStorage(), "comment", "comment-1", nil)
	rejected, matched := result.(AttachmentStorageRejected)
	if !matched {
		t.Fatalf("result = %T, want AttachmentStorageRejected", result)
	}
	if rejected.Reason != "attachment parent kind is invalid" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestAttachmentBrowserStorageRejectsOversizedAttachment(t *testing.T) {
	result := SaveAttachments(newTestBrowserStorage(), "submission", "submission-1", []StoredAttachment{{
		Name:        "large.txt",
		ContentType: "text/plain",
		SizeBytes:   500*1024 + 1,
		DataURL:     "data:text/plain;base64,aGVsbG8=",
	}})
	rejected, matched := result.(AttachmentStorageRejected)
	if !matched {
		t.Fatalf("result = %T, want AttachmentStorageRejected", result)
	}
	if rejected.Reason != "attachment size is invalid" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestAttachmentBrowserStorageRejectsTooManyAttachments(t *testing.T) {
	attachments := make([]StoredAttachment, 0, maxStoredAttachments+1)
	for index := 0; index < maxStoredAttachments+1; index++ {
		attachments = append(attachments, StoredAttachment{
			Name:        "brief.txt",
			ContentType: "text/plain",
			SizeBytes:   5,
			DataURL:     "data:text/plain;base64,aGVsbG8=",
		})
	}

	result := SaveAttachments(newTestBrowserStorage(), "task", "task-1", attachments)
	rejected, matched := result.(AttachmentStorageRejected)
	if !matched {
		t.Fatalf("result = %T, want AttachmentStorageRejected", result)
	}
	if rejected.Reason != "too many attachments" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestAttachmentBrowserStorageRejectsMismatchedStoredRecord(t *testing.T) {
	storage := newTestBrowserStorage()
	keyResult := NewStorageKey("attachments:task:task-1")
	key := keyResult.(StorageKeyAccepted)
	storage.values[key.Value.String()] = `[{"parent_kind":"submission","parent_id":"submission-1","name":"brief.txt","content_type":"text/plain","size_bytes":5,"data_url":"data:text/plain;base64,aGVsbG8="}]`

	result := ListAttachments(storage, "task", "task-1")
	rejected, matched := result.(AttachmentStorageRejected)
	if !matched {
		t.Fatalf("result = %T, want AttachmentStorageRejected", result)
	}
	if rejected.Reason != "attachment storage key contains mismatched record" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}
