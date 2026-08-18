package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ratneshrt/cf-daily/internal/codeforces"
	"github.com/ratneshrt/cf-daily/internal/model"
)

type DailyProblemRepository struct {
	db *pgxpool.Pool
}

func NewDailyProblemRepository(db *pgxpool.Pool) *DailyProblemRepository {
	return &DailyProblemRepository{
		db: db,
	}
}

func (r *DailyProblemRepository) GetByDate(ctx context.Context, date time.Time) (*model.DailyProblem, error) {
	query := `SELECT id,assigned_date,contest_id,problem_index,name,rating,url FROM daily_problems WHERE assigned_date = $1`

	var problem model.DailyProblem

	err := r.db.QueryRow(
		ctx,
		query,
		date,
	).Scan(
		&problem.ID,
		&problem.AssignedDate,
		&problem.ContestID,
		&problem.ProblemIndex,
		&problem.Name,
		&problem.Rating,
		&problem.URL,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"getting daily problem: %w",
			err,
		)
	}

	return &problem, nil
}

func (r *DailyProblemRepository) Create(ctx context.Context, problem codeforces.Problem, date time.Time) (*model.DailyProblem, error) {
	query := `INSERT INTO daily_problems (assigned_date, contest_id, problem_index,name,rating,url) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id,assigned_date, contest_id, problem_index,name,rating,url`

	var dailyproblem model.DailyProblem

	err := r.db.QueryRow(
		ctx,
		query,
		problem.ContestId,
		problem.Index,
		problem.Name,
		problem.Rating,
		problem.URL(),
	).Scan(
		&dailyproblem.ID,
		&dailyproblem.AssignedDate,
		&dailyproblem.ContestID,
		&dailyproblem.ProblemIndex,
		&dailyproblem.Name,
		&dailyproblem.Rating,
		&dailyproblem.URL,
	)

	if err != nil {
		return nil, fmt.Errorf("creating daily problem: %w", err)
	}

	return &dailyproblem, nil
}
