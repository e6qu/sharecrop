package wasmdemo

import "time"

// systemTestClock is the shared HandlerClock for tests that don't exercise
// time-dependent behavior; expiry tests use their own adjustable clock.
type systemTestClock struct{}

func (systemTestClock) Now() time.Time { return time.Now() }

// adjustableTestClock lets a test advance "now" to exercise expiry sweeps.
type adjustableTestClock struct {
	now time.Time
}

func (clock *adjustableTestClock) Now() time.Time { return clock.now }

// testBrowserStorage is the shared in-memory BrowserStorage fake every
// browserstore_*_test.go file builds its test environment on.
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
