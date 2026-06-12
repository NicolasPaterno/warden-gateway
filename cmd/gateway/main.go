package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/NicolasPaterno/warden-auth/authn"
	warden "github.com/NicolasPaterno/warden-gateway"
	"github.com/NicolasPaterno/warden-gateway/internal/config"
	httptransport "github.com/NicolasPaterno/warden-gateway/internal/http"
	"github.com/NicolasPaterno/warden-gateway/internal/hub"
	"github.com/NicolasPaterno/warden-gateway/internal/metrics"
	natspub "github.com/NicolasPaterno/warden-gateway/internal/nats"
	"github.com/NicolasPaterno/warden-gateway/internal/pipeline"
	"github.com/NicolasPaterno/warden-gateway/internal/postgres"
	"github.com/NicolasPaterno/warden-gateway/internal/sensor"
	"github.com/NicolasPaterno/warden-gateway/internal/service"
	"github.com/NicolasPaterno/warden-gateway/internal/tracing"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := metrics.Register(); err != nil {
		slog.Error("failed to register metrics", "error", err)
		os.Exit(1)
	}

	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found")
	}
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdown, err := tracing.Init(ctx, cfg.JaegerEndpoint)
	if err != nil {
		slog.Error("failed to init tracing", "error", err)
		os.Exit(1)
	}

	defer func() {
		if err := shutdown(ctx); err != nil {
			slog.Error("failed to shutdown tracing", "error", err)
		}
	}()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := postgres.NewReadingRepo(pool)
	svc := service.NewReadingService(repo)
	readingHandler := httptransport.NewReadingsHandler(svc)

	pub, err := natspub.NewPublisher(cfg.NATSUrl)
	if err != nil {
		slog.Error("failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := pub.Close(); err != nil {
			slog.Error("failed to drain NATS connection", "error", err)
		}
	}()

	pipe := pipeline.New(svc, pub)

	healthHandler := httptransport.NewHealthHandler(pool, pub)

	h := hub.NewHub()
	go h.Run(ctx)

	// The hub is fed by NATS (not the local channel), so every pod's hub
	// receives readings from ALL pods and can serve any tenant regardless of
	// which pod a client landed on.
	sub, err := natspub.NewSubscriber(cfg.NATSUrl)
	if err != nil {
		slog.Error("failed to connect subscriber to NATS", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := sub.Close(); err != nil {
			slog.Error("failed to drain NATS subscriber", "error", err)
		}
	}()

	if err := sub.Subscribe(ctx, "warden.sensors.v1.>", func(reading warden.SensorReading) {
		if err := h.Broadcast(reading); err != nil {
			slog.Error("failed to broadcast reading", "error", err)
		}
	}); err != nil {
		slog.Error("failed to subscribe to NATS", "error", err)
		os.Exit(1)
	}

	verifier := authn.New(cfg.JWKSURL, cfg.Issuer, cfg.Audience)
	wsHandler := httptransport.NewWsHandler(h, verifier)
	router := httptransport.NewRouter(wsHandler, readingHandler, healthHandler, verifier)

	go func() {
		if err := http.ListenAndServe(cfg.HTTPPort, router); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	if !cfg.SimulatorEnabled {
		slog.Info("simulator disabled; readings come from ingestion only")
		// Block until shutdown so deferred cleanup runs.
		<-ctx.Done()
		return
	}

	sensor1, err := sensor.NewSensor("s1", "tenant-a", "bedroom", warden.Motion, cfg.SensorInterval)
	if err != nil {
		slog.Error("unknown sensor type", "error", err)
		os.Exit(1)
	}

	sensor2, err := sensor.NewSensor("s2", "tenant-a", "bedroom", warden.Temperature, cfg.SensorInterval)
	if err != nil {
		slog.Error("unknown sensor type", "error", err)
		os.Exit(1)
	}

	sensor3, err := sensor.NewSensor("s3", "tenant-b", "bedroom", warden.CO2, cfg.SensorInterval)
	if err != nil {
		slog.Error("unknown sensor type", "error", err)
		os.Exit(1)
	}

	ch := make(chan warden.SensorReading)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		sensor1.Run(ctx, ch)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sensor2.Run(ctx, ch)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sensor3.Run(ctx, ch)
	}()

	go func() {
		wg.Wait()
		close(ch)
	}()

	// Blocks until all sensor goroutines stop and ch is closed.
	pipe.Run(ctx, ch)
}
