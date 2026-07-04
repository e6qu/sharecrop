package wasmdemo

import (
	"encoding/json"
	"strings"
)

type StoredSubmission struct {
	ID               string                            `json:"id"`
	TaskID           string                            `json:"task_id"`
	SubmitterID      string                            `json:"submitter_id"`
	State            string                            `json:"state"`
	ResponseJSON     string                            `json:"response_json"`
	ReviewNote       string                            `json:"review_note"`
	Attachments      []StoredAttachment                `json:"attachments"`
	ValidationErrors []StoredSubmissionValidationError `json:"validation_errors"`
	SensitiveFields  []StoredSubmissionSensitiveField  `json:"sensitive_fields"`
}

type StoredSubmissionValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type StoredSubmissionSensitiveField struct {
	Path       string `json:"path"`
	Category   string `json:"category"`
	Retention  string `json:"retention"`
	Redaction  string `json:"redaction"`
	State      string `json:"state"`
	RedactedAt string `json:"redacted_at"`
}

type StoredComment struct {
	ID           string `json:"id"`
	ParentKind   string `json:"parent_kind"`
	ParentID     string `json:"parent_id"`
	AuthorUserID string `json:"author_user_id"`
	Body         string `json:"body"`
	CreatedAt    string `json:"created_at"`
}

type StoredReservation struct {
	ID           string `json:"id"`
	TaskID       string `json:"task_id"`
	AssigneeKind string `json:"assignee_kind"`
	AssigneeID   string `json:"assignee_id"`
	State        string `json:"state"`
	RequestedBy  string `json:"requested_by"`
	// IssuedWorkerCredential mirrors the real backend's one-time reveal
	// field so the Elm client's response decoder stays in sync; the WASM
	// demo doesn't auto-issue task-scoped credentials, so this is always
	// empty here.
	IssuedWorkerCredential string `json:"issued_worker_credential"`
}

type StoredLedgerEntry struct {
	ID        string `json:"id"`
	OwnerKind string `json:"owner_kind"`
	OwnerID   string `json:"owner_id"`
	Kind      string `json:"kind"`
	Amount    int64  `json:"amount"`
	TaskID    string `json:"task_id"`
}

type SubmissionStorageResult interface {
	submissionStorageResult()
}

type SubmissionStored struct {
	Value StoredSubmission
}

type SubmissionsStored struct {
	Values []StoredSubmission
}

type SubmissionStorageRejected struct {
	Reason string
}

func (SubmissionStored) submissionStorageResult()          {}
func (SubmissionsStored) submissionStorageResult()         {}
func (SubmissionStorageRejected) submissionStorageResult() {}

