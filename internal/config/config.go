package config

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                   string
	LogLevel               slog.Level
	DatabaseURL            string
	MinRating              int
	MaxRating              int
	TelegramBotToken       string
	TelegramWebhookSecret  string
	CronSecret             string
	TelegramAllowedUserIDs []int64
	GitHubAppID            int64
	GitHubAppSlug          string
	GitHubClientID         string
	GitHubClientSecret     string
	GitHubPrivateKey       string
	GitHubCallbackURL      string
	GitHubRepositoryName   string
}

func Load() (Config, error) {

	githubAppIDString, err := requireEnv("FLUX_APP_ID")
	if err != nil {
		return Config{}, err
	}

	githubAppID, err := strconv.ParseInt(
		githubAppIDString,
		10,
		64,
	)

	if err != nil {
		return Config{}, fmt.Errorf("invalid FLUX_APP_ID: %w", err)
	}

	githubClientID, err := requireEnv("FLUX_CLIENT_ID")
	if err != nil {
		return Config{}, err
	}

	gitHubClientSecret, err := requireEnv("FLUX_CLIENT_SECRET")
	if err != nil {
		return Config{}, err
	}

	githubCallbackURL, err := requireEnv("FLUX_CALLBACK_URL")
	if err != nil {
		return Config{}, err
	}

	githubRepositoryName, err := requireEnv("FLUX_REPOSITORY")
	if err != nil {
		return Config{}, err
	}

	privateKeyB64, err := requireEnv("FLUX_PRIVATE_KEY_B64")
	if err != nil {
		return Config{}, err
	}

	keyBytes, err := base64.StdEncoding.DecodeString(
		privateKeyB64,
	)

	if err != nil {
		return Config{}, fmt.Errorf(
			"decoding FLUX_PRIVATE_KEY_B64: %w",
			err,
		)
	}

	privateKey := string(keyBytes)

	// These guard the webhook, the cron endpoints and the database. An empty
	// value would make the header comparisons succeed for a request that sends
	// no header at all, so they are required rather than optional.
	databaseURL, err := requireEnv("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	telegramBotToken, err := requireEnv("TELEGRAM_BOT_TOKEN")
	if err != nil {
		return Config{}, err
	}

	telegramWebhookSecret, err := requireEnv("TELEGRAM_WEBHOOK_SECRET")
	if err != nil {
		return Config{}, err
	}

	cronSecret, err := requireEnv("CRON_SECRET")
	if err != nil {
		return Config{}, err
	}

	minRating, err := strconv.Atoi(os.Getenv("MIN_RATING"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid MIN_RATING: %w", err)
	}

	maxRating, err := strconv.Atoi(os.Getenv("MAX_RATING"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid MAX_RATING: %w", err)
	}

	if minRating > maxRating {
		return Config{}, fmt.Errorf(
			"MIN_RATING (%d) is greater than MAX_RATING (%d)",
			minRating,
			maxRating,
		)
	}

	telegramAllowedUserIDs, err := parseTelegramAllowedUserIDs(
		os.Getenv("TELEGRAM_ALLOWED_USER_IDS"),
	)

	if err != nil {
		return Config{}, err
	}

	logLevel, err := parseLogLevel(os.Getenv("LOG_LEVEL"))

	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:                   getEnv("PORT", "8080"),
		LogLevel:               logLevel,
		DatabaseURL:            databaseURL,
		MinRating:              minRating,
		MaxRating:              maxRating,
		TelegramBotToken:       telegramBotToken,
		TelegramWebhookSecret:  telegramWebhookSecret,
		CronSecret:             cronSecret,
		TelegramAllowedUserIDs: telegramAllowedUserIDs,
		GitHubAppID:            githubAppID,
		GitHubAppSlug:          getEnv("FLUX_APP_SLUG", "8pieces"),
		GitHubClientID:         githubClientID,
		GitHubClientSecret:     gitHubClientSecret,
		GitHubCallbackURL:      githubCallbackURL,
		GitHubRepositoryName:   githubRepositoryName,
		GitHubPrivateKey:       privateKey,
	}, nil
}

func requireEnv(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}

	return value, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {
		return fallback
	}

	return value
}

func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid LOG_LEVEL %q", value)
	}
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
