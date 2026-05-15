package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nicaozx/warden-gateway/internal/sensor"
)

func main() {

	ctx := context.Background()
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

	go sensor1.Run(ctx, ch)
	go sensor2.Run(ctx, ch)
	go sensor3.Run(ctx, ch)

	for reading := range ch {
		fmt.Println(reading)
	}
}
