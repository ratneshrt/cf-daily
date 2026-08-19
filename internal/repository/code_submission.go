package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ratneshrt/cf-daily/internal/model"
)

type CodeSubmissionRepository struct {
	db *pgxpool.Pool
}

func NewCodeSubmissionRepository(db *pgxpool.Pool) *CodeSubmissionRepository {
	return &CodeSubmissionRepository{db: db}
}

func (r *CodeSubmissionRepository) Get(ctx context.Context, telegramUserID int64, dailyProblemID int64) (*model.CodeSubmission, error) {
	query := `SELECT id,telegram_user_id,daily_problem_id,code,language,created_at,updated_at FROM code_submissions WHERE telegram_user_id = $1 AND daily_problem_id = $2`

	var submission model.CodeSubmission

	err := r.db.QueryRow(
		ctx,
		query,
		telegramUserID,
		dailyProblemID,
	).Scan(
		&submission.ID,
		&submission.TelegramUserID,
		&submission.DailyProblemID,
		&submission.Code,
		&submission.Language,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"getting code submission: %w",
			err,
		)
	}

	return &submission, nil
}

func (r *CodeSubmissionRepository) Create(ctx context.Context, telegramUserID int64, dailyProblemID int64, code string) (*model.CodeSubmission, error) {
	query := `INSERT INTO code_submissions (telegram_user_id, daily_problem_id, code) VALUES ($1,$2,$3) RETURNING id, telegram_user_id,daily_problem_id,code,language,created_at,updated_at`

	var submission model.CodeSubmission

	err := r.db.QueryRow(
		ctx,
		query,
		telegramUserID,
		dailyProblemID,
		code,
	).Scan(
		&submission.ID,
		&submission.TelegramUserID,
		&submission.DailyProblemID,
		&submission.Code,
		&submission.Language,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"creating code submission: %w",
			err,
		)
	}

	return &submission, nil
}

func (r *CodeSubmissionRepository) Update(ctx context.Context, telegramUserID int64, dailyProblemID int64, code string) (*model.CodeSubmission, error) {
	query := `UPDATE code_submissions SET code = $3, updated_at = NOW() WHERE telegram_user_id = $1 AND daily_problem_id = $2 RETURNING id,telegram_user_id,daily_problem_id,code,language,created_at,updated_at`

	var submission model.CodeSubmission

	err := r.db.QueryRow(
		ctx,
		query,
		telegramUserID,
		dailyProblemID,
		code,
	).Scan(
		&submission.ID,
		&submission.TelegramUserID,
		&submission.DailyProblemID,
		&submission.Code,
		&submission.Language,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"updating code submission: %w",
			err,
		)
	}

	return &submission, nil
}

func (r *CodeSubmissionRepository) Delete(ctx context.Context, telegramUserID int64, dailyProblemID int64) error {
	query := `DELETE FROM code_submissions WHERE telegram_user_id = $1 AND daily_problem_id = $2`

	_, err := r.db.Exec(
		ctx,
		query,
		telegramUserID,
		dailyProblemID,
	)

	if err != nil {
		return fmt.Errorf(
			"deleting code submission: %w",
			err,
		)
	}

	return nil
}
