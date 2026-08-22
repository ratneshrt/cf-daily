package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ratneshrt/cf-daily/internal/model"
)

type TelegramUserRepository struct {
	db *pgxpool.Pool
}

func NewTelegramUserRepository(db *pgxpool.Pool) *TelegramUserRepository {
	return &TelegramUserRepository{
		db: db,
	}
}

func (r *TelegramUserRepository) GetByTelegramUserID(ctx context.Context, telegramUserID int64) (*model.TelegramUser, error) {
	query := `SELECT id, telegram_user_id, chat_id, username,first_name, active, created_at,updated_at FROM telegram_users WHERE telegram_user_id = $1`

	var user model.TelegramUser

	err := r.db.QueryRow(
		ctx,
		query,
		telegramUserID,
	).Scan(
		&user.ID,
		&user.TelegramUserID,
		&user.ChatID,
		&user.Username,
		&user.FirstName,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"getting telegram user: %w",
			err,
		)
	}

	return &user, nil
}

func (r *TelegramUserRepository) Create(ctx context.Context, userID int64, chatID int64, username string, firstName string) (*model.TelegramUser, error) {
	query := `INSERT INTO telegram_users (telegram_user_id, chat_id, username, first_name,active) VALUES ($1,$2,$3,$4,TRUE) RETURNING id, telegram_user_id,chat_id,username,first_name,active,created_at,updated_at`

	var user model.TelegramUser

	err := r.db.QueryRow(
		ctx,
		query,
		userID,
		chatID,
		username,
		firstName,
	).Scan(
		&user.ID,
		&user.TelegramUserID,
		&user.ChatID,
		&user.Username,
		&user.FirstName,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"creating telegram user: %w",
			err,
		)
	}

	return &user, nil
}

func (r *TelegramUserRepository) Activate(ctx context.Context, telegramUserID int64, chatID int64, username string, firstName string) (*model.TelegramUser, error) {
	query := `UPDATE telegram_users SET chat_id = $2, username = $3, first_name = $4, active = TRUE, updated_at = NOW() WHERE telegram_user_id = $1 RETURNING id, telegram_user_id, chat_id, username, first_name,active,created_at,updated_at`

	var user model.TelegramUser

	err := r.db.QueryRow(
		ctx,
		query,
		telegramUserID,
		chatID,
		username,
		firstName,
	).Scan(
		&user.ID,
		&user.TelegramUserID,
		&user.ChatID,
		&user.Username,
		&user.FirstName,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"activating telegram user: %w",
			err,
		)
	}

	return &user, nil
}

func (r *TelegramUserRepository) GetActiveUsers(ctx context.Context) ([]model.TelegramUser, error) {
	query := `SELECT id,telegram_user_id,chat_id,username,first_name,active,created_at,updated_at FROM telegram_users WHERE active = TRUE ORDER BY id`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf(
			"getting active telegram users: %w",
			err,
		)
	}

	defer rows.Close()

	users := make([]model.TelegramUser, 0)

	for rows.Next() {
		var user model.TelegramUser

		err := rows.Scan(
			&user.ID,
			&user.TelegramUserID,
			&user.ChatID,
			&user.Username,
			&user.FirstName,
			&user.Active,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"scanning telegram user: %w",
				err,
			)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterating telegram users: %w",
			err,
		)
	}

	return users, nil
}

func (r *TelegramUserRepository) ConnectGithub(ctx context.Context, telegramUserID int64, githubUserID int64, githubUsername string, githubInstallationID int64) error {
	query := `UPDATE telegram_users SET github_user_id = $1, github_username = $2, github_installation_id = $3, github_connected_at = NOW(), updated_at = NOW() WHERE telegram_user_id = $4`

	tag, err := r.db.Exec(
		ctx,
		query,
		githubUserID,
		githubUsername,
		githubInstallationID,
		telegramUserID,
	)

	if err != nil {
		return fmt.Errorf("connecting github account: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf(
			"telegram user %d not found",
			telegramUserID,
		)
	}

	return nil
}
