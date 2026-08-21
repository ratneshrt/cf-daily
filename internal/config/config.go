package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                   string
	DatabaseURL            string
	MinRating              int
	MaxRating              int
	TelegramBotToken       string
	TelegramWebhookSecret  string
	CronSecret             string
	TelegramAllowedUserIDs []int64
	GitHubAppID            int64
	GitHubClientID         string
	GitHubPrivateKey       string
	GitHubCallbackURL      string
	GitHubOwner            string
	GitHubRepositoryName   string
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

	telegramAllowedUserIDs, err := parseTelegramAllowedUserIDs(
		os.Getenv("TELEGRAM_ALLOWED_USER_IDS"),
	)

	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:                   getEnv("PORT", "8080"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		MinRating:              minRating,
		MaxRating:              maxRating,
		TelegramBotToken:       os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramWebhookSecret:  os.Getenv("TELEGRAM_WEBHOOK_SECRET"),
		CronSecret:             os.Getenv("CRON_SECRET"),
		TelegramAllowedUserIDs: telegramAllowedUserIDs,
	}, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}

func parseTelegramAllowedUserIDs(value string) ([]int64, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return []int64{}, nil
	}

	parts := strings.Split(
		value,
		",",
	)

	ids := make([]int64, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part == "" {
			continue
		}

		id, err := strconv.ParseInt(
			part,
			10,
			64,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"invalid TELEGRAM_ALLOWED_USER_IDS value %q: %w",
				part,
				err,
			)
		}

		ids = append(ids, id)
	}

	return ids, nil
}
