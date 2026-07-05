package wasmdemo

import (
	"encoding/json"
	"strings"
)

const maxStoredAttachments = 5

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

type StoredTask struct {
	ID                     string   `json:"id"`
	CreatedBy              string   `json:"created_by"`
	OwnerKind              string   `json:"owner_kind"`
	OwnerID                string   `json:"owner_id"`
	Title                  string   `json:"title"`
	State                  string   `json:"state"`
	Description            string   `json:"description"`
	TaskType               string   `json:"task_type"`
	RewardKind             string   `json:"reward_kind"`
	RewardCollectibleIDs   []string `json:"reward_collectible_ids"`
	RewardCollectibleCount int      `json:"reward_collectible_count"`
	RewardCreditAmount     int64    `json:"reward_credit_amount"`
	ParticipationPolicy    string   `json:"participation_policy"`
	ReservationExpiryHours int      `json:"reservation_expiry_hours"`
	AssigneeScope          string   `json:"assignee_scope"`
	VisibilityKind         string   `json:"visibility_kind"`
	VisibilityID           string   `json:"visibility_id"`
	SeriesKind             string   `json:"series_kind"`
	SeriesPosition         int      `json:"series_position"`
	SeriesID               string   `json:"series_id"`
	ReferenceURL           string   `json:"reference_url"`
	ResponseSchemaJSON     string   `json:"response_schema_json"`
	PayloadKind            string   `json:"payload_kind"`
	PayloadJSON            string   `json:"payload_json"`
	EscrowAmount           int64    `json:"escrow_amount"`
	FundedOrganizationID   string   `json:"funded_organization_id"`
}

type StoredNotification struct {
	ID              string `json:"id"`
	RecipientUserID string `json:"recipient_user_id"`
	ActorUserID     string `json:"actor_user_id"`
	Kind            string `json:"kind"`
	SubjectKind     string `json:"subject_kind"`
	SubjectID       string `json:"subject_id"`
	State           string `json:"state"`
	MetadataJSON    string `json:"metadata_json"`
	CreatedAt       string `json:"created_at"`
}

type StoredOrganization struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
}

type StoredOrganizationMember struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	UserID         string   `json:"user_id"`
	Status         string   `json:"status"`
	Roles          []string `json:"roles"`
}

type StoredTeam struct {
	ID             string `json:"id"`
	OwnerKind      string `json:"owner_kind"`
	OrganizationID string `json:"organization_id"`
	OwnerUserID    string `json:"owner_user_id"`
	Name           string `json:"name"`
	CreatedBy      string `json:"created_by"`
}

type StoredTaskSeries struct {
	ID          string `json:"id"`
	OwnerKind   string `json:"owner_kind"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	CreatedBy   string `json:"created_by"`
}

type StoredUser struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

type StoredAuditEvent struct {
	ID           string `json:"id"`
	ActorID      string `json:"actor_id"`
	Action       string `json:"action"`
	SubjectKind  string `json:"subject_kind"`
	SubjectID    string `json:"subject_id"`
	MetadataJSON string `json:"metadata_json"`
	CreatedAt    string `json:"created_at"`
}

type StoredPlatformAdmin struct {
	UserID    string `json:"user_id"`
	Source    string `json:"source"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
}

type StoredAccountToken struct {
	Token  string `json:"token"`
	Kind   string `json:"kind"`
	UserID string `json:"user_id"`
	State  string `json:"state"`
}

type StoredCollectible struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	State          string `json:"state"`
	TransferPolicy string `json:"transfer_policy"`
	OwnerID        string `json:"owner_id"`
	OwnerKind      string `json:"owner_kind"`
	OrganizationID string `json:"organization_id"`
	Art            string `json:"art"`
}

