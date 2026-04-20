package store

import (
  "context"
  "fmt"
  "sync"
  "time"
  "strings"

  "radius-accounting-server/internal/model"
)

//Whole memory Store
type MemoryStore struct {
  mutex sync.RWMutex
  records map[string] recordEntry
  maxRecords int
  //No Data. Only for signaling an event
  stopCh chan struct{}
  wg sync.WaitGroup
  now func() time.Time
}

//Accounting Record with Expiry 
type recordEntry struct {
  data *model.AccountingRecord
  expiry time.Time
}

//Create a Memory Store
func InitMemoryStore(cleanupInterval time.Duration,  maxRecords int) (*MemoryStore, error) {

    if cleanupInterval <= 0 {
      return nil, fmt.Errorf("Cleanup Interval is invalid")
    }

    if maxRecords <= 0 {
      return nil, fmt.Errorf("Max Records is invalid")
    }

    memoryStore := &MemoryStore{
                      records:  make(map[string]recordEntry),
                      maxRecords: maxRecords,
		      stopCh: make(chan struct{}),
		      now: time.Now,
	            }

    memoryStore.wg.Add(1)

    //Create a Go Routine for Cleanup of Old Entries.
    go memoryStore.cleanupLoop(cleanupInterval)

    return memoryStore, nil
}

//Save The Accounting Record
func (memoryStore *MemoryStore) Save(_ context.Context, record *model.AccountingRecord, ttl time.Duration) error {

    if record == nil {
      return fmt.Errorf("Invalid Accounting Record")
    }

    if ttl <= 0 {
      return fmt.Errorf("Invalid TTL for given Record")
    }

    memoryStore.mutex.Lock()
    defer memoryStore.mutex.Unlock()


    // Check for capacity otherwise cleanup Random entry

    if len(memoryStore.records) >= memoryStore.maxRecords {
      for record := range memoryStore.records {
        delete(memoryStore.records, record)
	break
      }
    }

    recordKey := record.Key()

    // Add record with TTL
    memoryStore.records[recordKey] = recordEntry {
	                         data: record,
				 expiry: memoryStore.now().Add(ttl),
                               }
    return nil
}

// Retrieve the record.
func (memoryStore *MemoryStore) Get(_ context.Context, recordKey string) (*model.AccountingRecord, error) {

    //Read Lock
    memoryStore.mutex.RLock()
    defer memoryStore.mutex.RUnlock()

    record, ok := memoryStore.records[recordKey]

    //Record already expired
    if !ok || record.expiry.Before(memoryStore.now()) {
      return nil, ErrNotFound
    }

    //Return the record 
    return record.data, nil
}

//List records matching prefix(Non Expired)
func (memoryStore *MemoryStore) List(_ context.Context, prefix string) ([]*model.AccountingRecord, error) {

    //Get a Read Lock
    memoryStore.mutex.RLock()
    defer memoryStore.mutex.RUnlock()

    now := memoryStore.now()

    var records []*model.AccountingRecord

    for key, entry := range memoryStore.records {

      if entry.expiry.Before(now) {
        continue
      }

      if strings.HasPrefix(key, prefix) {
        records = append(records, entry.data)
      }
    }

    return records, nil
}

//Delete — Removes Record by key
func (memoryStore *MemoryStore) Delete(_ context.Context, key string) error {

    memoryStore.mutex.Lock()
    defer memoryStore.mutex.Unlock()

    delete(memoryStore.records, key)
    return nil
}

func (memoryStore *MemoryStore) cleanupLoop(interval time.Duration) {
    
    //Start a new time ticker
    timeTicker := time.NewTicker(interval)

    defer func() {
      timeTicker.Stop()
      memoryStore.wg.Done()
    }()

    for {
      select {
        case <-timeTicker.C:
          memoryStore.removeExpired()

        case <-memoryStore.stopCh:
          return
      }
    }
}

func (memoryStore *MemoryStore) removeExpired() {

    //Create a separate Slice for records what we wanna delete	
    var expiredRecords []string

    //Read lock to scan for Records.
    memoryStore.mutex.RLock()

    now := memoryStore.now()
    for recordKey, record := range memoryStore.records {

      if record.expiry.Before(now) {
        expiredRecords = append(expiredRecords, recordKey)
      }

    }
    
    memoryStore.mutex.RUnlock()

    if len(expiredRecords) > 0 {
      memoryStore.mutex.Lock()

      for _, data := range expiredRecords{
        delete(memoryStore.records, data)
      }

      memoryStore.mutex.Unlock()
    }
}

func (memoryStore *MemoryStore) Healthy(_ context.Context) error {
    return nil
}

// Gracefully shuts down the Cleanup routine.
func (memoryStore *MemoryStore) Close() error {
    close(memoryStore.stopCh)
    memoryStore.wg.Wait()
    return nil
}