func SaveSubmission(storage BrowserStorage, submission StoredSubmission) SubmissionStorageResult {
	cleaned := cleanStoredSubmission(submission)
	if reason := validateStoredSubmission(cleaned); reason != "" {
		return SubmissionStorageRejected{Reason: reason}
	}
	keyResult := NewStorageKey("submission:" + cleaned.ID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return SubmissionStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return SubmissionStorageRejected{Reason: "submission encoding failed"}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return SubmissionStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	taskIndexResult := appendStringIndex(storage, "submission:index:task:"+cleaned.TaskID, cleaned.ID, "submission")
	if _, matched := taskIndexResult.(stringIndexStored); !matched {
		return SubmissionStorageRejected{Reason: taskIndexResult.(stringIndexRejected).reason}
	}
	userIndexResult := appendStringIndex(storage, "submission:index:user:"+cleaned.SubmitterID, cleaned.ID, "submission")
	if _, matched := userIndexResult.(stringIndexStored); !matched {
		return SubmissionStorageRejected{Reason: userIndexResult.(stringIndexRejected).reason}
	}
	globalIndexResult := appendStringIndex(storage, "submission:index", cleaned.ID, "submission")
	if _, matched := globalIndexResult.(stringIndexStored); !matched {
		return SubmissionStorageRejected{Reason: globalIndexResult.(stringIndexRejected).reason}
	}
	return SubmissionStored{Value: cleaned}
}

func LoadSubmission(storage BrowserStorage, submissionID string) SubmissionStorageResult {
	cleanID := strings.TrimSpace(submissionID)
	if cleanID == "" {
		return SubmissionStorageRejected{Reason: "submission id is required"}
	}
	keyResult := NewStorageKey("submission:" + cleanID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return SubmissionStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return SubmissionStorageRejected{Reason: storageReadReason(readResult, "submission")}
	}
	var submission StoredSubmission
	if err := json.Unmarshal([]byte(read.Value), &submission); err != nil {
		return SubmissionStorageRejected{Reason: "submission decoding failed"}
	}
	cleaned := cleanStoredSubmission(submission)
	if cleaned.ID != cleanID {
		return SubmissionStorageRejected{Reason: "submission storage key contains mismatched record"}
	}
	if reason := validateStoredSubmission(cleaned); reason != "" {
		return SubmissionStorageRejected{Reason: reason}
	}
	return SubmissionStored{Value: cleaned}
}

func ListTaskSubmissions(storage BrowserStorage, taskID string, page StoredListPage) SubmissionStorageResult {
	return listSubmissionsFromIndex(storage, "submission:index:task:"+strings.TrimSpace(taskID), "task", strings.TrimSpace(taskID), page)
}

func ListUserSubmissions(storage BrowserStorage, userID string, page StoredListPage) SubmissionStorageResult {
	return listSubmissionsFromIndex(storage, "submission:index:user:"+strings.TrimSpace(userID), "user", strings.TrimSpace(userID), page)
}

func ListAllSubmissions(storage BrowserStorage, page StoredListPage) SubmissionStorageResult {
	return listSubmissionsFromIndex(storage, "submission:index", "all", "all", page)
}

func listSubmissionsFromIndex(storage BrowserStorage, indexKey string, ownerKind string, ownerID string, page StoredListPage) SubmissionStorageResult {
	if ownerID == "" {
		return SubmissionStorageRejected{Reason: ownerKind + " id is required"}
	}
	idsResult := loadStringIndex(storage, indexKey, "submission")
	ids, idsMatched := idsResult.(stringIndexLoaded)
	if !idsMatched {
		return SubmissionStorageRejected{Reason: idsResult.(stringIndexRejected).reason}
	}
	start, end := pageBounds(len(ids.values), page)
	values := make([]StoredSubmission, 0, end-start)
	for index := start; index < end; index++ {
		loadResult := LoadSubmission(storage, ids.values[index])
		loaded, loadedMatched := loadResult.(SubmissionStored)
		if !loadedMatched {
			return loadResult
		}
		if ownerKind == "task" && loaded.Value.TaskID != ownerID {
			return SubmissionStorageRejected{Reason: "submission task index contains mismatched record"}
		}
		if ownerKind == "user" && loaded.Value.SubmitterID != ownerID {
			return SubmissionStorageRejected{Reason: "submission user index contains mismatched record"}
		}
		values = append(values, loaded.Value)
	}
	return SubmissionsStored{Values: values}
}

func cleanStoredSubmission(submission StoredSubmission) StoredSubmission {
	return StoredSubmission{
		ID:               strings.TrimSpace(submission.ID),
		TaskID:           strings.TrimSpace(submission.TaskID),
		SubmitterID:      strings.TrimSpace(submission.SubmitterID),
		State:            strings.TrimSpace(submission.State),
		ResponseJSON:     strings.TrimSpace(submission.ResponseJSON),
		ReviewNote:       strings.TrimSpace(submission.ReviewNote),
		Attachments:      submission.Attachments,
		ValidationErrors: submission.ValidationErrors,
		SensitiveFields:  submission.SensitiveFields,
	}
}

func validateStoredSubmission(submission StoredSubmission) string {
	if submission.ID == "" {
		return "submission id is required"
	}
	if submission.TaskID == "" {
		return "submission task id is required"
	}
	if submission.SubmitterID == "" {
		return "submission submitter id is required"
	}
	if !validStoredSubmissionState(submission.State) {
		return "submission state is invalid"
	}
	if submission.ResponseJSON == "" {
		return "submission response is required"
	}
	return ""
}

func validStoredSubmissionState(value string) bool {
	switch value {
	case "submitted", "invalid", "accepted", "changes_requested", "rejected":
		return true
	default:
		return false
	}
}

type CommentStorageResult interface {
	commentStorageResult()
}

type CommentStored struct {
	Value StoredComment
}

type CommentsStored struct {
	Values []StoredComment
}

type CommentStorageRejected struct {
	Reason string
}

func (CommentStored) commentStorageResult()          {}
func (CommentsStored) commentStorageResult()         {}
func (CommentStorageRejected) commentStorageResult() {}

func SaveComment(storage BrowserStorage, comment StoredComment) CommentStorageResult {
	cleaned := cleanStoredComment(comment)
	if reason := validateStoredComment(cleaned); reason != "" {
		return CommentStorageRejected{Reason: reason}
	}
	keyResult := NewStorageKey("comment:" + cleaned.ID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return CommentStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return CommentStorageRejected{Reason: "comment encoding failed"}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return CommentStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	indexResult := appendStringIndex(storage, "comment:index:"+cleaned.ParentKind+":"+cleaned.ParentID, cleaned.ID, "comment")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return CommentStorageRejected{Reason: indexResult.(stringIndexRejected).reason}
	}
	return CommentStored{Value: cleaned}
}

func ListComments(storage BrowserStorage, parentKind string, parentID string) CommentStorageResult {
	cleanKind := strings.TrimSpace(parentKind)
	cleanID := strings.TrimSpace(parentID)
	if !validStoredCommentParentKind(cleanKind) {
		return CommentStorageRejected{Reason: "comment parent kind is invalid"}
	}
	if cleanID == "" {
		return CommentStorageRejected{Reason: "comment parent id is required"}
	}
	idsResult := loadStringIndex(storage, "comment:index:"+cleanKind+":"+cleanID, "comment")
	ids, idsMatched := idsResult.(stringIndexLoaded)
	if !idsMatched {
		return CommentStorageRejected{Reason: idsResult.(stringIndexRejected).reason}
	}
	values := make([]StoredComment, 0, len(ids.values))
	for index := range ids.values {
		keyResult := NewStorageKey("comment:" + ids.values[index])
		key, keyMatched := keyResult.(StorageKeyAccepted)
		if !keyMatched {
			return CommentStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
		}
		readResult := storage.Get(key.Value)
		read, readMatched := readResult.(StorageRead)
		if !readMatched {
			return CommentStorageRejected{Reason: storageReadReason(readResult, "comment")}
		}
		var comment StoredComment
		if err := json.Unmarshal([]byte(read.Value), &comment); err != nil {
			return CommentStorageRejected{Reason: "comment decoding failed"}
		}
		cleaned := cleanStoredComment(comment)
		if cleaned.ID != ids.values[index] || cleaned.ParentKind != cleanKind || cleaned.ParentID != cleanID {
			return CommentStorageRejected{Reason: "comment index contains mismatched record"}
		}
		if reason := validateStoredComment(cleaned); reason != "" {
			return CommentStorageRejected{Reason: reason}
		}
		values = append(values, cleaned)
	}
	return CommentsStored{Values: values}
}

func cleanStoredComment(comment StoredComment) StoredComment {
	return StoredComment{
		ID:           strings.TrimSpace(comment.ID),
		ParentKind:   strings.TrimSpace(comment.ParentKind),
		ParentID:     strings.TrimSpace(comment.ParentID),
		AuthorUserID: strings.TrimSpace(comment.AuthorUserID),
		Body:         strings.TrimSpace(comment.Body),
		CreatedAt:    strings.TrimSpace(comment.CreatedAt),
	}
}

func validateStoredComment(comment StoredComment) string {
	if comment.ID == "" {
		return "comment id is required"
	}
	if !validStoredCommentParentKind(comment.ParentKind) {
		return "comment parent kind is invalid"
	}
	if comment.ParentID == "" {
		return "comment parent id is required"
	}
	if comment.AuthorUserID == "" {
		return "comment author is required"
	}
	if comment.Body == "" {
		return "comment body is required"
	}
	if comment.CreatedAt == "" {
		return "comment created time is required"
	}
	return ""
}

func validStoredCommentParentKind(value string) bool {
	switch value {
	case "task", "submission":
		return true
	default:
		return false
	}
}

type ReservationStorageResult interface {
	reservationStorageResult()
}

type ReservationStored struct {
	Value StoredReservation
}

type ReservationsStored struct {
	Values []StoredReservation
}

type ReservationStorageRejected struct {
	Reason string
}

func (ReservationStored) reservationStorageResult()          {}
func (ReservationsStored) reservationStorageResult()         {}
func (ReservationStorageRejected) reservationStorageResult() {}

func SaveReservation(storage BrowserStorage, reservation StoredReservation) ReservationStorageResult {
	cleaned := cleanStoredReservation(reservation)
	if reason := validateStoredReservation(cleaned); reason != "" {
		return ReservationStorageRejected{Reason: reason}
	}
	keyResult := NewStorageKey("reservation:" + cleaned.ID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return ReservationStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return ReservationStorageRejected{Reason: "reservation encoding failed"}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return ReservationStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	indexResult := appendStringIndex(storage, "reservation:index:task:"+cleaned.TaskID, cleaned.ID, "reservation")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return ReservationStorageRejected{Reason: indexResult.(stringIndexRejected).reason}
	}
	return ReservationStored{Value: cleaned}
}

func LoadReservation(storage BrowserStorage, reservationID string) ReservationStorageResult {
	cleanID := strings.TrimSpace(reservationID)
	if cleanID == "" {
		return ReservationStorageRejected{Reason: "reservation id is required"}
	}
	keyResult := NewStorageKey("reservation:" + cleanID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return ReservationStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return ReservationStorageRejected{Reason: storageReadReason(readResult, "reservation")}
	}
	var reservation StoredReservation
	if err := json.Unmarshal([]byte(read.Value), &reservation); err != nil {
		return ReservationStorageRejected{Reason: "reservation decoding failed"}
	}
	cleaned := cleanStoredReservation(reservation)
	if cleaned.ID != cleanID {
		return ReservationStorageRejected{Reason: "reservation storage key contains mismatched record"}
	}
	if reason := validateStoredReservation(cleaned); reason != "" {
		return ReservationStorageRejected{Reason: reason}
	}
	return ReservationStored{Value: cleaned}
}

func ListTaskReservations(storage BrowserStorage, taskID string, page StoredListPage) ReservationStorageResult {
	cleanTaskID := strings.TrimSpace(taskID)
	if cleanTaskID == "" {
		return ReservationStorageRejected{Reason: "reservation task id is required"}
	}
	idsResult := loadStringIndex(storage, "reservation:index:task:"+cleanTaskID, "reservation")
	ids, idsMatched := idsResult.(stringIndexLoaded)
	if !idsMatched {
		return ReservationStorageRejected{Reason: idsResult.(stringIndexRejected).reason}
	}
	start, end := pageBounds(len(ids.values), page)
	values := make([]StoredReservation, 0, end-start)
	for index := start; index < end; index++ {
		loadResult := LoadReservation(storage, ids.values[index])
		loaded, loadedMatched := loadResult.(ReservationStored)
		if !loadedMatched {
			return loadResult
		}
		if loaded.Value.TaskID != cleanTaskID {
			return ReservationStorageRejected{Reason: "reservation task index contains mismatched record"}
		}
		values = append(values, loaded.Value)
	}
	return ReservationsStored{Values: values}
}

func TransitionReservation(storage BrowserStorage, taskID string, reservationID string, nextState string) ReservationStorageResult {
	loadResult := LoadReservation(storage, reservationID)
	loaded, loadedMatched := loadResult.(ReservationStored)
	if !loadedMatched {
		return loadResult
	}
	if loaded.Value.TaskID != strings.TrimSpace(taskID) {
		return ReservationStorageRejected{Reason: "reservation does not belong to task"}
	}
	if !validStoredReservationTransition(loaded.Value.State, nextState) {
		return ReservationStorageRejected{Reason: "reservation state transition is invalid"}
	}
	loaded.Value.State = strings.TrimSpace(nextState)
	return SaveReservation(storage, loaded.Value)
}

func cleanStoredReservation(reservation StoredReservation) StoredReservation {
	return StoredReservation{
		ID:           strings.TrimSpace(reservation.ID),
		TaskID:       strings.TrimSpace(reservation.TaskID),
		AssigneeKind: strings.TrimSpace(reservation.AssigneeKind),
		AssigneeID:   strings.TrimSpace(reservation.AssigneeID),
		State:        strings.TrimSpace(reservation.State),
		RequestedBy:  strings.TrimSpace(reservation.RequestedBy),
	}
}

func validateStoredReservation(reservation StoredReservation) string {
	if reservation.ID == "" {
		return "reservation id is required"
	}
	if reservation.TaskID == "" {
		return "reservation task id is required"
	}
	if !validStoredReservationAssigneeKind(reservation.AssigneeKind) {
		return "reservation assignee kind is invalid"
	}
	if reservation.AssigneeID == "" {
		return "reservation assignee id is required"
	}
	if !validStoredReservationState(reservation.State) {
		return "reservation state is invalid"
	}
	if reservation.RequestedBy == "" {
		return "reservation requester is required"
	}
	return ""
}

func validStoredReservationAssigneeKind(value string) bool {
	switch value {
	case "user", "team", "organization_team":
		return true
	default:
		return false
	}
}

func validStoredReservationState(value string) bool {
	switch value {
	case "requested", "active", "declined", "cancelled":
		return true
	default:
		return false
	}
}

func validStoredReservationTransition(current string, next string) bool {
	switch next {
	case "active", "declined":
		return current == "requested"
	case "cancelled":
		return current == "requested" || current == "active"
	default:
		return false
	}
}

type LedgerStorageResult interface {
	ledgerStorageResult()
}

type LedgerEntryStored struct {
	Value StoredLedgerEntry
}

type LedgerEntriesStored struct {
	Values []StoredLedgerEntry
}

type LedgerBalanceStored struct {
	Amount int64
}

type LedgerStorageRejected struct {
	Reason string
}

func (LedgerEntryStored) ledgerStorageResult()     {}
func (LedgerEntriesStored) ledgerStorageResult()   {}
func (LedgerBalanceStored) ledgerStorageResult()   {}
func (LedgerStorageRejected) ledgerStorageResult() {}

func SaveLedgerEntry(storage BrowserStorage, entry StoredLedgerEntry) LedgerStorageResult {
	cleaned := cleanStoredLedgerEntry(entry)
	if reason := validateStoredLedgerEntry(cleaned); reason != "" {
		return LedgerStorageRejected{Reason: reason}
	}
	keyResult := NewStorageKey("ledger:" + cleaned.ID)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return LedgerStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return LedgerStorageRejected{Reason: "ledger entry encoding failed"}
	}
	writeResult := storage.Put(key.Value, string(encoded))
	if _, matched := writeResult.(StorageWritten); !matched {
		return LedgerStorageRejected{Reason: writeResult.(StorageWriteRejected).Reason}
	}
	indexResult := appendStringIndex(storage, "ledger:index:"+cleaned.OwnerKind+":"+cleaned.OwnerID, cleaned.ID, "ledger entry")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return LedgerStorageRejected{Reason: indexResult.(stringIndexRejected).reason}
	}
	return LedgerEntryStored{Value: cleaned}
}

