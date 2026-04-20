package store

import (
  "context"
  "sync"
  "testing"
  "time"

  "radius-accounting-server/internal/model"
)

//Create a MemoryStore with a short cleanup interval for tests
func createTestStore(t *testing.T) *MemoryStore {
    t.Helper()
    testStore, err := InitMemoryStore(10*time.Millisecond, 1000)
    if err != nil {
      t.Fatalf("Failed to create store: %v", err)
    }

    t.Cleanup(func() { testStore.Close() })
    return testStore
}

// Build a minimal Accounting Record for testing purpose
func createTestRecord(username, sessionID string) *model.AccountingRecord {
    return &model.AccountingRecord{
                   Username:       username,
                   AcctSessionID:  sessionID,
                   AcctStatusType: model.StatusStart,
                   Timestamp:      time.Now().UTC(),
	   }
}

// Basic Memory Operations
func TestSaveAndGet(t *testing.T) {

    testStore := createTestStore(t)
    ctx := context.Background()

    record := createTestRecord("John", "sess-001")
    if err := testStore.Save(ctx, record, time.Minute); err != nil {
      t.Fatalf("Save failed: %v", err)
    }

    getRecord, err := testStore.Get(ctx, record.Key())
    if err != nil {
      t.Fatalf("Get failed: %v", err)
    }

    if getRecord.Username != record.Username {
      t.Errorf("Expected username: %q, Got: %q", record.Username, getRecord.Username)
    }
}

func TestGetNotFound(t *testing.T) {

    testStore := createTestStore(t)
    ctx := context.Background()

    _, err := testStore.Get(ctx, "radius:acct:nonexistent:sess:ts")
    if err != ErrNotFound {
      t.Errorf("Expected ErrNotFound, Got: %v", err)
    }
}

func TestDelete(t *testing.T) {

    testStore := createTestStore(t)
    ctx := context.Background()

    record := createTestRecord("Doe", "sess-002")
    if err := testStore.Save(ctx, record, time.Minute); err != nil {
      t.Fatalf("Save failed: %v", err)
    }

    if err := testStore.Delete(ctx, record.Key()); err != nil {
      t.Fatalf("Delete failed: %v", err)
    }

    _, err := testStore.Get(ctx, record.Key())
    if err != ErrNotFound {
      t.Errorf("Expected ErrNotFound after delete, Got %v:", err)
    }
}

func TestList(t *testing.T) {
    
    testStore := createTestStore(t)
    ctx := context.Background()

    testStore.Save(ctx, createTestRecord("testuser-01", "sess-003"), time.Minute)
    testStore.Save(ctx, createTestRecord("testuser-02", "sess-004"), time.Minute)
    testStore.Save(ctx, createTestRecord("testuser-03", "sess-005"), time.Minute)

    records, err := testStore.List(ctx, "radius:acct:")
    if err != nil {
      t.Fatalf("List failed: %v", err)
    }

    if len(records) != 3 {
      t.Errorf("expected 3 records, got %d", len(records))
    }
}

func TestSaveNilRecord(t *testing.T) {

    testStore := createTestStore(t)

    if err := testStore.Save(context.Background(), nil, time.Minute); err == nil {
      t.Error("Expected error saving nil record")
    }
}

func TestSaveInvalidTTL(t *testing.T) {

    testStore := createTestStore(t)

    record := createTestRecord("alice", "sess-001")

    if err := testStore.Save(context.Background(), record, 0); err == nil {
      t.Error("Expected error for zero TTL")
    }
}

// Expiry related tests

func TestExpiredRecordNotReturned(t *testing.T) {
    
    testStore := createTestStore(t)
    ctx := context.Background()

    record := createTestRecord("test-user-4", "sess-006")

    // Freeze time so the record is already expired on save
    past := time.Now().Add(-2 * time.Minute)
    testStore.now = func() time.Time { return past }
    testStore.Save(ctx, record, time.Minute)


    // Restore real time so expiry check sees the record as expired
    testStore.now = time.Now

    _, err := testStore.Get(ctx, record.Key())
    if err != ErrNotFound {
      t.Errorf("Expected ErrNotFound for expired record, got: %v", err)
    }
}

func TestCleanupRemovesExpiredRecords(t *testing.T) {

    testStore := createTestStore(t)
    ctx := context.Background()

    record := createTestRecord("test-user-5", "sess-007")

    // Save with a TTL that's already expired
    past := time.Now().Add(-2 * time.Minute)
    testStore.now = func() time.Time { return past }
    testStore.Save(ctx, record, time.Minute)

    // Restore real time and wait for cleanup loop to run
    testStore.mutex.Lock()
    testStore.now = time.Now
    testStore.mutex.Unlock(	)
    time.Sleep(50 * time.Millisecond)

    testStore.mutex.RLock()
    _, exists := testStore.records[record.Key()]
    testStore.mutex.RUnlock()

    if exists {
      t.Error("Expected expired record to be removed by cleanup loop")
    }
}

// Test cases for Concurrent Access
func TestConcurrentSaveAndGet(t *testing.T) {

    testStore := createTestStore(t)
    ctx := context.Background()

    const goroutines = 5
    var wg sync.WaitGroup
    wg.Add(goroutines)

    for i := 0; i < goroutines; i++ {

      go func(i int) {
        defer wg.Done()
        record := createTestRecord("user", string(rune('a'+i)))
	testStore.Save(ctx, record, time.Minute)
	testStore.Get(ctx, record.Key())
      }(i)
    }

    wg.Wait()
}

func TestConcurrentSaveAndList(t *testing.T) {
    
    testStore := createTestStore(t)
    ctx := context.Background()

    const goroutines = 5
    var wg sync.WaitGroup
    wg.Add(goroutines)

    for i := 0; i < goroutines; i++ {

      go func(i int) {
        defer wg.Done()
        record := createTestRecord("dummyUser", string(rune('a'+i)))
	testStore.Save(ctx, record, time.Minute)
	testStore.List(ctx, "radius:acct:")
      }(i)
    }

    wg.Wait()
}

func TestConcurrentSaveAndDelete(t *testing.T) {
    
    testStore := createTestStore(t)
    ctx := context.Background()

    const goroutines = 5
    var wg sync.WaitGroup
    wg.Add(goroutines*2)

    for i := 0; i < goroutines; i++ {
        record := createTestRecord("delUser", string(rune('a'+i)))

	go func() {
          defer wg.Done()
          testStore.Save(ctx, record, time.Minute)
	}()

	go func() {
          defer wg.Done()
          testStore.Delete(ctx, record.Key())
	}()
    }

    wg.Wait()
}


