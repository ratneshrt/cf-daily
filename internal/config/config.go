package config

import (
	"encoding/base64"
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
	GitHubClientSecret     string
	GitHubPrivateKey       string
	GitHubCallbackURL      string
	GitHubOwner            string
	GitHubRepositoryName   string
}

func Load() (Config, error) {

	githubAppIDString := strings.TrimSpace(
		os.Getenv("FLUX_APP_ID"),
	)

	if githubAppIDString == "" {
		return Config{}, fmt.Errorf("FLUX_APP_ID is required")
	}

	githubAppID, err := strconv.ParseInt(
		githubAppIDString,
		10,
		64,
	)

	if err != nil {
		return Config{}, fmt.Errorf("invalid FLUX_APP_ID: %w", err)
	}

	githubClientID := strings.TrimSpace(
		os.Getenv("FLUX_CLIENT_ID"),
	)

	if githubClientID == "" {
		return Config{}, fmt.Errorf("FLUX_CLIENT_ID is required")
	}

	gitHubClientSecret := strings.TrimSpace(
		os.Getenv("FLUX_CLIENT_SECRET"),
	)

	if gitHubClientSecret == "" {
		return Config{}, fmt.Errorf(
			"FLUX_CLIENT_SECRET is required",
		)
	}

	githubCallbackURL := strings.TrimSpace(
		os.Getenv("FLUX_CALLBACK_URL"),
	)

	if githubCallbackURL == "" {
		return Config{}, fmt.Errorf(
			"FLUX_CALLBACK_URL is required",
		)
	}

	githubRepositoryName := strings.TrimSpace(
		os.Getenv("FLUX_REPOSITORY"),
	)

	if githubRepositoryName == "" {
		return Config{}, fmt.Errorf(
			"FLUX_REPOSITORY is required",
		)
	}

	privateKeyB64 := strings.TrimSpace(
		os.Getenv("FLUX_PRIVATE_KEY_B64"),
	)

	if privateKeyB64 == "" {
		return Config{}, fmt.Errorf(
			"FLUX_PRIVATE_KEY_B64 is required",
		)
	}

	keyBytes, err := base64.StdEncoding.DecodeString(
		privateKeyB64,
	)

	if err != nil {
		return Config{}, fmt.Errorf(
			"decoding flux private key: %w",
			err,
		)
	}

	privateKey := string(keyBytes)

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
		GitHubAppID:            githubAppID,
		GitHubClientID:         gitHubClientSecret,
		GitHubClientSecret:     gitHubClientSecret,
		GitHubOwner:            os.Getenv("FLUX_OWNER"),
		GitHubCallbackURL:      githubCallbackURL,
		GitHubRepositoryName:   githubRepositoryName,
		GitHubPrivateKey:       privateKey,
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
