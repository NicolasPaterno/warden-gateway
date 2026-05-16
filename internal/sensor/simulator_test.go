package sensor

import (
	"context"
	"testing"
	"time"
)

func TestNewSensor_ValidType(t *testing.T) {
	s, err := NewSensor("sensor-1", "living-room", Temperature, time.Second)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.unit != "°C" {
		t.Errorf("expected unit °C, got %s", s.unit)
	}
}

func TestNewSensor_InvalidType(t *testing.T) {
	s, err := NewSensor("sensor-1", "living-room", SensorType("smoke"), time.Second)
	if err == nil {
		t.Fatalf("expected error, got none")
	}
	if s != nil {
		t.Errorf("expected nil sensor, got %v", s)
	}
}

func TestSensorRangeByType(t *testing.T) {
	tests := []struct {
		sensorType SensorType
		min        float64
		max        float64
	}{
		{Temperature, -50, 100},
		{Humidity, 0, 100},
		{Motion, 0, 1},
		{CO2, 400, 5000},
	}

	for _, tt := range tests {
		t.Run(string(tt.sensorType), func(t *testing.T) {
			min, max := sensorRangeByType(tt.sensorType)
			if min != tt.min || max != tt.max {
				t.Errorf("expected [%v, %v], got [%v, %v]", tt.min, tt.max, min, max)
			}
		})
	}
}

func TestSensorRangeByType_UnknownTypePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for unknown sensor type")
		}
	}()
	sensorRangeByType(SensorType("smoke"))
}

func TestUnitForType(t *testing.T) {
	tests := []struct {
		sensorType SensorType
		unit       string
	}{
		{Temperature, "°C"},
		{Humidity, "%"},
		{Motion, "bool"},
		{CO2, "ppm"},
	}

	for _, tt := range tests {
		t.Run(string(tt.sensorType), func(t *testing.T) {
			unit, err := unitForType(tt.sensorType)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if unit != tt.unit {
				t.Errorf("expected %s, got %s", tt.unit, unit)
			}
		})
	}
}

func TestUnitForType_UnknownType(t *testing.T) {
	_, err := unitForType(SensorType("smoke"))
	if err == nil {
		t.Errorf("expected error for unknown sensor type")
	}
}

func TestRun_ReadingsInRange(t *testing.T) {
	s, err := NewSensor("sensor-1", "living-room", Temperature, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := make(chan Reading, 10)
	go s.Run(ctx, out)

	min, max := sensorRangeByType(Temperature)

	for i := 0; i < 5; i++ {
		r := <-out
		if r.Value < min || r.Value > max {
			t.Errorf("value %v out of range [%v, %v]", r.Value, min, max)
		}
		if r.Unit != "°C" {
			t.Errorf("expected unit °C, got %s", r.Unit)
		}
		if r.SensorID != "sensor-1" {
			t.Errorf("expected sensor-1, got %s", r.SensorID)
		}
		if r.Type != Temperature {
			t.Errorf("expected Temperature, got %s", r.Type)
		}

	}
}
