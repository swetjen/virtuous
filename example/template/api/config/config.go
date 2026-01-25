package config

import (
	"os"
	"strings"
)

type Config struct {
	Port             string
	AdminBearerToken string
	AllowedOrigins   []string
}

func Load() Config {
	return Config{
		Port:             getEnv("PORT", "8000"),
		AdminBearerToken: getEnv("ADMIN_BEARER_TOKEN", "dev-admin-token"),
		AllowedOrigins:   splitEnvList("CORS_ALLOW_ORIGINS", []string{"*"}),
	}
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitEnvList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
