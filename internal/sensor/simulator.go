package sensor

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	warden "github.com/NicolasPaterno/warden-gateway"
)

type Sensor struct {
	id       string
	room     string
	sType    warden.SensorType
	interval time.Duration
	unit     string
}

func NewSensor(id, room string, sensorType warden.SensorType, interval time.Duration) (*Sensor, error) {
	unit, err := unitForType(sensorType)
	if err != nil {
		return nil, err
	}
	return &Sensor{
		id:       id,
		room:     room,
		sType:    sensorType,
		interval: interval,
		unit:     unit,
	}, nil
}

func (s *Sensor) Run(ctx context.Context, out chan<- warden.SensorReading) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	min, max := sensorRangeByType(s.sType)

	for {
		select {
		case t := <-ticker.C:
			out <- warden.SensorReading{
				SensorID:  s.id,
				Room:      s.room,
				Type:      s.sType,
				Value:     min + rand.Float64()*(max-min),
				Unit:      s.unit,
				Timestamp: t,
			}
		case <-ctx.Done():
			return
		}
	}
}

func sensorRangeByType(sensorType warden.SensorType) (min, max float64) {
	switch sensorType {
	case warden.Humidity:
		return 0, 100
	case warden.Temperature:
		return -50, 100
	case warden.Motion:
		return 0, 1
	case warden.CO2:
		return 400, 5000
	default:
		panic(fmt.Sprintf("Unknown sensor type: %s", sensorType))
	}
}

func unitForType(sensorType warden.SensorType) (string, error) {
	switch sensorType {
	case warden.Humidity:
		return "%", nil
	case warden.Temperature:
		return "°C", nil
	case warden.Motion:
		return "bool", nil
	case warden.CO2:
		return "ppm", nil
	default:
		return "", fmt.Errorf("unknown sensor type: %s", sensorType)
	}
}
