package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port          string
	DatabaseURL   string
	MinRating     int
	MaxRating     int
	TelegramToken string
}

func Load() (Config, error) {
	minRating, err := strconv.Atoi(os.Getenv("MIN_RATING"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid MIN_RATING: %w", err)
	}

	maxRating, err := strconv.Atoi(os.Getenv("MAX_RATING"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid MAX_RATING: %w", err)
	}

	return Config{
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		MinRating:     minRating,
		MaxRating:     maxRating,
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
	}, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
