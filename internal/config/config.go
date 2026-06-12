package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL      string
	NATSUrl          string
	HTTPPort         string
	JaegerEndpoint   string
	JWKSURL          string
	Issuer           string
	Audience         string
	SimulatorEnabled bool
	SensorInterval   time.Duration
}

func Load() *Config {
	return &Config{
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/warden_gateway_db"),
		NATSUrl:          getEnv("NATS_URL", "nats://localhost:4222"),
		HTTPPort:         getEnv("HTTP_PORT", ":8080"),
		JaegerEndpoint:   getEnv("JAEGER_ENDPOINT", "localhost:4318"),
		JWKSURL:          getEnv("JWKS_URL", "http://localhost:8082/.well-known/jwks.json"),
		Issuer:           getEnv("ISSUER", "warden-auth"),
		Audience:         getEnv("AUDIENCE", "warden-gateway"),
		SimulatorEnabled: getEnvBool("SIMULATOR_ENABLED", true),
		SensorInterval:   getEnvDuration("SENSOR_INTERVAL", 500*time.Millisecond),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
