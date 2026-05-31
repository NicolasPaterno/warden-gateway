package warden

import "time"

type SensorType string

const (
	Temperature SensorType = "temperature"
	Humidity    SensorType = "humidity"
	Motion      SensorType = "motion"
	CO2         SensorType = "co2"
)

type SensorReading struct {
	SensorID  string     `json:"sensor_id"`
	Room      string     `json:"room"`
	Type      SensorType `json:"type"`
	Value     float64    `json:"value"`
	Unit      string     `json:"unit"`
	Timestamp time.Time  `json:"timestamp"`
}
