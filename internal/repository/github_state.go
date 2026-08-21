package repository

import (
	"context"
	"fmt"
	"time"

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

	query := `
		INSERT INTO github_connection_states (
			state,
			telegram_user_id,
			expires_at
		)
		VALUES ($1, $2, $3)
	`

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