type StoredAgentCredential struct {
	ID        string   `json:"id"`
	OwnerID   string   `json:"owner_id"`
	Label     string   `json:"label"`
	Scopes    []string `json:"scopes"`
	State     string   `json:"state"`
	ExpiresAt string   `json:"expires_at"`
	TaskID    string   `json:"task_id"`
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

type TaskStorageResult interface {
	taskStorageResult()
}

type TaskStored struct {
	Value StoredTask
}

type TaskStorageRejected struct {
	Reason string
}

func (TaskStored) taskStorageResult()          {}
func (TaskStorageRejected) taskStorageResult() {}

func SaveTask(storage BrowserStorage, task StoredTask) TaskStorageResult {
	cleaned := cleanStoredTask(task)
	if reason := validateStoredTask(cleaned); reason != "" {
		return TaskStorageRejected{Reason: reason}
	}
	keyResult := NewStorageKey("task:" + cleaned.ID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return TaskStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return TaskStorageRejected{Reason: "task encoding failed"}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return TaskStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	indexResult := appendStringIndex(storage, "task:index", cleaned.ID, "task")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return TaskStorageRejected{Reason: indexResult.(stringIndexRejected).reason}
	}
	return TaskStored{Value: cleaned}
}

func LoadTask(storage BrowserStorage, taskID string) TaskStorageResult {
	cleanID := strings.TrimSpace(taskID)
	if cleanID == "" {
		return TaskStorageRejected{Reason: "task id is required"}
	}
	keyResult := NewStorageKey("task:" + cleanID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return TaskStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return TaskStorageRejected{Reason: taskReadReason(readResult)}
	}
	var task StoredTask
	if err := json.Unmarshal([]byte(read.Value), &task); err != nil {
		return TaskStorageRejected{Reason: "task decoding failed"}
	}
	cleaned := cleanStoredTask(task)
	if cleaned.ID != cleanID {
		return TaskStorageRejected{Reason: "task storage key contains mismatched record"}
	}
	if reason := validateStoredTask(cleaned); reason != "" {
		return TaskStorageRejected{Reason: reason}
	}
	return TaskStored{Value: cleaned}
}

type TasksStored struct {
	Values []StoredTask
}

func (TasksStored) taskStorageResult() {}

func ListTasks(storage BrowserStorage, query string, scope string, userID string, organizationID string, states []string, page StoredListPage) TaskStorageResult {
	idsResult := loadStringIndex(storage, "task:index", "task")
	ids, idsMatched := idsResult.(stringIndexLoaded)
	if !idsMatched {
		return TaskStorageRejected{Reason: idsResult.(stringIndexRejected).reason}
	}
	cleanQuery := strings.ToLower(strings.TrimSpace(query))
	cleanScope := strings.TrimSpace(scope)
	cleanUserID := strings.TrimSpace(userID)
	cleanOrganizationID := strings.TrimSpace(organizationID)
	cleanStates := make(map[string]bool, len(states))
	for _, state := range states {
		trimmed := strings.TrimSpace(state)
		if trimmed != "" {
			cleanStates[trimmed] = true
		}
	}
	values := make([]StoredTask, 0, len(ids.values))
	for index := range ids.values {
		loadResult := LoadTask(storage, ids.values[index])
		loaded, loadedMatched := loadResult.(TaskStored)
		if !loadedMatched {
			return loadResult
		}
		task := loaded.Value
		if cleanScope == "public" && (task.VisibilityKind != "public" || task.State != "open") {
			continue
		}
		if cleanScope == "user" && cleanUserID != "" && task.CreatedBy != cleanUserID {
			continue
		}
		if cleanScope == "organization" {
			if task.VisibilityKind != "organization" && task.OwnerKind != "organization" && task.OwnerKind != "organization_team" {
				continue
			}
			if cleanOrganizationID != "" && task.OwnerID != cleanOrganizationID && task.VisibilityID != cleanOrganizationID {
				continue
			}
		}
		if len(cleanStates) > 0 && !cleanStates[task.State] {
			continue
		}
		if cleanQuery != "" && !strings.Contains(strings.ToLower(task.Title), cleanQuery) && !strings.Contains(strings.ToLower(task.Description), cleanQuery) && !strings.Contains(strings.ToLower(task.ID), cleanQuery) {
			continue
		}
		values = append(values, task)
	}
	start, end := pageBounds(len(values), page)
	return TasksStored{Values: values[start:end]}
}

func cleanStoredTask(task StoredTask) StoredTask {
	return StoredTask{
		ID:                     strings.TrimSpace(task.ID),
		OwnerKind:              strings.TrimSpace(task.OwnerKind),
		OwnerID:                strings.TrimSpace(task.OwnerID),
		Title:                  strings.TrimSpace(task.Title),
		Description:            strings.TrimSpace(task.Description),
		TaskType:               strings.TrimSpace(task.TaskType),
		ReferenceURL:           strings.TrimSpace(task.ReferenceURL),
		RewardKind:             strings.TrimSpace(task.RewardKind),
		RewardCollectibleIDs:   cleanStorageStringSlice(task.RewardCollectibleIDs),
		RewardCreditAmount:     task.RewardCreditAmount,
		RewardCollectibleCount: task.RewardCollectibleCount,
		ParticipationPolicy:    strings.TrimSpace(task.ParticipationPolicy),
		AssigneeScope:          strings.TrimSpace(task.AssigneeScope),
		ReservationExpiryHours: task.ReservationExpiryHours,
		State:                  strings.TrimSpace(task.State),
		VisibilityKind:         strings.TrimSpace(task.VisibilityKind),
		VisibilityID:           strings.TrimSpace(task.VisibilityID),
		SeriesKind:             strings.TrimSpace(task.SeriesKind),
		SeriesID:               strings.TrimSpace(task.SeriesID),
		SeriesPosition:         task.SeriesPosition,
		ResponseSchemaJSON:     strings.TrimSpace(task.ResponseSchemaJSON),
		PayloadKind:            strings.TrimSpace(task.PayloadKind),
		PayloadJSON:            strings.TrimSpace(task.PayloadJSON),
		CreatedBy:              strings.TrimSpace(task.CreatedBy),
		EscrowAmount:           task.EscrowAmount,
		FundedOrganizationID:   strings.TrimSpace(task.FundedOrganizationID),
	}
}

func cleanStorageStringSlice(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for index := range values {
		value := strings.TrimSpace(values[index])
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func validateStoredTask(task StoredTask) string {
	if task.ID == "" {
		return "task id is required"
	}
	if !validStoredTaskOwnerKind(task.OwnerKind) {
		return "task owner kind is invalid"
	}
	if task.OwnerID == "" {
		return "task owner id is required"
	}
	if task.Title == "" {
		return "task title is required"
	}
	if task.Description == "" {
		return "task description is required"
	}
	if task.TaskType == "" {
		return "task type is required"
	}
	if !validStoredTaskRewardKind(task.RewardKind) {
		return "task reward kind is invalid"
	}
	if !validStoredTaskParticipation(task.ParticipationPolicy) {
		return "task participation policy is invalid"
	}
	if !validStoredTaskAssigneeScope(task.AssigneeScope) {
		return "task assignee scope is invalid"
	}
	if task.ReservationExpiryHours < 1 {
		return "task reservation expiry is invalid"
	}
	if !validStoredTaskState(task.State) {
		return "task state is invalid"
	}
	if !validStoredTaskVisibilityKind(task.VisibilityKind) {
		return "task visibility kind is invalid"
	}
	if task.SeriesKind == "" {
		return "task series kind is required"
	}
	if task.ResponseSchemaJSON == "" {
		return "task response schema is required"
	}
	if !validStoredTaskPayloadKind(task.PayloadKind) {
		return "task payload kind is invalid"
	}
	if task.CreatedBy == "" {
		return "task creator is required"
	}
	return ""
}

func taskReadReason(result StorageReadResult) string {
	switch rejected := result.(type) {
	case StorageMissing:
		return rejected.Reason
	case StorageReadRejected:
		return rejected.Reason
	default:
		return "task read failed"
	}
}

func validStoredTaskOwnerKind(value string) bool {
	switch value {
	case "user", "team", "organization", "organization_team":
		return true
	default:
		return false
	}
}

func validStoredTaskRewardKind(value string) bool {
	switch value {
	case "none", "credit", "collectible", "bundle":
		return true
	default:
		return false
	}
}

func validStoredTaskParticipation(value string) bool {
	switch value {
	case "open", "reservation_required", "approval_required":
		return true
	default:
		return false
	}
}

func validStoredTaskAssigneeScope(value string) bool {
	switch value {
	case "user", "team", "organization_team":
		return true
	default:
		return false
	}
}

func validStoredTaskState(value string) bool {
	switch value {
	case "draft", "open", "cancelled", "refunded", "closed":
		return true
	default:
		return false
	}
}

func validStoredTaskVisibilityKind(value string) bool {
	switch value {
	case "public", "user", "team", "organization", "organization_team", "default":
		return true
	default:
		return false
	}
}

func validStoredTaskPayloadKind(value string) bool {
	switch value {
	case "none", "json":
		return true
	default:
		return false
	}
}

type NotificationPage struct {
	limit  int
	offset int
}

type NotificationPageResult interface {
	notificationPageResult()
}

type NotificationPageAccepted struct {
	Value NotificationPage
}

type NotificationPageRejected struct {
	Reason string
}

func (NotificationPageAccepted) notificationPageResult() {}
func (NotificationPageRejected) notificationPageResult() {}

func NewNotificationPage(limit int, offset int) NotificationPageResult {
	if limit < 1 {
		return NotificationPageRejected{Reason: "notification page limit is invalid"}
	}
	if offset < 0 {
		return NotificationPageRejected{Reason: "notification page offset is invalid"}
	}
	return NotificationPageAccepted{Value: NotificationPage{limit: limit, offset: offset}}
}

func DefaultNotificationPage() NotificationPage {
	return NotificationPage{limit: 20, offset: 0}
}

type NotificationStorageResult interface {
	notificationStorageResult()
}

type NotificationStored struct {
	Value StoredNotification
}

type NotificationsStored struct {
	Values []StoredNotification
}

type NotificationStorageRejected struct {
	Reason string
}

func (NotificationStored) notificationStorageResult()          {}
func (NotificationsStored) notificationStorageResult()         {}
func (NotificationStorageRejected) notificationStorageResult() {}

func SaveNotification(storage BrowserStorage, notification StoredNotification) NotificationStorageResult {
	cleaned := cleanStoredNotification(notification)
	if reason := validateStoredNotification(cleaned); reason != "" {
		return NotificationStorageRejected{Reason: reason}
	}
	keyResult := notificationKey(cleaned.ID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return NotificationStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return NotificationStorageRejected{Reason: "notification encoding failed"}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return NotificationStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	indexResult := appendNotificationIndex(storage, cleaned.RecipientUserID, cleaned.ID)
	if _, matched := indexResult.(NotificationsStored); !matched {
		return indexResult
	}
	return NotificationStored{Value: cleaned}
}

func ListNotifications(storage BrowserStorage, recipientUserID string, page NotificationPage) NotificationStorageResult {
	cleanRecipientID := strings.TrimSpace(recipientUserID)
	if cleanRecipientID == "" {
		return NotificationStorageRejected{Reason: "notification recipient is required"}
	}
	idsResult := loadNotificationIndex(storage, cleanRecipientID)
	ids, idsMatched := idsResult.(notificationIDsLoaded)
	if !idsMatched {
		return NotificationStorageRejected{Reason: idsResult.(notificationIDsRejected).Reason}
	}
	start := page.offset
	if start > len(ids.Values) {
		start = len(ids.Values)
	}
	end := start + page.limit
	if end > len(ids.Values) {
		end = len(ids.Values)
	}
	values := make([]StoredNotification, 0, end-start)
	for index := start; index < end; index++ {
		loadResult := LoadNotification(storage, ids.Values[index])
		loaded, loadedMatched := loadResult.(NotificationStored)
		if !loadedMatched {
			return loadResult
		}
		if loaded.Value.RecipientUserID != cleanRecipientID {
			return NotificationStorageRejected{Reason: "notification index contains mismatched record"}
		}
		values = append(values, loaded.Value)
	}
	return NotificationsStored{Values: values}
}

func MarkNotificationRead(storage BrowserStorage, notificationID string, recipientUserID string) NotificationStorageResult {
	loadResult := LoadNotification(storage, notificationID)
	loaded, loadedMatched := loadResult.(NotificationStored)
	if !loadedMatched {
		return loadResult
	}
	cleanRecipientID := strings.TrimSpace(recipientUserID)
	if cleanRecipientID == "" {
		return NotificationStorageRejected{Reason: "notification recipient is required"}
	}
	if loaded.Value.RecipientUserID != cleanRecipientID {
		return NotificationStorageRejected{Reason: "notification does not belong to actor"}
	}
	loaded.Value.State = "read"
	return SaveNotification(storage, loaded.Value)
}

func LoadNotification(storage BrowserStorage, notificationID string) NotificationStorageResult {
	cleanID := strings.TrimSpace(notificationID)
	if cleanID == "" {
		return NotificationStorageRejected{Reason: "notification id is required"}
	}
	keyResult := notificationKey(cleanID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return NotificationStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return NotificationStorageRejected{Reason: notificationReadReason(readResult)}
	}
	var notification StoredNotification
	if err := json.Unmarshal([]byte(read.Value), &notification); err != nil {
		return NotificationStorageRejected{Reason: "notification decoding failed"}
	}
	cleaned := cleanStoredNotification(notification)
	if cleaned.ID != cleanID {
		return NotificationStorageRejected{Reason: "notification storage key contains mismatched record"}
	}
	if reason := validateStoredNotification(cleaned); reason != "" {
		return NotificationStorageRejected{Reason: reason}
	}
	return NotificationStored{Value: cleaned}
}

func cleanStoredNotification(notification StoredNotification) StoredNotification {
	return StoredNotification{
		ID:              strings.TrimSpace(notification.ID),
		RecipientUserID: strings.TrimSpace(notification.RecipientUserID),
		ActorUserID:     strings.TrimSpace(notification.ActorUserID),
		Kind:            strings.TrimSpace(notification.Kind),
		SubjectKind:     strings.TrimSpace(notification.SubjectKind),
		SubjectID:       strings.TrimSpace(notification.SubjectID),
		State:           strings.TrimSpace(notification.State),
		MetadataJSON:    strings.TrimSpace(notification.MetadataJSON),
		CreatedAt:       strings.TrimSpace(notification.CreatedAt),
	}
}

func validateStoredNotification(notification StoredNotification) string {
	if notification.ID == "" {
		return "notification id is required"
	}
	if notification.RecipientUserID == "" {
		return "notification recipient is required"
	}
	if notification.ActorUserID == "" {
		return "notification actor is required"
	}
	if notification.Kind == "" {
		return "notification kind is required"
	}
	if notification.SubjectKind == "" {
		return "notification subject kind is required"
	}
	if notification.SubjectID == "" {
		return "notification subject id is required"
	}
	if !validStoredNotificationState(notification.State) {
		return "notification state is invalid"
	}
	if notification.MetadataJSON == "" {
		return "notification metadata is required"
	}
	if notification.CreatedAt == "" {
		return "notification created time is required"
	}
	return ""
}

func validStoredNotificationState(value string) bool {
	switch value {
	case "unread", "read":
		return true
	default:
		return false
	}
}

type notificationIDsResult interface {
	notificationIDsResult()
}

type notificationIDsLoaded struct {
	Values []string
}

type notificationIDsRejected struct {
	Reason string
}

func (notificationIDsLoaded) notificationIDsResult()   {}
func (notificationIDsRejected) notificationIDsResult() {}

func notificationKey(id string) StorageKeyResult {
	return NewStorageKey("notification:" + strings.TrimSpace(id))
}

func notificationIndexKey(recipientUserID string) StorageKeyResult {
	return NewStorageKey("notification:index:" + strings.TrimSpace(recipientUserID))
}

func loadNotificationIndex(storage BrowserStorage, recipientUserID string) notificationIDsResult {
	keyResult := notificationIndexKey(recipientUserID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return notificationIDsRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	if _, missing := readResult.(StorageMissing); missing {
		return notificationIDsLoaded{Values: []string{}}
	}
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return notificationIDsRejected{Reason: notificationReadReason(readResult)}
	}
	var ids []string
	if err := json.Unmarshal([]byte(read.Value), &ids); err != nil {
		return notificationIDsRejected{Reason: "notification index decoding failed"}
	}
	for index := range ids {
		if strings.TrimSpace(ids[index]) == "" {
			return notificationIDsRejected{Reason: "notification index contains an invalid id"}
		}
	}
	return notificationIDsLoaded{Values: ids}
}

func appendNotificationIndex(storage BrowserStorage, recipientUserID string, id string) NotificationStorageResult {
	idsResult := loadNotificationIndex(storage, recipientUserID)
	ids, idsMatched := idsResult.(notificationIDsLoaded)
	if !idsMatched {
		return NotificationStorageRejected{Reason: idsResult.(notificationIDsRejected).Reason}
	}
	for index := range ids.Values {
		if ids.Values[index] == id {
			return NotificationsStored{Values: []StoredNotification{}}
		}
	}
	ids.Values = append([]string{id}, ids.Values...)
	encoded, err := json.Marshal(ids.Values)
	if err != nil {
		return NotificationStorageRejected{Reason: "notification index encoding failed"}
	}
	keyResult := notificationIndexKey(recipientUserID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return NotificationStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return NotificationStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	return NotificationsStored{Values: []StoredNotification{}}
}

func notificationReadReason(result StorageReadResult) string {
	switch rejected := result.(type) {
	case StorageMissing:
		return rejected.Reason
	case StorageReadRejected:
		return rejected.Reason
	default:
		return "notification read failed"
	}
}

type StoredListPage struct {
	limit  int
	offset int
}

type StoredListPageResult interface {
	storedListPageResult()
}

type StoredListPageAccepted struct {
	Value StoredListPage
}

type StoredListPageRejected struct {
	Reason string
}

func (StoredListPageAccepted) storedListPageResult() {}
func (StoredListPageRejected) storedListPageResult() {}

func NewStoredListPage(limit int, offset int) StoredListPageResult {
	if limit < 1 {
		return StoredListPageRejected{Reason: "list page limit is invalid"}
	}
	if offset < 0 {
		return StoredListPageRejected{Reason: "list page offset is invalid"}
	}
	return StoredListPageAccepted{Value: StoredListPage{limit: limit, offset: offset}}
}

func DefaultStoredListPage() StoredListPage {
	return StoredListPage{limit: 20, offset: 0}
}

type OrganizationStorageResult interface {
	organizationStorageResult()
}

type OrganizationStored struct {
	Value StoredOrganization
}

type OrganizationsStored struct {
	Values []StoredOrganization
}

type OrganizationStorageRejected struct {
	Reason string
}

func (OrganizationStored) organizationStorageResult()          {}
func (OrganizationsStored) organizationStorageResult()         {}
func (OrganizationStorageRejected) organizationStorageResult() {}

func SaveOrganization(storage BrowserStorage, organization StoredOrganization) OrganizationStorageResult {
	cleaned := cleanStoredOrganization(organization)
	if reason := validateStoredOrganization(cleaned); reason != "" {
		return OrganizationStorageRejected{Reason: reason}
	}
	keyResult := NewStorageKey("organization:" + cleaned.ID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return OrganizationStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return OrganizationStorageRejected{Reason: "organization encoding failed"}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return OrganizationStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	indexResult := appendStringIndex(storage, "organization:index", cleaned.ID, "organization")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return OrganizationStorageRejected{Reason: indexResult.(stringIndexRejected).reason}
	}
	return OrganizationStored{Value: cleaned}
}

func ListOrganizations(storage BrowserStorage, query string, page StoredListPage) OrganizationStorageResult {
	idsResult := loadStringIndex(storage, "organization:index", "organization")
	ids, idsMatched := idsResult.(stringIndexLoaded)
	if !idsMatched {
		return OrganizationStorageRejected{Reason: idsResult.(stringIndexRejected).reason}
	}
	cleanQuery := strings.ToLower(strings.TrimSpace(query))
	values := make([]StoredOrganization, 0, len(ids.values))
	for index := range ids.values {
		loadResult := LoadOrganization(storage, ids.values[index])
		loaded, loadedMatched := loadResult.(OrganizationStored)
		if !loadedMatched {
			return loadResult
		}
		if cleanQuery == "" || strings.Contains(strings.ToLower(loaded.Value.Name), cleanQuery) || strings.Contains(strings.ToLower(loaded.Value.ID), cleanQuery) {
			values = append(values, loaded.Value)
		}
	}
	return OrganizationsStored{Values: pageStoredOrganizations(values, page)}
}

func LoadOrganization(storage BrowserStorage, organizationID string) OrganizationStorageResult {
	cleanID := strings.TrimSpace(organizationID)
	if cleanID == "" {
		return OrganizationStorageRejected{Reason: "organization id is required"}
	}
	keyResult := NewStorageKey("organization:" + cleanID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return OrganizationStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return OrganizationStorageRejected{Reason: storageReadReason(readResult, "organization")}
	}
	var organization StoredOrganization
	if err := json.Unmarshal([]byte(read.Value), &organization); err != nil {
		return OrganizationStorageRejected{Reason: "organization decoding failed"}
	}
	cleaned := cleanStoredOrganization(organization)
	if cleaned.ID != cleanID {
		return OrganizationStorageRejected{Reason: "organization storage key contains mismatched record"}
	}
	if reason := validateStoredOrganization(cleaned); reason != "" {
		return OrganizationStorageRejected{Reason: reason}
	}
	return OrganizationStored{Value: cleaned}
}

type TaskSeriesStorageResult interface {
	taskSeriesStorageResult()
}

type TaskSeriesStored struct {
	Value StoredTaskSeries
}

type TaskSeriesListStored struct {
	Values []StoredTaskSeries
}

type TaskSeriesStorageRejected struct {
	Reason string
}

func (TaskSeriesStored) taskSeriesStorageResult()     {}
func (TaskSeriesListStored) taskSeriesStorageResult() {}

func (TaskSeriesStorageRejected) taskSeriesStorageResult() {}

func SaveTaskSeries(storage BrowserStorage, series StoredTaskSeries) TaskSeriesStorageResult {
	cleaned := cleanStoredTaskSeries(series)
	if reason := validateStoredTaskSeries(cleaned); reason != "" {
		return TaskSeriesStorageRejected{Reason: reason}
	}
	keyResult := NewStorageKey("task_series:" + cleaned.ID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return TaskSeriesStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return TaskSeriesStorageRejected{Reason: "task series encoding failed"}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return TaskSeriesStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	indexResult := appendStringIndex(storage, "task_series:index", cleaned.ID, "task series")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return TaskSeriesStorageRejected{Reason: indexResult.(stringIndexRejected).reason}
	}
	return TaskSeriesStored{Value: cleaned}
}

func ListTaskSeries(storage BrowserStorage, page StoredListPage) TaskSeriesStorageResult {
	idsResult := loadStringIndex(storage, "task_series:index", "task series")
	ids, idsMatched := idsResult.(stringIndexLoaded)
	if !idsMatched {
		return TaskSeriesStorageRejected{Reason: idsResult.(stringIndexRejected).reason}
	}
	values := make([]StoredTaskSeries, 0, len(ids.values))
	for index := range ids.values {
		loadResult := LoadTaskSeries(storage, ids.values[index])
		loaded, loadedMatched := loadResult.(TaskSeriesStored)
		if !loadedMatched {
			return loadResult
		}
		values = append(values, loaded.Value)
	}
	start, end := pageBounds(len(values), page)
	return TaskSeriesListStored{Values: values[start:end]}
}

func LoadTaskSeries(storage BrowserStorage, seriesID string) TaskSeriesStorageResult {
	cleanID := strings.TrimSpace(seriesID)
	if cleanID == "" {
		return TaskSeriesStorageRejected{Reason: "task series id is required"}
	}
	keyResult := NewStorageKey("task_series:" + cleanID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return TaskSeriesStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return TaskSeriesStorageRejected{Reason: storageReadReason(readResult, "task series")}
	}
	var series StoredTaskSeries
	if err := json.Unmarshal([]byte(read.Value), &series); err != nil {
		return TaskSeriesStorageRejected{Reason: "task series decoding failed"}
	}
	cleaned := cleanStoredTaskSeries(series)
	if cleaned.ID != cleanID {
		return TaskSeriesStorageRejected{Reason: "task series storage key contains mismatched record"}
	}
	if reason := validateStoredTaskSeries(cleaned); reason != "" {
		return TaskSeriesStorageRejected{Reason: reason}
	}
	return TaskSeriesStored{Value: cleaned}
}

func cleanStoredTaskSeries(series StoredTaskSeries) StoredTaskSeries {
	return StoredTaskSeries{
		ID:          strings.TrimSpace(series.ID),
		OwnerKind:   strings.TrimSpace(series.OwnerKind),
		Title:       strings.TrimSpace(series.Title),
		Description: strings.TrimSpace(series.Description),
		State:       strings.TrimSpace(series.State),
		CreatedBy:   strings.TrimSpace(series.CreatedBy),
	}
}

func validateStoredTaskSeries(series StoredTaskSeries) string {
	if series.ID == "" {
		return "task series id is required"
	}
	if series.Title == "" {
		return "task series title is required"
	}
	if series.CreatedBy == "" {
		return "task series creator is required"
	}
	return ""
}

type OrganizationMemberStorageResult interface {
	organizationMemberStorageResult()
}

type OrganizationMemberStored struct {
	Value StoredOrganizationMember
}

type OrganizationMembersStored struct {
	Values []StoredOrganizationMember
}

type OrganizationMemberStorageRejected struct {
	Reason string
}

func (OrganizationMemberStored) organizationMemberStorageResult()          {}
func (OrganizationMembersStored) organizationMemberStorageResult()         {}
func (OrganizationMemberStorageRejected) organizationMemberStorageResult() {}

func SaveOrganizationMember(storage BrowserStorage, member StoredOrganizationMember) OrganizationMemberStorageResult {
	cleaned := cleanStoredOrganizationMember(member)
	if reason := validateStoredOrganizationMember(cleaned); reason != "" {
		return OrganizationMemberStorageRejected{Reason: reason}
	}
	keyResult := organizationMemberKey(cleaned.OrganizationID, cleaned.UserID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return OrganizationMemberStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return OrganizationMemberStorageRejected{Reason: "organization member encoding failed"}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return OrganizationMemberStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	indexResult := appendStringIndex(storage, organizationMemberIndexKey(cleaned.OrganizationID), cleaned.UserID, "organization member")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return OrganizationMemberStorageRejected{Reason: indexResult.(stringIndexRejected).reason}
	}
	return OrganizationMemberStored{Value: cleaned}
}

func ListOrganizationMembers(storage BrowserStorage, organizationID string, page StoredListPage) OrganizationMemberStorageResult {
	cleanOrganizationID := strings.TrimSpace(organizationID)
	if cleanOrganizationID == "" {
		return OrganizationMemberStorageRejected{Reason: "organization id is required"}
	}
	idsResult := loadStringIndex(storage, organizationMemberIndexKey(cleanOrganizationID), "organization member")
	ids, idsMatched := idsResult.(stringIndexLoaded)
	if !idsMatched {
		return OrganizationMemberStorageRejected{Reason: idsResult.(stringIndexRejected).reason}
	}
	values := make([]StoredOrganizationMember, 0, len(ids.values))
	for index := range ids.values {
		loadResult := LoadOrganizationMember(storage, cleanOrganizationID, ids.values[index])
		loaded, loadedMatched := loadResult.(OrganizationMemberStored)
		if !loadedMatched {
			return loadResult
		}
		values = append(values, loaded.Value)
	}
	return OrganizationMembersStored{Values: pageStoredOrganizationMembers(values, page)}
}

func LoadOrganizationMember(storage BrowserStorage, organizationID string, userID string) OrganizationMemberStorageResult {
	cleanOrganizationID := strings.TrimSpace(organizationID)
	cleanUserID := strings.TrimSpace(userID)
	if cleanOrganizationID == "" {
		return OrganizationMemberStorageRejected{Reason: "organization id is required"}
	}
	if cleanUserID == "" {
		return OrganizationMemberStorageRejected{Reason: "organization member user id is required"}
	}
	keyResult := organizationMemberKey(cleanOrganizationID, cleanUserID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return OrganizationMemberStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return OrganizationMemberStorageRejected{Reason: storageReadReason(readResult, "organization member")}
	}
	var member StoredOrganizationMember
	if err := json.Unmarshal([]byte(read.Value), &member); err != nil {
		return OrganizationMemberStorageRejected{Reason: "organization member decoding failed"}
	}
	cleaned := cleanStoredOrganizationMember(member)
	if cleaned.OrganizationID != cleanOrganizationID || cleaned.UserID != cleanUserID {
		return OrganizationMemberStorageRejected{Reason: "organization member storage key contains mismatched record"}
	}
	if reason := validateStoredOrganizationMember(cleaned); reason != "" {
		return OrganizationMemberStorageRejected{Reason: reason}
	}
	return OrganizationMemberStored{Value: cleaned}
}

func UpdateOrganizationMemberRoles(storage BrowserStorage, organizationID string, userID string, roles []string) OrganizationMemberStorageResult {
	loadResult := LoadOrganizationMember(storage, organizationID, userID)
	loaded, loadedMatched := loadResult.(OrganizationMemberStored)
	if !loadedMatched {
		return loadResult
	}
	loaded.Value.Roles = cleanStoredOrganizationRoles(roles)
	return SaveOrganizationMember(storage, loaded.Value)
}

func DeactivateOrganizationMember(storage BrowserStorage, organizationID string, userID string) OrganizationMemberStorageResult {
	loadResult := LoadOrganizationMember(storage, organizationID, userID)
	loaded, loadedMatched := loadResult.(OrganizationMemberStored)
	if !loadedMatched {
		return loadResult
	}
	loaded.Value.Status = "deactivated"
	return SaveOrganizationMember(storage, loaded.Value)
}

type TeamStorageResult interface {
	teamStorageResult()
}

type TeamStored struct {
	Value StoredTeam
}

type TeamsStored struct {
	Values []StoredTeam
}

type TeamStorageRejected struct {
	Reason string
}

func (TeamStored) teamStorageResult()          {}
func (TeamsStored) teamStorageResult()         {}
func (TeamStorageRejected) teamStorageResult() {}

func SaveTeam(storage BrowserStorage, team StoredTeam) TeamStorageResult {
	cleaned := cleanStoredTeam(team)
	if reason := validateStoredTeam(cleaned); reason != "" {
		return TeamStorageRejected{Reason: reason}
	}
	keyResult := NewStorageKey("team:" + cleaned.ID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return TeamStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return TeamStorageRejected{Reason: "team encoding failed"}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return TeamStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	var indexKey string
	if cleaned.OwnerKind == "organization" {
		indexKey = organizationTeamIndexKey(cleaned.OrganizationID)
	} else {
		indexKey = standaloneTeamIndexKey(cleaned.OwnerUserID)
	}
	indexResult := appendStringIndex(storage, indexKey, cleaned.ID, "team")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return TeamStorageRejected{Reason: indexResult.(stringIndexRejected).reason}
	}
	return TeamStored{Value: cleaned}
}

func LoadTeam(storage BrowserStorage, teamID string) TeamStorageResult {
	cleanID := strings.TrimSpace(teamID)
	if cleanID == "" {
		return TeamStorageRejected{Reason: "team id is required"}
	}
	keyResult := NewStorageKey("team:" + cleanID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return TeamStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return TeamStorageRejected{Reason: storageReadReason(readResult, "team")}
	}
	var team StoredTeam
	if err := json.Unmarshal([]byte(read.Value), &team); err != nil {
		return TeamStorageRejected{Reason: "team decoding failed"}
	}
	cleaned := cleanStoredTeam(team)
	if cleaned.ID != cleanID {
		return TeamStorageRejected{Reason: "team storage key contains mismatched record"}
	}
	if reason := validateStoredTeam(cleaned); reason != "" {
		return TeamStorageRejected{Reason: reason}
	}
	return TeamStored{Value: cleaned}
}

func ListOrganizationTeams(storage BrowserStorage, organizationID string, query string, page StoredListPage) TeamStorageResult {
	return listTeamsFromIndex(storage, organizationTeamIndexKey(strings.TrimSpace(organizationID)), strings.TrimSpace(organizationID), "", query, page)
}

func ListStandaloneTeams(storage BrowserStorage, ownerUserID string, query string, page StoredListPage) TeamStorageResult {
	return listTeamsFromIndex(storage, standaloneTeamIndexKey(strings.TrimSpace(ownerUserID)), "", strings.TrimSpace(ownerUserID), query, page)
}

func listTeamsFromIndex(storage BrowserStorage, indexKey string, organizationID string, ownerUserID string, query string, page StoredListPage) TeamStorageResult {
	if strings.TrimSpace(indexKey) == "" {
		return TeamStorageRejected{Reason: "team index key is required"}
	}
	if organizationID == "" && ownerUserID == "" {
		return TeamStorageRejected{Reason: "team owner is required"}
	}
	idsResult := loadStringIndex(storage, indexKey, "team")
	ids, idsMatched := idsResult.(stringIndexLoaded)
	if !idsMatched {
		return TeamStorageRejected{Reason: idsResult.(stringIndexRejected).reason}
	}
	cleanQuery := strings.ToLower(strings.TrimSpace(query))
	values := make([]StoredTeam, 0, len(ids.values))
	for index := range ids.values {
		loadResult := LoadTeam(storage, ids.values[index])
		loaded, loadedMatched := loadResult.(TeamStored)
		if !loadedMatched {
			return loadResult
		}
		if organizationID != "" && loaded.Value.OrganizationID != organizationID {
			return TeamStorageRejected{Reason: "team index contains mismatched organization record"}
		}
		if ownerUserID != "" && loaded.Value.OwnerUserID != ownerUserID {
			return TeamStorageRejected{Reason: "team index contains mismatched user record"}
		}
		if cleanQuery == "" || strings.Contains(strings.ToLower(loaded.Value.Name), cleanQuery) || strings.Contains(strings.ToLower(loaded.Value.ID), cleanQuery) {
			values = append(values, loaded.Value)
		}
	}
	return TeamsStored{Values: pageStoredTeams(values, page)}
}

func cleanStoredOrganization(organization StoredOrganization) StoredOrganization {
	return StoredOrganization{
		ID:        strings.TrimSpace(organization.ID),
		Name:      strings.TrimSpace(organization.Name),
		CreatedBy: strings.TrimSpace(organization.CreatedBy),
	}
}

func validateStoredOrganization(organization StoredOrganization) string {
	if organization.ID == "" {
		return "organization id is required"
	}
	if organization.Name == "" {
		return "organization name is required"
	}
	if organization.CreatedBy == "" {
		return "organization creator is required"
	}
	return ""
}

func cleanStoredOrganizationMember(member StoredOrganizationMember) StoredOrganizationMember {
	return StoredOrganizationMember{
		ID:             strings.TrimSpace(member.ID),
		OrganizationID: strings.TrimSpace(member.OrganizationID),
		UserID:         strings.TrimSpace(member.UserID),
		Status:         strings.TrimSpace(member.Status),
		Roles:          cleanStoredOrganizationRoles(member.Roles),
	}
}

func cleanStoredOrganizationRoles(roles []string) []string {
	cleaned := make([]string, 0, len(roles))
	for index := range roles {
		role := strings.TrimSpace(roles[index])
		if role != "" {
			cleaned = append(cleaned, role)
		}
	}
	return cleaned
}

func validateStoredOrganizationMember(member StoredOrganizationMember) string {
	if member.ID == "" {
		return "organization member id is required"
	}
	if member.OrganizationID == "" {
		return "organization id is required"
	}
	if member.UserID == "" {
		return "organization member user id is required"
	}
	if !validStoredOrganizationMemberStatus(member.Status) {
		return "organization member status is invalid"
	}
	if len(member.Roles) == 0 {
		return "organization member role is required"
	}
	for index := range member.Roles {
		if !validStoredOrganizationRole(member.Roles[index]) {
			return "organization member role is invalid"
		}
	}
	return ""
}

func validStoredOrganizationMemberStatus(value string) bool {
	switch value {
	case "active", "deactivated":
		return true
	default:
		return false
	}
}

func validStoredOrganizationRole(value string) bool {
	switch value {
	case "owner", "admin", "member", "billing", "reviewer", "public_publisher":
		return true
	default:
		return false
	}
}

func cleanStoredTeam(team StoredTeam) StoredTeam {
	return StoredTeam{
		ID:             strings.TrimSpace(team.ID),
		OwnerKind:      strings.TrimSpace(team.OwnerKind),
		OrganizationID: strings.TrimSpace(team.OrganizationID),
		OwnerUserID:    strings.TrimSpace(team.OwnerUserID),
		Name:           strings.TrimSpace(team.Name),
		CreatedBy:      strings.TrimSpace(team.CreatedBy),
	}
}

func validateStoredTeam(team StoredTeam) string {
	if team.ID == "" {
		return "team id is required"
	}
	if !validStoredTeamOwnerKind(team.OwnerKind) {
		return "team owner kind is invalid"
	}
	if team.OwnerKind == "organization" && team.OrganizationID == "" {
		return "team organization id is required"
	}
	if team.OwnerKind == "user" && team.OwnerUserID == "" {
		return "team owner user id is required"
	}
	if team.Name == "" {
		return "team name is required"
	}
	if team.CreatedBy == "" {
		return "team creator is required"
	}
	return ""
}

func validStoredTeamOwnerKind(value string) bool {
	switch value {
	case "user", "organization":
		return true
	default:
		return false
	}
}

func organizationMemberKey(organizationID string, userID string) StorageKeyResult {
	return NewStorageKey("organization_member:" + strings.TrimSpace(organizationID) + ":" + strings.TrimSpace(userID))
}

func organizationMemberIndexKey(organizationID string) string {
	return "organization_member:index:" + strings.TrimSpace(organizationID)
}

func organizationTeamIndexKey(organizationID string) string {
	return "organization_team:index:" + strings.TrimSpace(organizationID)
}

func standaloneTeamIndexKey(ownerUserID string) string {
	return "standalone_team:index:" + strings.TrimSpace(ownerUserID)
}

func pageStoredOrganizations(values []StoredOrganization, page StoredListPage) []StoredOrganization {
	start, end := pageBounds(len(values), page)
	return values[start:end]
}

func pageStoredOrganizationMembers(values []StoredOrganizationMember, page StoredListPage) []StoredOrganizationMember {
	start, end := pageBounds(len(values), page)
	return values[start:end]
}

func pageStoredTeams(values []StoredTeam, page StoredListPage) []StoredTeam {
	start, end := pageBounds(len(values), page)
	return values[start:end]
}

func pageBounds(length int, page StoredListPage) (int, int) {
	start := page.offset
	if start > length {
		start = length
	}
	end := start + page.limit
	if end > length {
		end = length
	}
	return start, end
}

type stringIndexResult interface {
	stringIndexResult()
}

type stringIndexLoaded struct {
	values []string
}

type stringIndexStored struct{}
type stringIndexRejected struct {
	reason string
}

func (stringIndexLoaded) stringIndexResult()   {}
func (stringIndexStored) stringIndexResult()   {}
func (stringIndexRejected) stringIndexResult() {}

func loadStringIndex(storage BrowserStorage, rawKey string, label string) stringIndexResult {
	keyResult := NewStorageKey(rawKey)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return stringIndexRejected{reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	if _, missing := readResult.(StorageMissing); missing {
		return stringIndexLoaded{values: []string{}}
	}
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return stringIndexRejected{reason: storageReadReason(readResult, label)}
	}
	var values []string
	if err := json.Unmarshal([]byte(read.Value), &values); err != nil {
		return stringIndexRejected{reason: label + " index decoding failed"}
	}
	for index := range values {
		if strings.TrimSpace(values[index]) == "" {
			return stringIndexRejected{reason: label + " index contains an invalid id"}
		}
	}
	return stringIndexLoaded{values: values}
}

func appendStringIndex(storage BrowserStorage, rawKey string, id string, label string) stringIndexResult {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return stringIndexRejected{reason: label + " id is required"}
	}
	loadedResult := loadStringIndex(storage, rawKey, label)
	loaded, loadedMatched := loadedResult.(stringIndexLoaded)
	if !loadedMatched {
		return loadedResult
	}
	for index := range loaded.values {
		if loaded.values[index] == cleanID {
			return stringIndexStored{}
		}
	}
	loaded.values = append(loaded.values, cleanID)
	encoded, err := json.Marshal(loaded.values)
	if err != nil {
		return stringIndexRejected{reason: label + " index encoding failed"}
	}
	keyResult := NewStorageKey(rawKey)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return stringIndexRejected{reason: keyResult.(StorageKeyRejected).Reason}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return stringIndexRejected{reason: writeResult.(StorageWriteRejected).Reason}
	}
	return stringIndexStored{}
}

func storageReadReason(result StorageReadResult, label string) string {
	switch rejected := result.(type) {
	case StorageMissing:
		return rejected.Reason
	case StorageReadRejected:
		return rejected.Reason
	default:
		return label + " read failed"
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
	if len(attachments) > maxStoredAttachments {
		return AttachmentStorageRejected{Reason: "too many attachments"}
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
