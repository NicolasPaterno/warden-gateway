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
	SensorID  string
	Room      string
	Type      SensorType
	Value     float64
	Unit      string
	Timestamp time.Time
}
