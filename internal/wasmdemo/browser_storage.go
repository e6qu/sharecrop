package wasmdemo

import (
	"encoding/json"
	"strings"
	"time"
)

type BrowserStorage interface {
	Put(StorageKey, string) StorageWriteResult
	Get(StorageKey) StorageReadResult
}

// HandlerClock lets a host supply its own notion of "now" (e.g. the browser's
// Date.now(), or a fixed value for deterministic tests/measurement) rather
// than every browser store reaching for time.Now() directly.
type HandlerClock interface {
	Now() time.Time
}

// InteractionIDSource generates entity ids the host is responsible for
// (rather than core.New*ID(), which every browser store calls directly for
// ids it can safely generate itself). NextLedgerEntryID is the only method
// still called by a browser store (AuthBrowserStore/OrgBrowserStore's
// signup/org credit grants) - the others exist for interface-compatibility
// with hosts that still expect this shape.
type InteractionIDSource interface {
	NextSubmissionID() string
	NextCommentID() string
	NextReservationID() string
	NextLedgerEntryID() string
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

type StoredListPage struct {
	limit  int
	offset int
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

func (stringIndexLoaded) stringIndexResult() {}

func (stringIndexStored) stringIndexResult() {}

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

// removeFromStringIndex drops id from the index if present, otherwise it's a
// no-op success (removing something already absent is not an error).
func removeFromStringIndex(storage BrowserStorage, rawKey string, id string) bool {
	loadedResult := loadStringIndex(storage, rawKey, "index")
	loaded, loadedMatched := loadedResult.(stringIndexLoaded)
	if !loadedMatched {
		return false
	}
	remaining := make([]string, 0, len(loaded.values))
	for _, value := range loaded.values {
		if value != id {
			remaining = append(remaining, value)
		}
	}
	encoded, err := json.Marshal(remaining)
	if err != nil {
		return false
	}
	keyResult := NewStorageKey(rawKey)
	key, keyMatched := keyResult.(StorageKeyAccepted)
	if !keyMatched {
		return false
	}
	_, matched := storage.Put(key.Value, string(encoded)).(StorageWritten)
	return matched
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
