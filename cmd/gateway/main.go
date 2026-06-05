package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	warden "github.com/nicaozx/warden-gateway"
	httptransport "github.com/nicaozx/warden-gateway/internal/http"
	"github.com/nicaozx/warden-gateway/internal/hub"
	natspub "github.com/nicaozx/warden-gateway/internal/nats"
	"github.com/nicaozx/warden-gateway/internal/sensor"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ch := make(chan warden.SensorReading)

	h := hub.NewHub()
	go h.Run(ctx)
	wsHandler := httptransport.NewWsHandler(h)
	router := httptransport.NewRouter(wsHandler)

	go func() {
		if err := http.ListenAndServe(":8080", router); err != nil {
			log.Fatalf("http server error: %v", err)
		}
	}()

	sensor1, err := sensor.NewSensor("s1", "bedroom", warden.Humidity, 800*time.Millisecond)
	if err != nil {
		log.Fatalf("unknown sensor type: %v", err)
	}

	sensor2, err := sensor.NewSensor("s2", "bedroom", warden.Temperature, 500*time.Millisecond)
	if err != nil {
		log.Fatalf("unknown sensor type: %v", err)
	}

	sensor3, err := sensor.NewSensor("s3", "bedroom", warden.CO2, 200*time.Millisecond)
	if err != nil {
		log.Fatalf("unknown sensor type: %v", err)
	}

	publishURL := "nats://localhost:4222"
	pub, err := natspub.NewPublisher(publishURL)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer func() {
		if err := pub.Close(); err != nil {
			log.Printf("failed to drain NATS connection: %v", err)
		}
	}()

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
		err := pub.Publish(reading)
		if err != nil {
			log.Printf("error on Publish: %v", err)
		}
		go func() {
			if err := h.Broadcast(reading); err != nil {
				log.Printf("error on broadcast: %v", err)
			}
		}()
	}
}
