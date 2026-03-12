package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port          int
	HealthPort    int    // HTTP health endpoint (0 = disabled)
	DataDir       string
	HostKeyDir    string
	AdminKey      string // path to admin public key file
	ResendAPIKey  string // Resend API key for email notifications
	ResendFrom    string // From address for emails (empty = Resend default)
}

func Load() Config {
	dataDir := envOr("BBS_DATA_DIR", "data")
	port := 2222
	if p, err := strconv.Atoi(os.Getenv("HUB_PORT")); err == nil && p > 0 {
		port = p
	}
	healthPort := 0
	if p, err := strconv.Atoi(os.Getenv("HEALTH_PORT")); err == nil && p > 0 {
		healthPort = p
	}
	return Config{
		Port:         port,
		HealthPort:   healthPort,
		DataDir:      dataDir,
		HostKeyDir:   filepath.Join(dataDir),
		AdminKey:     os.Getenv("BBS_ADMIN_KEY"),
		ResendAPIKey: os.Getenv("RESEND_API_KEY"),
		ResendFrom:   os.Getenv("RESEND_FROM"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
