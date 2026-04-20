package store

import (
  "context"
  "fmt"

  "radius-accounting-server/internal/config"
)

func CreateStorageBackend(cfg *config.Config) (Store, error) {

    switch cfg.StorageBackend {

      case "in-memory":
        return InitMemoryStore(cfg.CleanupInterval, cfg.MaxRecords) 
      case "redis":
        return InitRedisStore(context.Background(), cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
      default:
        return nil, fmt.Errorf("Unsupported Storage BE: %s", cfg.StorageBackend)
    
    }

}

