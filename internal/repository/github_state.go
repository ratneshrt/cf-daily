package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GitHubStateRepository struct {
	db *pgxpool.Pool
}

func NewGitHubStateRepository(
	db *pgxpool.Pool,
) *GitHubStateRepository {
	return &GitHubStateRepository{
		db: db,
	}
}

func (r *GitHubStateRepository) Create(
	ctx context.Context,
	state string,
	telegramUserID int64,
	expiresAt time.Time,
) error {

	// Expired states are never consumed, so clear them out here rather than
	// running a separate reaper.
	if _, err := r.db.Exec(
		ctx,
		`DELETE FROM github_connection_states WHERE expires_at <= NOW()`,
	); err != nil {
		slog.Warn("failed to clean up expired github connection states", "error", err)
	}

	query := `INSERT INTO github_connection_states (state,telegram_user_id,expires_at) VALUES ($1, $2, $3)`

	_, err := r.db.Exec(
		ctx,
		query,
		state,
		telegramUserID,
		expiresAt,
	)

	if err != nil {
		return fmt.Errorf(
			"creating github connection state: %w",
			err,
		)
	}

	return nil
}

func (r *GitHubStateRepository) Consume(ctx context.Context, state string) (int64, error) {
	query := `DELETE FROM github_connection_states WHERE state = $1 AND expires_at > NOW() RETURNING telegram_user_id`

	var telegramUserID int64

	err := r.db.QueryRow(
		ctx,
		query,
		state,
	).Scan(&telegramUserID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("github connection state not found, expired or already consumed")
		}
		return 0, fmt.Errorf(
			"consuming github connection state: %w",
			err,
		)
	}

	return telegramUserID, nil
}
