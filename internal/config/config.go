package config

import (
  "fmt"
  "os"
  "strconv"
  "time"

  "gopkg.in/yaml.v3"
)

type rawConfig struct {
  Radius     *rawRadius     `yaml:"radius"`
  Storage    *rawStorage    `yaml:"storage"`
  Redis      *rawRedis      `yaml:"redis"`
  Logging    *rawLogging    `yaml:"logging"`
  Subscriber *rawSubscriber `yaml:"subscriber"`
}

type rawRadius struct {
  ListenAddr   *string `yaml:"listen_addr"`
  SharedSecret *string `yaml:"shared_secret"`
  SessionTTL   *string `yaml:"session_ttl"`
}

type rawStorage struct {
  Backend         *string `yaml:"backend"`
  CleanupInterval *string `yaml:"cleanup_interval"`
  MaxRecords *int `yaml:"max_records"`
}

type rawRedis struct {
  Addr     *string `yaml:"addr"`
  Password *string `yaml:"password"`
  DB       *int    `yaml:"db"`
}

type rawLogging struct {
  Level      *string `yaml:"level"`
  FilePath   *string `yaml:"file_path"`
  MaxSizeMB  *int    `yaml:"max_size_mb"`
  MaxBackups *int    `yaml:"max_backups"`
  MaxAgeDays *int    `yaml:"max_age_days"`
  Compress   *bool   `yaml:"compress"`
}

type rawSubscriber struct {
  ReconnectInterval *string `yaml:"reconnect_interval"`
  MaxReconnectRetry *int    `yaml:"max_reconnect_retry"`
}

type Config struct {
  Env               string
  ListenAddr        string
  SharedSecret      string
  StorageBackend    string
  SessionTTL        time.Duration
  CleanupInterval   time.Duration
  RedisAddr         string
  RedisPassword     string
  RedisDB           int
  LogLevel          string
  LogFilePath       string
  LogMaxSizeMB      int
  LogMaxBackups     int
  LogMaxAgeDays     int
  LogCompress       bool
  ReconnectInterval time.Duration
  MaxReconnectRetry int
  MaxRecords int
}

func Load() (*Config, error) {

    env := envOr("APP_ENV", "development")
    configDir := envOr("CONFIG_DIR", "./configs")

    base, err := loadYAML(configDir + "/base.yaml")
    if err != nil && !os.IsNotExist(err) {
        return nil, fmt.Errorf("load base config: %w", err)
    }

    envCfg, err := loadYAML(configDir + "/" + env + ".yaml")
    if err != nil && !os.IsNotExist(err) {
        return nil, fmt.Errorf("load %s config: %w", env, err)
    }

    merged := merge(base, envCfg)
    cfg := resolve(env, merged)

    return cfg, cfg.validate()
}

func loadYAML(path string) (*rawConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var rc rawConfig
    if err := yaml.Unmarshal(data, &rc); err != nil {
        return nil, fmt.Errorf("parse %s: %w", path, err)
    }
    return &rc, nil
}

func merge(base, overlay *rawConfig) *rawConfig {

    if base == nil {
        base = &rawConfig{}
    }

    if overlay == nil {
        return base
    }

    result := *base

    if overlay.Radius != nil {
        if result.Radius == nil {
            result.Radius = &rawRadius{}
        }
        if overlay.Radius.ListenAddr != nil {
            result.Radius.ListenAddr = overlay.Radius.ListenAddr
        }
        if overlay.Radius.SharedSecret != nil {
            result.Radius.SharedSecret = overlay.Radius.SharedSecret
        }
        if overlay.Radius.SessionTTL != nil {
            result.Radius.SessionTTL = overlay.Radius.SessionTTL
        }
    }

    if overlay.Storage != nil {
        if result.Storage == nil {
            result.Storage = &rawStorage{}
        }
        if overlay.Storage.Backend != nil {
            result.Storage.Backend = overlay.Storage.Backend
        }
        if overlay.Storage.CleanupInterval != nil {
            result.Storage.CleanupInterval = overlay.Storage.CleanupInterval
        }
    }

    if overlay.Redis != nil {
        if result.Redis == nil {
            result.Redis = &rawRedis{}
        }
        if overlay.Redis.Addr != nil {
            result.Redis.Addr = overlay.Redis.Addr
        }
        if overlay.Redis.Password != nil {
            result.Redis.Password = overlay.Redis.Password
        }
        if overlay.Redis.DB != nil {
            result.Redis.DB = overlay.Redis.DB
        }
    }

    if overlay.Logging != nil {
        if result.Logging == nil {
            result.Logging = &rawLogging{}
        }
        if overlay.Logging.Level != nil {
            result.Logging.Level = overlay.Logging.Level
        }
        if overlay.Logging.FilePath != nil {
            result.Logging.FilePath = overlay.Logging.FilePath
        }
        if overlay.Logging.MaxSizeMB != nil {
            result.Logging.MaxSizeMB = overlay.Logging.MaxSizeMB
        }
        if overlay.Logging.MaxBackups != nil {
            result.Logging.MaxBackups = overlay.Logging.MaxBackups
        }
        if overlay.Logging.MaxAgeDays != nil {
            result.Logging.MaxAgeDays = overlay.Logging.MaxAgeDays
        }
        if overlay.Logging.Compress != nil {
            result.Logging.Compress = overlay.Logging.Compress
        }
    }

    if overlay.Subscriber != nil {
        if result.Subscriber == nil {
            result.Subscriber = &rawSubscriber{}
        }
        if overlay.Subscriber.ReconnectInterval != nil {
            result.Subscriber.ReconnectInterval = overlay.Subscriber.ReconnectInterval
        }
        if overlay.Subscriber.MaxReconnectRetry != nil {
            result.Subscriber.MaxReconnectRetry = overlay.Subscriber.MaxReconnectRetry
        }
    }

    return &result
}

