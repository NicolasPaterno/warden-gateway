package config

import (
	"os"
)

type Config struct {
	DatabaseURL string
	NATSUrl     string
	HTTPPort    string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/warden_gateway_db"),
		NATSUrl:     getEnv("NATS_URL", "nats://localhost:4222"),
		HTTPPort:    getEnv("HTTP_PORT", ":8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
