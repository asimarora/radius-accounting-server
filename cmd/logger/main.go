package main

import (
  "context"
  "log/slog"
  "os"
  "os/signal"
  "syscall"
  "fmt"

  "radius-accounting-server/internal/config"
  "radius-accounting-server/internal/subscriber"
)

//Entry point to Radius Control Plane Logger
func main() {

    //Run the application and capture any fatal errors
    if err := run(); err != nil {
      fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
      os.Exit(1)
    }
}

//Core Application logic is abstracted into run() function
func run() error {

    //Load Configuration
    cfg, err := config.Load()
    if err != nil {
      fmt.Fprintf(os.Stderr, "Failed to load configuration: %s\n", err)
      return err
    }

    //Initialize Logger based on configuration
    logger := initializeLogger(cfg)

    sub, err := subscriber.InitSubscriber(cfg, logger)
    if err != nil {
      logger.Error("Failed to initialize subscriber", "error", err)
      return err
    }

    defer sub.Close()

    sigCtx, sigStop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer sigStop()

    errCh := make(chan error, 1)

    go func() {
      logger.Info("Redis subscriber started")
      errCh <- sub.Run(sigCtx)
    }()

    select {
      case <-sigCtx.Done():
        logger.Info("Received shutdown signal")
      case err := <-errCh:
        if err != nil {
          logger.Error("Subscriber Error,", "Error:", err)
          return err
        }
    }

    logger.Info("Redis subscriber stopped")
    return nil
}

//Creates and configure the logger based on config
func initializeLogger(cfg *config.Config) *slog.Logger {
    level := mapLogLevel(cfg.LogLevel)
    return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

//Helper function for mapping log level strings to slog.Level
func mapLogLevel(logLevel string) slog.Level {

    switch logLevel {

        case "debug":
            return slog.LevelDebug

        case "info":
            return slog.LevelInfo

        case "warn":
            return slog.LevelWarn

        case "error":
            return slog.LevelError
        //Default case
        default:
            return slog.LevelInfo
    }

}