func resolve(env string, raw *rawConfig) *Config {

    if raw == nil {
        raw = &rawConfig{}
    }

    return &Config{
	Env:               env,
        ListenAddr:        envOr("RADIUS_LISTEN_ADDR", ptrOr(safe(raw.Radius).ListenAddr, ":1813")),
        SharedSecret:      envOr("RADIUS_SHARED_SECRET", ptrOr(safe(raw.Radius).SharedSecret, "")),
        StorageBackend:    envOr("RADIUS_STORAGE_BACKEND", ptrOr(safe(raw.Storage).Backend, "in-memory")),
        SessionTTL:        durationOr("RADIUS_SESSION_TTL", parseDur(ptrOr(safe(raw.Radius).SessionTTL, "24h"))),
        CleanupInterval:   durationOr("RADIUS_CLEANUP_INTERVAL", parseDur(ptrOr(safe(raw.Storage).CleanupInterval, "10s"))),
        RedisAddr:         envOr("REDIS_ADDR", ptrOr(safe(raw.Redis).Addr, "localhost:6379")),
        RedisPassword:     envOr("REDIS_PASSWORD", ptrOr(safe(raw.Redis).Password, "")),
        RedisDB:           intOr("REDIS_DB", ptrOr(safe(raw.Redis).DB, 0)),
        LogLevel:          envOr("LOG_LEVEL", ptrOr(safe(raw.Logging).Level, "info")),
        LogFilePath:       envOr("LOG_FILE_PATH", ptrOr(safe(raw.Logging).FilePath, "/var/log/radius_updates.log")),
        LogMaxSizeMB:      intOr("LOG_MAX_SIZE_MB", ptrOr(safe(raw.Logging).MaxSizeMB, 100)),
        LogMaxBackups:     intOr("LOG_MAX_BACKUPS", ptrOr(safe(raw.Logging).MaxBackups, 10)),
        LogMaxAgeDays:     intOr("LOG_MAX_AGE_DAYS", ptrOr(safe(raw.Logging).MaxAgeDays, 7)),
        LogCompress:       boolOr("LOG_COMPRESS", ptrOr(safe(raw.Logging).Compress, true)),
        ReconnectInterval: durationOr("RECONNECT_INTERVAL", parseDur(ptrOr(safe(raw.Subscriber).ReconnectInterval, "2s"))),
        MaxReconnectRetry: intOr("RECONNECT_MAX_RETRY", ptrOr(safe(raw.Subscriber).MaxReconnectRetry, 0)),
	MaxRecords: intOr("RADIUS_MAX_RECORDS", ptrOr(safe(raw.Storage).MaxRecords, 1000)),
    }
}

func (c *Config) validate() error {

    if c.SharedSecret == "" {
        return fmt.Errorf("Radius Shared Secret is required (set RADIUS_SHARED_SECRET or add to config file)")
    }

    if c.StorageBackend != "in-memory" && c.StorageBackend != "redis" {
        return fmt.Errorf("Storage BE must be 'in-memory' or 'redis', got %q", c.StorageBackend)
    }

    if c.StorageBackend == "redis" && c.RedisAddr == "" {
        return fmt.Errorf("redis.addr required when storage backend is redis")
    }

    if c.SessionTTL <= 0 {
        return fmt.Errorf(" Radius Session TTL must be positive")
    }

    if c.LogMaxSizeMB <= 0 {
        return fmt.Errorf("logging.max_size_mb must be positive")
    }

    if c.LogMaxBackups < 0 {
        return fmt.Errorf("logging.max_backups must be non-negative")
    }

    if c.LogMaxAgeDays < 0 {
        return fmt.Errorf("logging.max_age_days must be non-negative")
    }

    if c.MaxRecords <= 0 {
      return fmt.Errorf("Storage Max Records must be positive")
    }

    return nil
}

// Helpers functions

func envOr(key, fallback string) string {

    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}

func durationOr(key string, fallback time.Duration) time.Duration {
    if v := os.Getenv(key); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            return d
        }
    }
    return fallback
}

func intOr(key string, fallback int) int {

    if v := os.Getenv(key); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            return n
        }
    }
    return fallback
}

func boolOr(key string, fallback bool) bool {

    if v := os.Getenv(key); v != "" {
        if b, err := strconv.ParseBool(v); err == nil {
            return b
        }
    }
    return fallback
}

func ptrOr[T any](p *T, fallback T) T {

    if p != nil {
      return *p
    }

    return fallback

}

func parseDur(s string) time.Duration {

    d, _ := time.ParseDuration(s)
    return d
}

func safe[T any](p *T) *T {

    if p != nil {
        return p
    }
    var zero T
    return &zero
}



