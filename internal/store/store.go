package store

import (
  "context"
  "errors"
  "time"

  "radius-accounting-server/internal/model"
)

//Just for Interface Test
var ErrNotFound = errors.New("Record not found")

type Store interface {

    // Save stores the record. In MemoryStore it's O(1); in Redis it is a network call.
    Save(ctx context.Context, record *model.AccountingRecord, ttl time.Duration) error

    //Get retrieves a record. Memory BE's should return (nil, ErrNotFound) if missing.
    Get(ctx context.Context, key string) (*model.AccountingRecord, error)

    //List all Records
    List(ctx context.Context, prefix string) ([]*model.AccountingRecord, error)

    //Delete by Key.
    Delete(ctx context.Context, key string) error

    //Check if BE's are reachable
    Healthy(ctx context.Context) error

    Close() error
}


