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

type StoredSavedQueueView struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Scope       string `json:"scope"`
	Name        string `json:"name"`
	Query       string `json:"query"`
	StateFilter string `json:"state_filter"`
	TypeFilter  string `json:"type_filter"`
	Sort        string `json:"sort"`
}

type StoredAttachment struct {
	ParentKind  string `json:"parent_kind"`
	ParentID    string `json:"parent_id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	SizeBytes   int    `json:"size_bytes"`
	DataURL     string `json:"data_url"`
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

type SavedQueueViewStorageResult interface {
	savedQueueViewStorageResult()
}

type SavedQueueViewStored struct {
	Value StoredSavedQueueView
}

type SavedQueueViewsStored struct {
	Values []StoredSavedQueueView
}

type SavedQueueViewStorageRejected struct {
	Reason string
}

func (SavedQueueViewStored) savedQueueViewStorageResult()          {}
func (SavedQueueViewsStored) savedQueueViewStorageResult()         {}
func (SavedQueueViewStorageRejected) savedQueueViewStorageResult() {}

func SaveSavedQueueView(storage BrowserStorage, view StoredSavedQueueView) SavedQueueViewStorageResult {
	cleaned := StoredSavedQueueView{
		ID:          strings.TrimSpace(view.ID),
		UserID:      strings.TrimSpace(view.UserID),
		Scope:       strings.TrimSpace(view.Scope),
		Name:        strings.TrimSpace(view.Name),
		Query:       strings.TrimSpace(view.Query),
		StateFilter: strings.TrimSpace(view.StateFilter),
		TypeFilter:  strings.TrimSpace(view.TypeFilter),
		Sort:        strings.TrimSpace(view.Sort),
	}
	if cleaned.ID == "" {
		return SavedQueueViewStorageRejected{Reason: "saved queue view id is required"}
	}
	if cleaned.UserID == "" {
		return SavedQueueViewStorageRejected{Reason: "saved queue view actor is required"}
	}
	if cleaned.Name == "" {
		return SavedQueueViewStorageRejected{Reason: "saved queue view name is required"}
	}
	if !validSavedQueueScope(cleaned.Scope) {
		return SavedQueueViewStorageRejected{Reason: "saved queue view scope is invalid"}
	}
	keyResult := savedQueueViewKey(cleaned.UserID, cleaned.Scope, cleaned.Name)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return SavedQueueViewStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	existingResult := storage.Get(key.Value)
	if read, matched := existingResult.(StorageRead); matched {
		var existing StoredSavedQueueView
		if err := json.Unmarshal([]byte(read.Value), &existing); err != nil {
			return SavedQueueViewStorageRejected{Reason: "saved queue view decoding failed"}
		}
		if rejectedReason := validateStoredSavedQueueView(existing); rejectedReason != "" {
			return SavedQueueViewStorageRejected{Reason: rejectedReason}
		}
		if strings.TrimSpace(existing.UserID) != cleaned.UserID || strings.TrimSpace(existing.Scope) != cleaned.Scope || strings.TrimSpace(existing.Name) != cleaned.Name {
			return SavedQueueViewStorageRejected{Reason: "saved queue view storage key contains mismatched record"}
		}
		cleaned.ID = strings.TrimSpace(existing.ID)
	} else if _, missing := existingResult.(StorageMissing); !missing {
		return SavedQueueViewStorageRejected{Reason: savedQueueViewReadReason(existingResult)}
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return SavedQueueViewStorageRejected{Reason: "saved queue view encoding failed"}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return SavedQueueViewStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	indexResult := appendSavedQueueViewIndex(storage, cleaned.UserID, cleaned.Scope, key.Value.String())
	if _, matched := indexResult.(SavedQueueViewsStored); !matched {
		return indexResult
	}
	return SavedQueueViewStored{Value: cleaned}
}

func ListSavedQueueViews(storage BrowserStorage, userID string, scope string) SavedQueueViewStorageResult {
	cleanUserID := strings.TrimSpace(userID)
	cleanScope := strings.TrimSpace(scope)
	if cleanUserID == "" {
		return SavedQueueViewStorageRejected{Reason: "saved queue view actor is required"}
	}
	if cleanScope == "" {
		teamResult := ListSavedQueueViews(storage, cleanUserID, "team_work")
		teamViews, teamMatched := teamResult.(SavedQueueViewsStored)
		if !teamMatched {
			return teamResult
		}
		orgResult := ListSavedQueueViews(storage, cleanUserID, "organization_tasks")
		orgViews, orgMatched := orgResult.(SavedQueueViewsStored)
		if !orgMatched {
			return orgResult
		}
		values := make([]StoredSavedQueueView, 0, len(teamViews.Values)+len(orgViews.Values))
		values = append(values, teamViews.Values...)
		values = append(values, orgViews.Values...)
		return SavedQueueViewsStored{Values: values}
	}
	if !validSavedQueueScope(cleanScope) {
		return SavedQueueViewStorageRejected{Reason: "saved queue view scope is invalid"}
	}
	keysResult := loadSavedQueueViewIndex(storage, cleanUserID, cleanScope)
	keys, keysMatched := keysResult.(savedQueueViewKeysLoaded)
	if !keysMatched {
		return SavedQueueViewStorageRejected{Reason: keysResult.(savedQueueViewKeysRejected).Reason}
	}
	values := make([]StoredSavedQueueView, 0, len(keys.Values))
	for index := range keys.Values {
		keyResult := NewStorageKey(keys.Values[index])
		key, keyMatched := keyResult.(StorageKeyAccepted)
		if !keyMatched {
			return SavedQueueViewStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
		}
		readResult := storage.Get(key.Value)
		read, readMatched := readResult.(StorageRead)
		if !readMatched {
			return SavedQueueViewStorageRejected{Reason: savedQueueViewReadReason(readResult)}
		}
		var view StoredSavedQueueView
		if err := json.Unmarshal([]byte(read.Value), &view); err != nil {
			return SavedQueueViewStorageRejected{Reason: "saved queue view decoding failed"}
		}
		if rejectedReason := validateStoredSavedQueueView(view); rejectedReason != "" {
			return SavedQueueViewStorageRejected{Reason: rejectedReason}
		}
		if strings.TrimSpace(view.UserID) != cleanUserID || strings.TrimSpace(view.Scope) != cleanScope {
			return SavedQueueViewStorageRejected{Reason: "saved queue view index contains mismatched record"}
		}
		values = append(values, view)
	}
	return SavedQueueViewsStored{Values: values}
}

type savedQueueViewKeysResult interface {
	savedQueueViewKeysResult()
}

type savedQueueViewKeysLoaded struct {
	Values []string
}

type savedQueueViewKeysRejected struct {
	Reason string
}

func (savedQueueViewKeysLoaded) savedQueueViewKeysResult()   {}
func (savedQueueViewKeysRejected) savedQueueViewKeysResult() {}

func savedQueueViewKey(userID string, scope string, name string) StorageKeyResult {
	return NewStorageKey("saved_queue_view:" + userID + ":" + scope + ":" + name)
}

func savedQueueViewIndexKey(userID string, scope string) StorageKeyResult {
	return NewStorageKey("saved_queue_view:index:" + userID + ":" + scope)
}

func loadSavedQueueViewIndex(storage BrowserStorage, userID string, scope string) savedQueueViewKeysResult {
	keyResult := savedQueueViewIndexKey(userID, scope)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return savedQueueViewKeysRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	if _, missing := readResult.(StorageMissing); missing {
		return savedQueueViewKeysLoaded{Values: []string{}}
	}
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return savedQueueViewKeysRejected{Reason: savedQueueViewReadReason(readResult)}
	}
	var keys []string
	if err := json.Unmarshal([]byte(read.Value), &keys); err != nil {
		return savedQueueViewKeysRejected{Reason: "saved queue view index decoding failed"}
	}
	for index := range keys {
		if strings.TrimSpace(keys[index]) == "" {
			return savedQueueViewKeysRejected{Reason: "saved queue view index contains an invalid key"}
		}
	}
	return savedQueueViewKeysLoaded{Values: keys}
}

func appendSavedQueueViewIndex(storage BrowserStorage, userID string, scope string, viewKey string) SavedQueueViewStorageResult {
	keysResult := loadSavedQueueViewIndex(storage, userID, scope)
	keys, keysMatched := keysResult.(savedQueueViewKeysLoaded)
	if !keysMatched {
		return SavedQueueViewStorageRejected{Reason: keysResult.(savedQueueViewKeysRejected).Reason}
	}
	for index := range keys.Values {
		if keys.Values[index] == viewKey {
			return SavedQueueViewsStored{Values: []StoredSavedQueueView{}}
		}
	}
	keys.Values = append(keys.Values, viewKey)
	encoded, err := json.Marshal(keys.Values)
	if err != nil {
		return SavedQueueViewStorageRejected{Reason: "saved queue view index encoding failed"}
	}
	keyResult := savedQueueViewIndexKey(userID, scope)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return SavedQueueViewStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return SavedQueueViewStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	return SavedQueueViewsStored{Values: []StoredSavedQueueView{}}
}

func savedQueueViewReadReason(result StorageReadResult) string {
	switch rejected := result.(type) {
	case StorageMissing:
		return rejected.Reason
	case StorageReadRejected:
		return rejected.Reason
	default:
		return "saved queue view read failed"
	}
}

func validateStoredSavedQueueView(view StoredSavedQueueView) string {
	if strings.TrimSpace(view.ID) == "" {
		return "saved queue view id is required"
	}
	if strings.TrimSpace(view.UserID) == "" {
		return "saved queue view actor is required"
	}
	if strings.TrimSpace(view.Name) == "" {
		return "saved queue view name is required"
	}
	if !validSavedQueueScope(strings.TrimSpace(view.Scope)) {
		return "saved queue view scope is invalid"
	}
	return ""
}

func validSavedQueueScope(value string) bool {
	switch value {
	case "team_work", "organization_tasks":
		return true
	default:
		return false
	}
}

type AttachmentStorageResult interface {
	attachmentStorageResult()
}

type AttachmentsStored struct {
	Values []StoredAttachment
}

type AttachmentStorageRejected struct {
	Reason string
}

func (AttachmentsStored) attachmentStorageResult()         {}
func (AttachmentStorageRejected) attachmentStorageResult() {}

func SaveAttachments(storage BrowserStorage, parentKind string, parentID string, attachments []StoredAttachment) AttachmentStorageResult {
	cleanKind := strings.TrimSpace(parentKind)
	cleanID := strings.TrimSpace(parentID)
	if !validAttachmentParentKind(cleanKind) {
		return AttachmentStorageRejected{Reason: "attachment parent kind is invalid"}
	}
	if cleanID == "" {
		return AttachmentStorageRejected{Reason: "attachment parent id is required"}
	}
	values := make([]StoredAttachment, 0, len(attachments))
	for index := range attachments {
		cleaned := StoredAttachment{
			ParentKind:  cleanKind,
			ParentID:    cleanID,
			Name:        strings.TrimSpace(attachments[index].Name),
			ContentType: strings.TrimSpace(attachments[index].ContentType),
			SizeBytes:   attachments[index].SizeBytes,
			DataURL:     strings.TrimSpace(attachments[index].DataURL),
		}
		if reason := validateStoredAttachment(cleaned); reason != "" {
			return AttachmentStorageRejected{Reason: reason}
		}
		values = append(values, cleaned)
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return AttachmentStorageRejected{Reason: "attachments encoding failed"}
	}
	keyResult := NewStorageKey("attachments:" + cleanKind + ":" + cleanID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return AttachmentStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return AttachmentStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	return AttachmentsStored{Values: values}
}

func ListAttachments(storage BrowserStorage, parentKind string, parentID string) AttachmentStorageResult {
	cleanKind := strings.TrimSpace(parentKind)
	cleanID := strings.TrimSpace(parentID)
	if !validAttachmentParentKind(cleanKind) {
		return AttachmentStorageRejected{Reason: "attachment parent kind is invalid"}
	}
	if cleanID == "" {
		return AttachmentStorageRejected{Reason: "attachment parent id is required"}
	}
	keyResult := NewStorageKey("attachments:" + cleanKind + ":" + cleanID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return AttachmentStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return AttachmentStorageRejected{Reason: attachmentReadReason(readResult)}
	}
	var attachments []StoredAttachment
	if err := json.Unmarshal([]byte(read.Value), &attachments); err != nil {
		return AttachmentStorageRejected{Reason: "attachments decoding failed"}
	}
	for index := range attachments {
		if reason := validateStoredAttachment(attachments[index]); reason != "" {
			return AttachmentStorageRejected{Reason: reason}
		}
		if attachments[index].ParentKind != cleanKind || attachments[index].ParentID != cleanID {
			return AttachmentStorageRejected{Reason: "attachment storage key contains mismatched record"}
		}
	}
	return AttachmentsStored{Values: attachments}
}

func validateStoredAttachment(attachment StoredAttachment) string {
	if !validAttachmentParentKind(strings.TrimSpace(attachment.ParentKind)) {
		return "attachment parent kind is invalid"
	}
	if strings.TrimSpace(attachment.ParentID) == "" {
		return "attachment parent id is required"
	}
	if strings.TrimSpace(attachment.Name) == "" {
		return "attachment name is required"
	}
	if strings.TrimSpace(attachment.ContentType) == "" {
		return "attachment content type is required"
	}
	if attachment.SizeBytes <= 0 || attachment.SizeBytes > 500*1024 {
		return "attachment size is invalid"
	}
	if strings.TrimSpace(attachment.DataURL) == "" {
		return "attachment data URL is required"
	}
	return ""
}

func validAttachmentParentKind(value string) bool {
	switch value {
	case "task", "submission":
		return true
	default:
		return false
	}
}

func attachmentReadReason(result StorageReadResult) string {
	switch rejected := result.(type) {
	case StorageMissing:
		return rejected.Reason
	case StorageReadRejected:
		return rejected.Reason
	default:
		return "attachments read failed"
	}
}
