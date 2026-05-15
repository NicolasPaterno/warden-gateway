package sensor

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type SensorType string

const (
	Temperature SensorType = "temperature"
	Humidity    SensorType = "humidity"
	Motion      SensorType = "motion"
	CO2         SensorType = "co2"
)

type Reading struct {
	SensorID string
	Room     string
	Type     SensorType
	Value    float64
	Unit     string
	Time     time.Time
}

type Sensor struct {
	id       string
	room     string
	Type     SensorType
	interval time.Duration
	unit     string
}

func NewSensor(id, room string, sensorType SensorType, interval time.Duration) (*Sensor, error) {
	unit, err := unitForType(sensorType)
	if err != nil {
		return nil, err
	}
	return &Sensor{
		id:       id,
		room:     room,
		Type:     sensorType,
		interval: interval,
		unit:     unit,
	}, nil
}

func (s *Sensor) Run(ctx context.Context, out chan<- Reading) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	min, max := sensorRangeByType(s.Type)

	for {
		select {
		case t := <-ticker.C:
			out <- Reading{
				SensorID: s.id,
				Room:     s.room,
				Type:     s.Type,
				Value:    min + rand.Float64()*(max-min),
				Unit:     s.unit,
				Time:     t,
			}
		case <-ctx.Done():
			return
		}
	}
}

func sensorRangeByType(sensorType SensorType) (min, max float64) {
	switch sensorType {
	case Humidity:
		return 0, 100
	case Temperature:
		return -50, 100
	case Motion:
		return 0, 1
	case CO2:
		return 400, 5000
	default:
		panic(fmt.Sprintf("Unknown sensor type: %s", sensorType))
	}
}

func unitForType(sensorType SensorType) (string, error) {
	switch sensorType {
	case Humidity:
		return "%", nil
	case Temperature:
		return "°C", nil
	case Motion:
		return "bool", nil
	case CO2:
		return "ppm", nil
	default:
		return "", fmt.Errorf("unknown sensor type: %s", sensorType)
	}
}
