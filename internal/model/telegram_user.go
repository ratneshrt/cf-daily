package model

import "time"

type TelegramUser struct {
	ID                   int64
	TelegramUserID       int64
	ChatID               int64
	Username             string
	FirstName            string
	Active               bool
	GithubUserID         *int64
	GithubUsername       *string
	GithubInstallationID *int64
	GithubConnectedAt    *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
