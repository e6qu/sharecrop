package wasmdemo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/notification"
)

// NotificationBrowserStore implements notification.Store against
// BrowserStorage, so the real notification.Service (the same code
// cmd/sharecrop runs against Postgres) can serve the browser demo directly,
// instead of internal/wasmdemo's own from-scratch notification handling.
type NotificationBrowserStore struct {
	storage BrowserStorage
}

func NewNotificationBrowserStore(storage BrowserStorage) NotificationBrowserStore {
	return NotificationBrowserStore{storage: storage}
}

type storedNotificationRecord struct {
	ID            string `json:"id"`
	RecipientID   string `json:"recipient_id"`
	ActorID       string `json:"actor_id"`
	Kind          string `json:"kind"`
	SubjectKind   string `json:"subject_kind"`
	SubjectID     string `json:"subject_id"`
	State         string `json:"state"`
	MetadataJSON  string `json:"metadata_json"`
	CreatedAtUnix int64  `json:"created_at_unix"`
}

func notificationRecordKey(id string) string {
	return "notification:" + id
}

func notificationRecipientIndexKey(recipientID string) string {
	return "notification:index:recipient:" + recipientID
}

func (store NotificationBrowserStore) Create(_ context.Context, value notification.Notification) notification.CreateStoreResult {
	record := storedNotificationRecord{
		ID:            value.ID.String(),
		RecipientID:   value.RecipientID.String(),
		ActorID:       value.ActorID.String(),
		Kind:          value.Kind.String(),
		SubjectKind:   value.Subject.Kind,
		SubjectID:     value.Subject.ID,
		State:         value.State.String(),
		MetadataJSON:  value.Metadata.JSON,
		CreatedAtUnix: value.CreatedAt.UnixNano(),
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return notification.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "notification encoding failed")}
	}
	keyResult := NewStorageKey(notificationRecordKey(record.ID))
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return notification.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, keyResult.(StorageKeyRejected).Reason)}
	}
	if _, matched := store.storage.Put(key.Value, string(encoded)).(StorageWritten); !matched {
		return notification.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "notification write failed")}
	}
	indexResult := appendStringIndex(store.storage, notificationRecipientIndexKey(record.RecipientID), record.ID, "notification")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return notification.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "notification index update failed")}
	}
	return notification.CreateStoreAccepted{}
}

func (store NotificationBrowserStore) List(_ context.Context, recipient core.UserID, page core.Page) notification.ListStoreResult {
	indexResult := loadStringIndex(store.storage, notificationRecipientIndexKey(recipient.String()), "notification")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return notification.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, indexResult.(stringIndexRejected).reason)}
	}

	// Newest first, matching internal/notification.MemoryStore's ordering -
	// the index is append order (oldest first), so walk it in reverse.
	matching := make([]notification.Notification, 0, len(loaded.values))
	for index := len(loaded.values) - 1; index >= 0; index-- {
		value, found, rejected := store.loadRecord(loaded.values[index])
		if rejected != nil {
			return notification.ListStoreRejected{Reason: *rejected}
		}
		if found {
			matching = append(matching, value)
		}
	}

	start := page.Offset()
	if start > len(matching) {
		start = len(matching)
	}
	end := start + page.Limit()
	if end > len(matching) {
		end = len(matching)
	}
	values := make([]notification.Notification, end-start)
	copy(values, matching[start:end])
	return notification.ListStoreAccepted{Values: values}
}

func (store NotificationBrowserStore) MarkRead(_ context.Context, recipient core.UserID, id core.NotificationID) notification.MarkReadStoreResult {
	value, found, rejected := store.loadRecord(id.String())
	if rejected != nil {
		return notification.MarkReadStoreRejected{Reason: *rejected}
	}
	if !found || value.RecipientID != recipient {
		return notification.MarkReadStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "notification was not found")}
	}
	value.State = notification.StateRead
	record := storedNotificationRecord{
		ID:            value.ID.String(),
		RecipientID:   value.RecipientID.String(),
		ActorID:       value.ActorID.String(),
		Kind:          value.Kind.String(),
		SubjectKind:   value.Subject.Kind,
		SubjectID:     value.Subject.ID,
		State:         value.State.String(),
		MetadataJSON:  value.Metadata.JSON,
		CreatedAtUnix: value.CreatedAt.UnixNano(),
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return notification.MarkReadStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "notification encoding failed")}
	}
	keyResult := NewStorageKey(notificationRecordKey(record.ID))
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return notification.MarkReadStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, keyResult.(StorageKeyRejected).Reason)}
	}
	if _, matched := store.storage.Put(key.Value, string(encoded)).(StorageWritten); !matched {
		return notification.MarkReadStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "notification write failed")}
	}
	return notification.MarkReadStoreAccepted{Value: value}
}

// loadRecord returns (value, found, rejectedReason). found is false only for
// a missing key (not an error); rejectedReason is non-nil only for a genuine
// storage/decoding failure.
func (store NotificationBrowserStore) loadRecord(id string) (notification.Notification, bool, *core.DomainError) {
	keyResult := NewStorageKey(notificationRecordKey(id))
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, keyResult.(StorageKeyRejected).Reason)
		return notification.Notification{}, false, &reason
	}
	readResult := store.storage.Get(key.Value)
	if _, missing := readResult.(StorageMissing); missing {
		return notification.Notification{}, false, nil
	}
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "notification read failed")
		return notification.Notification{}, false, &reason
	}
	var record storedNotificationRecord
	if err := json.Unmarshal([]byte(read.Value), &record); err != nil {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "notification decoding failed")
		return notification.Notification{}, false, &reason
	}
	recipientResult := core.ParseUserID(record.RecipientID)
	recipient, recipientMatched := recipientResult.(core.UserIDCreated)
	if !recipientMatched {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "notification recipient id is invalid")
		return notification.Notification{}, false, &reason
	}
	actorResult := core.ParseUserID(record.ActorID)
	actor, actorMatched := actorResult.(core.UserIDCreated)
	if !actorMatched {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "notification actor id is invalid")
		return notification.Notification{}, false, &reason
	}
	idResult := core.ParseNotificationID(record.ID)
	notificationID, idMatched := idResult.(core.NotificationIDCreated)
	if !idMatched {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "notification id is invalid")
		return notification.Notification{}, false, &reason
	}
	return notification.Notification{
		ID:          notificationID.Value,
		RecipientID: recipient.Value,
		ActorID:     actor.Value,
		Kind:        notification.KindFromString(record.Kind),
		Subject:     notification.Subject{Kind: record.SubjectKind, ID: record.SubjectID},
		State:       notification.StateFromString(record.State),
		Metadata:    notification.Metadata{JSON: record.MetadataJSON},
		CreatedAt:   time.Unix(0, record.CreatedAtUnix).UTC(),
	}, true, nil
}
