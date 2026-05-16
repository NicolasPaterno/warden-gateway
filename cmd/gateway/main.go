package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nicaozx/warden-gateway/internal/publisher"
	"github.com/nicaozx/warden-gateway/internal/sensor"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ch := make(chan sensor.Reading)

	sensor1, err := sensor.NewSensor("s1", "bedroom", sensor.Humidity, 800*time.Second)
	if err != nil {
		panic(fmt.Sprintf("unknown sensor type: %s", err))
	}

	sensor2, err := sensor.NewSensor("s2", "bedroom", sensor.Temperature, 500*time.Millisecond)
	if err != nil {
		panic(fmt.Sprintf("unknown sensor type: %s", err))
	}

	sensor3, err := sensor.NewSensor("s3", "bedroom", sensor.CO2, 200*time.Millisecond)
	if err != nil {
		panic(fmt.Sprintf("unknown sensor type: %s", err))
	}
	publishUrl := "nats://localhost:4222"
	pub, err := publisher.NewPublisher(publishUrl)
	if err != nil {
		panic(err)
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
		pub.Publish(reading)
	}
}
