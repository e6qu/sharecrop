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

type StoredPrivacyRequest struct {
	ID                 string `json:"id"`
	Kind               string `json:"kind"`
	Status             string `json:"status"`
	RequestedBy        string `json:"requested_by"`
	ExportJSON         string `json:"export_json"`
	ResolutionNote     string `json:"resolution_note"`
	CreatedAt          string `json:"created_at"`
	ResolvedAt         string `json:"resolved_at"`
	RedactedFieldCount int    `json:"redacted_field_count"`
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

type PrivacyRequestStorageResult interface {
	privacyRequestStorageResult()
}

type PrivacyRequestStored struct {
	Value StoredPrivacyRequest
}

type PrivacyRequestsStored struct {
	Values []StoredPrivacyRequest
}

type PrivacyRequestStorageRejected struct {
	Reason string
}

func (PrivacyRequestStored) privacyRequestStorageResult()          {}
func (PrivacyRequestsStored) privacyRequestStorageResult()         {}
func (PrivacyRequestStorageRejected) privacyRequestStorageResult() {}

func SavePrivacyRequest(storage BrowserStorage, request StoredPrivacyRequest) PrivacyRequestStorageResult {
	keyResult := NewStorageKey("privacy_request:" + strings.TrimSpace(request.ID))
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return PrivacyRequestStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	if !validStoredPrivacyKind(request.Kind) {
		return PrivacyRequestStorageRejected{Reason: "privacy request kind is invalid"}
	}
	if !validStoredPrivacyStatus(request.Status) {
		return PrivacyRequestStorageRejected{Reason: "privacy request status is invalid"}
	}
	if strings.TrimSpace(request.RequestedBy) == "" {
		return PrivacyRequestStorageRejected{Reason: "privacy request actor is required"}
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return PrivacyRequestStorageRejected{Reason: "privacy request encoding failed"}
	}
	result := storage.Put(key.Value, string(encoded))
	if _, matched := result.(StorageWritten); !matched {
		return PrivacyRequestStorageRejected{Reason: result.(StorageWriteRejected).Reason}
	}
	indexResult := appendPrivacyRequestIndex(storage, request.ID)
	if _, matched := indexResult.(PrivacyRequestsStored); !matched {
		return indexResult
	}
	return PrivacyRequestStored{Value: request}
}

func ListPrivacyRequests(storage BrowserStorage) PrivacyRequestStorageResult {
	idsResult := loadPrivacyRequestIndex(storage)
	ids, idsMatched := idsResult.(privacyRequestIDsLoaded)
	if !idsMatched {
		return PrivacyRequestStorageRejected{Reason: idsResult.(privacyRequestIDsRejected).Reason}
	}
	values := make([]StoredPrivacyRequest, 0, len(ids.Values))
	for index := range ids.Values {
		keyResult := NewStorageKey("privacy_request:" + ids.Values[index])
		key, keyMatched := keyResult.(StorageKeyAccepted)
		if !keyMatched {
			return PrivacyRequestStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
		}
		readResult := storage.Get(key.Value)
		read, readMatched := readResult.(StorageRead)
		if !readMatched {
			return PrivacyRequestStorageRejected{Reason: privacyReadReason(readResult)}
		}
		var request StoredPrivacyRequest
		if err := json.Unmarshal([]byte(read.Value), &request); err != nil {
			return PrivacyRequestStorageRejected{Reason: "privacy request decoding failed"}
		}
		if !validStoredPrivacyKind(request.Kind) {
			return PrivacyRequestStorageRejected{Reason: "privacy request kind is invalid"}
		}
		if !validStoredPrivacyStatus(request.Status) {
			return PrivacyRequestStorageRejected{Reason: "privacy request status is invalid"}
		}
		values = append(values, request)
	}
	return PrivacyRequestsStored{Values: values}
}

type privacyRequestIDsResult interface {
	privacyRequestIDsResult()
}

type privacyRequestIDsLoaded struct {
	Values []string
}

type privacyRequestIDsRejected struct {
	Reason string
}

func (privacyRequestIDsLoaded) privacyRequestIDsResult()   {}
func (privacyRequestIDsRejected) privacyRequestIDsResult() {}

func loadPrivacyRequestIndex(storage BrowserStorage) privacyRequestIDsResult {
	keyResult := NewStorageKey("privacy_request:index")
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return privacyRequestIDsRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	if _, missing := readResult.(StorageMissing); missing {
		return privacyRequestIDsLoaded{Values: []string{}}
	}
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return privacyRequestIDsRejected{Reason: privacyReadReason(readResult)}
	}
	var ids []string
	if err := json.Unmarshal([]byte(read.Value), &ids); err != nil {
		return privacyRequestIDsRejected{Reason: "privacy request index decoding failed"}
	}
	for index := range ids {
		if strings.TrimSpace(ids[index]) == "" {
			return privacyRequestIDsRejected{Reason: "privacy request index contains an invalid id"}
		}
	}
	return privacyRequestIDsLoaded{Values: ids}
}

func appendPrivacyRequestIndex(storage BrowserStorage, id string) PrivacyRequestStorageResult {
	idsResult := loadPrivacyRequestIndex(storage)
	ids, idsMatched := idsResult.(privacyRequestIDsLoaded)
	if !idsMatched {
		return PrivacyRequestStorageRejected{Reason: idsResult.(privacyRequestIDsRejected).Reason}
	}
	cleanID := strings.TrimSpace(id)
	for index := range ids.Values {
		if ids.Values[index] == cleanID {
			return PrivacyRequestsStored{Values: []StoredPrivacyRequest{}}
		}
	}
	ids.Values = append(ids.Values, cleanID)
	encoded, err := json.Marshal(ids.Values)
	if err != nil {
		return PrivacyRequestStorageRejected{Reason: "privacy request index encoding failed"}
	}
	keyResult := NewStorageKey("privacy_request:index")
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return PrivacyRequestStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return PrivacyRequestStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	return PrivacyRequestsStored{Values: []StoredPrivacyRequest{}}
}

func privacyReadReason(result StorageReadResult) string {
	switch rejected := result.(type) {
	case StorageMissing:
		return rejected.Reason
	case StorageReadRejected:
		return rejected.Reason
	default:
		return "privacy request read failed"
	}
}

func validStoredPrivacyKind(value string) bool {
	switch value {
	case "data_export", "sensitive_field_deletion":
		return true
	default:
		return false
	}
}

func validStoredPrivacyStatus(value string) bool {
	switch value {
	case "queued", "resolved":
		return true
	default:
		return false
	}
}