func ListLedgerEntries(storage BrowserStorage, ownerKind string, ownerID string, page StoredListPage) LedgerStorageResult {
	cleanKind := strings.TrimSpace(ownerKind)
	cleanID := strings.TrimSpace(ownerID)
	if !validStoredLedgerOwnerKind(cleanKind) {
		return LedgerStorageRejected{Reason: "ledger owner kind is invalid"}
	}
	if cleanID == "" {
		return LedgerStorageRejected{Reason: "ledger owner id is required"}
	}
	idsResult := loadStringIndex(storage, "ledger:index:"+cleanKind+":"+cleanID, "ledger entry")
	ids, idsMatched := idsResult.(stringIndexLoaded)
	if !idsMatched {
		return LedgerStorageRejected{Reason: idsResult.(stringIndexRejected).reason}
	}
	start, end := pageBounds(len(ids.values), page)
	values := make([]StoredLedgerEntry, 0, end-start)
	for index := start; index < end; index++ {
		loadResult := loadLedgerEntry(storage, ids.values[index])
		loaded, loadedMatched := loadResult.(LedgerEntryStored)
		if !loadedMatched {
			return loadResult
		}
		if loaded.Value.OwnerKind != cleanKind || loaded.Value.OwnerID != cleanID {
			return LedgerStorageRejected{Reason: "ledger index contains mismatched record"}
		}
		values = append(values, loaded.Value)
	}
	return LedgerEntriesStored{Values: values}
}

