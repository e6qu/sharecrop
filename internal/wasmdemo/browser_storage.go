package wasmdemo

import (
	"encoding/json"
	"strings"
)

type BrowserStorage interface {
	Put(StorageKey, string) StorageWriteResult
	Get(StorageKey) StorageReadResult
}

type StorageKey struct {
	value string
}

func (key StorageKey) String() string {
	return key.value
}

type StorageKeyResult interface {
	storageKeyResult()
}

type StorageKeyAccepted struct {
	Value StorageKey
}

type StorageKeyRejected struct {
	Reason string
}

func (StorageKeyAccepted) storageKeyResult() {}
func (StorageKeyRejected) storageKeyResult() {}

func NewStorageKey(raw string) StorageKeyResult {
	value := strings.TrimSpace(raw)
	if value == "" || strings.Contains(value, "\n") || strings.Contains(value, "\x00") {
		return StorageKeyRejected{Reason: "storage key is invalid"}
	}
	return StorageKeyAccepted{Value: StorageKey{value: value}}
}

type StorageWriteResult interface {
	storageWriteResult()
}

type StorageWritten struct{}
type StorageWriteRejected struct{ Reason string }

func (StorageWritten) storageWriteResult()       {}
func (StorageWriteRejected) storageWriteResult() {}

type StorageReadResult interface {
	storageReadResult()
}

type StorageRead struct{ Value string }
type StorageMissing struct{ Reason string }
type StorageReadRejected struct{ Reason string }

func (StorageRead) storageReadResult()         {}
func (StorageMissing) storageReadResult()      {}
func (StorageReadRejected) storageReadResult() {}

type StoredModerationTriage struct {
	ReportID       string `json:"report_id"`
	State          string `json:"state"`
	ResolutionNote string `json:"resolution_note"`
	UpdatedBy      string `json:"updated_by"`
	UpdatedAt      string `json:"updated_at"`
}

type ModerationTriageStorageResult interface {
	moderationTriageStorageResult()
}

type ModerationTriageStored struct {
	Value StoredModerationTriage
}

type ModerationTriageStorageRejected struct {
	Reason string
}

func (ModerationTriageStored) moderationTriageStorageResult()          {}
func (ModerationTriageStorageRejected) moderationTriageStorageResult() {}

func SaveModerationTriage(storage BrowserStorage, triage StoredModerationTriage) ModerationTriageStorageResult {
	keyResult := NewStorageKey("moderation_triage:" + strings.TrimSpace(triage.ReportID))
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return ModerationTriageStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	if !validStoredModerationState(triage.State) {
		return ModerationTriageStorageRejected{Reason: "moderation triage state is invalid"}
	}
	encoded, err := json.Marshal(triage)
	if err != nil {
		return ModerationTriageStorageRejected{Reason: "moderation triage encoding failed"}
	}
	result := storage.Put(key.Value, string(encoded))
	if _, matched := result.(StorageWritten); !matched {
		return ModerationTriageStorageRejected{Reason: result.(StorageWriteRejected).Reason}
	}
	return ModerationTriageStored{Value: triage}
}

func LoadModerationTriage(storage BrowserStorage, reportID string) ModerationTriageStorageResult {
	keyResult := NewStorageKey("moderation_triage:" + strings.TrimSpace(reportID))
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return ModerationTriageStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	result := storage.Get(key.Value)
	read, readMatched := result.(StorageRead)
	if !readMatched {
		switch rejected := result.(type) {
		case StorageMissing:
			return ModerationTriageStorageRejected{Reason: rejected.Reason}
		case StorageReadRejected:
			return ModerationTriageStorageRejected{Reason: rejected.Reason}
		default:
			return ModerationTriageStorageRejected{Reason: "moderation triage read failed"}
		}
	}
	var triage StoredModerationTriage
	if err := json.Unmarshal([]byte(read.Value), &triage); err != nil {
		return ModerationTriageStorageRejected{Reason: "moderation triage decoding failed"}
	}
	if !validStoredModerationState(triage.State) {
		return ModerationTriageStorageRejected{Reason: "moderation triage state is invalid"}
	}
	return ModerationTriageStored{Value: triage}
}

func validStoredModerationState(value string) bool {
	switch value {
	case "open", "resolved", "dismissed":
		return true
	default:
		return false
	}
}
