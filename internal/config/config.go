package config

import (
	"os"
)

type Config struct {
	DatabaseURL    string
	NATSUrl        string
	HTTPPort       string
	JaegerEndpoint string
	JWKSURL        string
	Issuer         string
	Audience       string
}

func Load() *Config {
	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/warden_gateway_db"),
		NATSUrl:        getEnv("NATS_URL", "nats://localhost:4222"),
		HTTPPort:       getEnv("HTTP_PORT", ":8080"),
		JaegerEndpoint: getEnv("JAEGER_ENDPOINT", "localhost:4318"),
		JWKSURL:        getEnv("JWKS_URL", "http://localhost:8082/.well-known/jwks.json"),
		Issuer:         getEnv("ISSUER", "warden-auth"),
		Audience:       getEnv("AUDIENCE", "warden-gateway"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
