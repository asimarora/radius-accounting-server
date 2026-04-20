package main

import (
    "context"
    "fmt"
    "log/slog"
    "net"
    "os"
    "os/signal"
    "syscall"
    "time"
    "errors"

    "layeh.com/radius"

    "radius-accounting-server/internal/config"
    "radius-accounting-server/internal/handler"
    "radius-accounting-server/internal/store"
)

//Entry point to Radius Accounting Server
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

    //Initialize Storage Backend
    storage, err := store.CreateStorageBackend(cfg)
    if err != nil {
      logger.Error("Failed to initialize Storage Backend, Error",err)
      return err
    }

    defer storage.Close()

    //Run health check on Storage Backend
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := storage.Healthy(ctx); err != nil {
      logger.Error("Health check failure",err)
      return err
    }

    //Initialize the Radius Accounting handler
    acctHandler := handler.InitAccountingHandler(storage, cfg, logger)

    //Start the UDP Server
    udpServer, err := net.ListenPacket("udp", cfg.ListenAddr)
    if err != nil {
      logger.Error("Failed to bind UDP listener to address:", cfg.ListenAddr, ",Error:", err)
      return err
    }

    defer udpServer.Close()

    radiusServer := &radius.PacketServer {
                 Handler: acctHandler,
                 SecretSource: radius.StaticSecretSource([]byte(cfg.SharedSecret)),
              }

    //Signal Handling for Shutdown
    sigCtx, sigStop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer sigStop()

    errorChannel := make(chan error, 1)

    //Start the Radius Accounting Server in a Go Routine

    go func() {
      logger.Info("Radius accounting server started:", "Address:", cfg.ListenAddr, "Storage BE", cfg.StorageBackend)
      errorChannel <- radiusServer.Serve(udpServer)
    }()


    //Wait for Server Error or Shutdown Signal
    select {
      case <- sigCtx.Done():
        logger.Info("Recieved Shutdown Signal")

	// Shutdown timeout context
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer shutdownCancel()

	if err := radiusServer.Shutdown(shutdownCtx); err != nil {
          logger.Error("Graceful server shutdown failed,", "Error:", err)
	  return err
        }

      case err := <-errorChannel:
        if err != nil {
	  logger.Error("Server runtime error,", "Error", err)
          return err
        }

	//Silent stop case (unexpected)
	logger.Error("Server stopped unexpectedly without an error")
	return errors.New("Unexpected server Error")
    }

    logger.Info("Radius Accounting server has stopped successfully")
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
