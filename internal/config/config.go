package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Host       string
	Port       string
	DataDir    string
	DBPath     string
	WebDistDir string
	AppName    string
}

func Load() Config {
	dataDir := envOrDefault("HARBORX_DATA_DIR", "./data")
	return Config{
		Host:       envOrDefault("HARBORX_HOST", envOrDefault("MMWX_HOST", "0.0.0.0")),
		Port:       envOrDefault("HARBORX_PORT", envOrDefault("MMWX_PORT", "18080")),
		DataDir:    dataDir,
		DBPath:     envOrDefault("HARBORX_DB_PATH", envOrDefault("MMWX_DB_PATH", filepath.Join(dataDir, "harborx.sqlite"))),
		WebDistDir: envOrDefault("HARBORX_WEB_DIST_DIR", filepath.Join("web", "dist")),
		AppName:    "HarborX",
	}
}

func (c Config) ListenAddress() string {
	return c.Host + ":" + c.Port
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
