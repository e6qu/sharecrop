package wasmdemo

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
