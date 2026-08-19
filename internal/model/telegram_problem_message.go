package model

import "time"

type TelegramProblemMessage struct {
	ID                int64
	TelegramUserID    int64
	DailyProblemID    int64
	TelegramMessageID int64
	SentAt            time.Time
}
