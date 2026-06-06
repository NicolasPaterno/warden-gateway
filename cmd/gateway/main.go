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
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	warden "github.com/nicaozx/warden-gateway"
	"github.com/nicaozx/warden-gateway/internal/config"
	httptransport "github.com/nicaozx/warden-gateway/internal/http"
	"github.com/nicaozx/warden-gateway/internal/hub"
	natspub "github.com/nicaozx/warden-gateway/internal/nats"
	"github.com/nicaozx/warden-gateway/internal/postgres"
	"github.com/nicaozx/warden-gateway/internal/sensor"
	"github.com/nicaozx/warden-gateway/internal/service"
)

func main() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found")
	}
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ch := make(chan warden.SensorReading)

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

	healthHandler := httptransport.NewHealthHandler(pool, pub)

	h := hub.NewHub()
	go h.Run(ctx)
	wsHandler := httptransport.NewWsHandler(h)
	router := httptransport.NewRouter(wsHandler, readingHandler, healthHandler)

	go func() {
		if err := http.ListenAndServe(cfg.HTTPPort, router); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	sensor1, err := sensor.NewSensor("s1", "bedroom", warden.Humidity, 800*time.Millisecond)
	if err != nil {
		slog.Error("unknown sensor type", "error", err)
		os.Exit(1)
	}

	sensor2, err := sensor.NewSensor("s2", "bedroom", warden.Temperature, 500*time.Millisecond)
	if err != nil {
		slog.Error("unknown sensor type", "error", err)
		os.Exit(1)
	}

	sensor3, err := sensor.NewSensor("s3", "bedroom", warden.CO2, 200*time.Millisecond)
	if err != nil {
		slog.Error("unknown sensor type", "error", err)
		os.Exit(1)
	}

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

	for reading := range ch {
		if err := svc.Save(ctx, reading); err != nil {
			slog.Error("failed to save reading", "error", err)
		}
		err := pub.Publish(reading)
		if err != nil {
			slog.Error("error on publish", "error", err)
		}
		go func() {
			if err := h.Broadcast(reading); err != nil {
				slog.Error("failed to broadcast reading", "error", err)
			}
		}()
	}
}
