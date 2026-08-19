package model

import "time"

type CodeSubmission struct {
	ID             int64
	TelegramUserID int64
	DailyProblemID int64
	Code           string
	Language       *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
