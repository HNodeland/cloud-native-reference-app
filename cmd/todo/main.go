package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HNodeland/cloud-native-reference-app/internal/api"
	"github.com/HNodeland/cloud-native-reference-app/internal/health"
	"github.com/HNodeland/cloud-native-reference-app/internal/store"
	"github.com/HNodeland/cloud-native-reference-app/internal/telemetry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type config struct {
	Port             string
	LogLevel         string
	TelemetryEnabled bool
	OTLPEndpoint     string
	ServiceName      string
}

func main() {
	cfg := loadConfig()

	logger, err := newLogger(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	ctx := context.Background()
	otel, shutdownTelemetry, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:      cfg.ServiceName,
		Endpoint:         cfg.OTLPEndpoint,
		TelemetryEnabled: cfg.TelemetryEnabled,
	})
	if err != nil {
		logger.Fatal("failed to initialize telemetry", zap.Error(err))
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(shutdownCtx); err != nil {
			logger.Error("failed to shut down telemetry", zap.Error(err))
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.Liveness)
	mux.HandleFunc("/readyz", health.Readiness)
	api.RegisterRoutes(mux, store.New(), logger, otel)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("server starting", zap.String("port", cfg.Port), zap.Bool("telemetryEnabled", cfg.TelemetryEnabled))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	logger.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}
}

func loadConfig() config {
	return config{
		Port:             getEnv("PORT", "8080"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		TelemetryEnabled: getEnv("TELEMETRY_ENABLED", "true") == "true",
		OTLPEndpoint:     getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
		ServiceName:      getEnv("SERVICE_NAME", "todo-api"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newLogger(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.EncoderConfig.TimeKey = "timestamp"
	var lvl zapcore.Level
	if err := lvl.Set(level); err != nil {
		return nil, err
	}
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	return cfg.Build()
}
