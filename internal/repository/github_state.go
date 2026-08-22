package repository

import (
	"context"
	"errors"
	"fmt"
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

func (r *GitHubStateRepository) Get(ctx context.Context, state string) (int64, error) {
	query := `SELECT telegram_user_id FROM github_connection_states WHERE state = $1 AND expires_at > NOW()`

	var telegramUserID int64

	err := r.db.QueryRow(
		ctx,
		query,
		state,
	).Scan(&telegramUserID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("invalid or expires github state")
		}

		return 0, fmt.Errorf("getting github connection state: %w", err)
	}

	return telegramUserID, nil
}

func (r *GitHubStateRepository) Delete(ctx context.Context, state string) error {

	query := `DELETE FROM github_connection_states WHERE state = $1`

	tag, err := r.db.Exec(
		ctx,
		query,
		state,
	)

	if err != nil {
		return fmt.Errorf(
			"deleting github connection state: %w",
			err,
		)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf(
			"github connection state was already consumed or does not exist",
		)
	}

	return nil
}
