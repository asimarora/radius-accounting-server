package subscriber

import (
  "context"
  "fmt"
  "log/slog"
  "strings"
  "time"

  "github.com/redis/go-redis/v9"
  "gopkg.in/natefinch/lumberjack.v2"

  "radius-accounting-server/internal/config"
)

//Subsciber 
type Subscriber struct {
    redisClient       *redis.Client
    writer            *lumberjack.Logger
    logger            *slog.Logger
    reconnectInterval time.Duration
    maxReconnectRetry int
}

const logStampLayout = "2006-01-02 15:04:05.000000"

func InitSubscriber(cfg *config.Config, logger *slog.Logger) (*Subscriber, error) {
    redisClient := redis.NewClient(&redis.Options{
	                           Addr:     cfg.RedisAddr,
				   Password: cfg.RedisPassword,
				   DB:       cfg.RedisDB,
			           })

    // Redis Connectivity check
    err := redisClient.Ping(context.Background()).Err()
    if err != nil {
      return nil, fmt.Errorf("Redis connection failed: %w", err)
    }

    //Contains Radius Updates.
    fileWriter := &lumberjack.Logger{
                       Filename:   cfg.LogFilePath,
                       MaxSize:    cfg.LogMaxSizeMB,
                       MaxBackups: cfg.LogMaxBackups,
                       MaxAge:     cfg.LogMaxAgeDays,
                       Compress:   cfg.LogCompress,
                  }

    return &Subscriber{
                redisClient:       redisClient,
		writer:            fileWriter,
		logger:            logger,
		reconnectInterval: cfg.ReconnectInterval,
		maxReconnectRetry: cfg.MaxReconnectRetry,
            }, nil
}

// Listens for Redis keyspace notifications and logs events.
func (subscriber *Subscriber) Run(ctx context.Context) error {

    // Listen for SET events Format: __keyevent@*__:<event>
    pubsub := subscriber.redisClient.PSubscribe(ctx, "__keyevent@*__:set")
    //No defer - Managed manually

    subscriber.logger.Info("Subscriber started, Listening for keyspace events")

    // go-redis library pushes the messages into this Go channel.
    ch := pubsub.Channel()

    for {
      select {
        case <-ctx.Done():
          pubsub.Close()
          return ctx.Err()

        case msg, ok := <-ch:
	  if !ok {
	    subscriber.logger.Error("Redis subscription channel closed unexpectedly, Recoonecting ...")
            pubsub.Close()
	    var err error
	    pubsub, err = subscriber.reconnect(ctx)
	    if err != nil {
              return err
	    }

	    ch = pubsub.Channel()

	    continue
          }

          // Only interested in keys matching radius:acct:*
	  if strings.HasPrefix(msg.Payload, "radius:acct:") {
            subscriber.logEvent(msg.Payload)
          }
      }
    }
}

//Try Reconecting to Redis
func (subscriber *Subscriber) reconnect(ctx context.Context) (*redis.PubSub, error) {

    for attempt := 1; attempt <= subscriber.maxReconnectRetry; attempt++ {

      select {
         case <-ctx.Done():
             return nil, ctx.Err()

         case <-time.After(subscriber.reconnectInterval):
      }

      subscriber.logger.Info("Reconnecting to Redis DB,", "Attempt:", attempt)

      pubsub := subscriber.redisClient.PSubscribe(ctx, "__keyevent@*__:set")

      if err := subscriber.redisClient.Ping(ctx).Err(); err == nil {
        subscriber.logger.Info("Reconnected to Redis successfully")
	return pubsub, nil
      }

      pubsub.Close()
    }

    return nil, fmt.Errorf("Failed to reconnect after: %d attempts", subscriber.maxReconnectRetry)
 }


// Write to the updates file
func (subscriber *Subscriber) logEvent(key string) {

    // Ensures YYYY-MM-DD HH:MM:SS.ffffff
    timeStamp := time.Now().UTC().Format(logStampLayout)
    line := fmt.Sprintf("%s - Received update for key: %s\n", timeStamp, key)

    if _, err := subscriber.writer.Write([]byte(line)); err != nil {
      subscriber.logger.Error("Failed to log event,", "Error:", err, ",Key:", key)
    }
}

func (subscriber *Subscriber) Close() error {
    subscriber.logger.Info("Subscriber shutting down")

    if err := subscriber.writer.Close(); err != nil {
      subscriber.logger.Error("Failed to close log writer", "error", err)
    }

    return subscriber.redisClient.Close()
}

