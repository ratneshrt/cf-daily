package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ratneshrt/cf-daily/internal/model"
)

type TelegramProblemMessageRepository struct {
	db *pgxpool.Pool
}

func NewTelegramProblemMessageRepository(db *pgxpool.Pool) *TelegramProblemMessageRepository {
	return &TelegramProblemMessageRepository{db: db}
}

func (r *TelegramProblemMessageRepository) Create(ctx context.Context, telegramUserID int64, dailyProblemID int64, telegramMessageID int64) (*model.TelegramProblemMessage, error) {
	query := `INSERT INTO telegram_problem_messages (telegram_user_id, daily_problem_id, telegram_message_id) VALUES ($1,$2,$3) RETURNING id, telegram_user_id, daily_problem_id, telegram_message_id,sent_at`

	var message model.TelegramProblemMessage

	err := r.db.QueryRow(
		ctx,
		query,
		telegramUserID,
		dailyProblemID,
		telegramMessageID,
	).Scan(
		&message.ID,
		&message.TelegramUserID,
		&message.DailyProblemID,
		&message.TelegramMessageID,
		&message.SentAt,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"creating telegram problem message: %w",
			err,
		)
	}

	return &message, nil
}

func (r *TelegramProblemMessageRepository) GetByMessageID(ctx context.Context, telegramUserID int64, telegramMessageID int64) (*model.TelegramProblemMessage, error) {
	query := `SELECT id,telegram_user_id,daily_problem_id,telegram_message_id,sent_at FROM telegram_problem_messages WHERE telegram_user_id = $1 AND telegram_message_id = $2 LIMIT 1`

	var message model.TelegramProblemMessage

	err := r.db.QueryRow(
		ctx,
		query,
		telegramUserID,
		telegramMessageID,
	).Scan(
		&message.ID,
		&message.TelegramUserID,
		&message.DailyProblemID,
		&message.TelegramMessageID,
		&message.SentAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"getting telegram problem message: %w",
			err,
		)
	}

	return &message, nil
}

func (r *TelegramProblemMessageRepository) GetByUserAndProblem(ctx context.Context, telegramUserID int64, dailyProblemID int64) (*model.TelegramProblemMessage, error) {
	query := `SELECT id, telegram_user_id, daily_problem_id,telegram_message_id,sent_at FROM telegram_problem_messages WHERE telegram_user_id = $1 AND daily_problem_id = $2 LIMIT 1`

	var message model.TelegramProblemMessage

	err := r.db.QueryRow(
		ctx,
		query,
		telegramUserID,
		dailyProblemID,
	).Scan(
		&message.ID,
		&message.TelegramUserID,
		&message.DailyProblemID,
		&message.TelegramMessageID,
		&message.SentAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"getting telegram problem message: %w",
			err,
		)
	}

	return &message, nil
}