func LedgerBalance(storage BrowserStorage, ownerKind string, ownerID string) LedgerStorageResult {
	entriesResult := ListLedgerEntries(storage, ownerKind, ownerID, StoredListPage{limit: 1000000, offset: 0})
	entries, entriesMatched := entriesResult.(LedgerEntriesStored)
	if !entriesMatched {
		return entriesResult
	}
	var amount int64
	if strings.TrimSpace(ownerKind) == "user" {
		amount = 100
	}
	for index := range entries.Values {
		amount += entries.Values[index].Amount
	}
	return LedgerBalanceStored{Amount: amount}
}

func loadLedgerEntry(storage BrowserStorage, ledgerID string) LedgerStorageResult {
	keyResult := NewStorageKey("ledger:" + strings.TrimSpace(ledgerID))
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return LedgerStorageRejected{Reason: keyResult.(StorageKeyRejected).Reason}
	}
	readResult := storage.Get(key.Value)
	read, readMatched := readResult.(StorageRead)
	if !readMatched {
		return LedgerStorageRejected{Reason: storageReadReason(readResult, "ledger entry")}
	}
	var entry StoredLedgerEntry
	if err := json.Unmarshal([]byte(read.Value), &entry); err != nil {
		return LedgerStorageRejected{Reason: "ledger entry decoding failed"}
	}
	cleaned := cleanStoredLedgerEntry(entry)
	if cleaned.ID != strings.TrimSpace(ledgerID) {
		return LedgerStorageRejected{Reason: "ledger storage key contains mismatched record"}
	}
	if reason := validateStoredLedgerEntry(cleaned); reason != "" {
		return LedgerStorageRejected{Reason: reason}
	}
	return LedgerEntryStored{Value: cleaned}
}

func cleanStoredLedgerEntry(entry StoredLedgerEntry) StoredLedgerEntry {
	return StoredLedgerEntry{
		ID:        strings.TrimSpace(entry.ID),
		OwnerKind: strings.TrimSpace(entry.OwnerKind),
		OwnerID:   strings.TrimSpace(entry.OwnerID),
		Kind:      strings.TrimSpace(entry.Kind),
		Amount:    entry.Amount,
		TaskID:    strings.TrimSpace(entry.TaskID),
	}
}

func validateStoredLedgerEntry(entry StoredLedgerEntry) string {
	if entry.ID == "" {
		return "ledger entry id is required"
	}
	if !validStoredLedgerOwnerKind(entry.OwnerKind) {
		return "ledger owner kind is invalid"
	}
	if entry.OwnerID == "" {
		return "ledger owner id is required"
	}
	if entry.Kind == "" {
		return "ledger entry kind is required"
	}
	return ""
}

func validStoredLedgerOwnerKind(value string) bool {
	switch value {
	case "user", "organization":
		return true
	default:
		return false
	}
}
