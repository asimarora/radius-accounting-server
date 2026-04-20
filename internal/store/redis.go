package store

import (
  "context"
  "errors"
  "time"
  "fmt"

  "github.com/redis/go-redis/v9"

  "radius-accounting-server/internal/model"
)

type RedisStore struct {
  redisClient *redis.Client
}

//Create a Redis Store
func InitRedisStore(ctx context.Context, address, password string, db int) (*RedisStore, error) {

    redisClient := redis.NewClient(&redis.Options{
	             Addr:     address,
		     Password: password,
		     DB:       db,
                   })

    return &RedisStore{redisClient: redisClient}, nil
}


//Save The Accounting Record
func (redisStore *RedisStore) Save(ctx context.Context, record *model.AccountingRecord, ttl time.Duration) error {

    if record == nil {
      return fmt.Errorf("Invalid Accounting Record")
    }

    if ttl <= 0 {
      return fmt.Errorf("Invalid TTL for given Record")
    }

    recordKey := record.Key()

    // Convert struct to bytes (JSON or Protobuf usually)
    accountRecord, err := record.Marshal()
    if err != nil {
      return err
    }

    return redisStore.redisClient.Set(ctx, recordKey, accountRecord, ttl).Err()
}

// Retrieve the record.
func (redisStore *RedisStore) Get(ctx context.Context, recordKey string) (*model.AccountingRecord, error) {

    data, err := redisStore.redisClient.Get(ctx, recordKey).Bytes()
    if err != nil {
      if errors.Is(err, redis.Nil) {
        return nil, ErrNotFound
      }

      return nil, err
    }

    accountRecord := &model.AccountingRecord{}

    //Get the Account Record from Redis Data
    if err := accountRecord.Unmarshal(data); err != nil {
      return nil, err
    }

    return accountRecord, nil
}

//List records matching prefix
func (redisStore *RedisStore) List(ctx context.Context, prefix string) ([]*model.AccountingRecord, error) {
  
    var records []*model.AccountingRecord
    var cursor uint64

    for {
      keys, nextCursor, err := redisStore.redisClient.Scan(ctx, cursor, prefix+"*", 100).Result()

      if err != nil {
        return nil, err
      }

      for _, key := range keys {
        record, err := redisStore.Get(ctx, key)
        if err != nil {
          continue
        }
        records = append(records, record)
       }

       cursor = nextCursor
       if cursor == 0 {
         break
       }
   }

   return records, nil
 }

//Delete — Removes Record by key
func (redisStore *RedisStore) Delete(ctx context.Context, key string) error {
    return redisStore.redisClient.Del(ctx, key).Err()
}

func (redisStore *RedisStore) Healthy(ctx context.Context) error {
    return redisStore.redisClient.Ping(ctx).Err()
}

func (redisStore *RedisStore) Close() error {
    return redisStore.redisClient.Close()
}

