package repository

import (
	"context"
	"errors"
	"fmt"

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

// GetByDate looks up the problem assigned to an IST calendar date, passed as
// a YYYY-MM-DD string so the comparison never depends on the database session
// timezone.
func (r *DailyProblemRepository) GetByDate(ctx context.Context, date string) (*model.DailyProblem, error) {
	query := `SELECT id,assigned_date,contest_id,problem_index,name,rating,url,tags FROM daily_problems WHERE assigned_date = $1::date`

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
		&problem.Tags,
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

func (r *DailyProblemRepository) Create(ctx context.Context, problem codeforces.Problem, date string) (*model.DailyProblem, error) {
	query := `INSERT INTO daily_problems (assigned_date, contest_id, problem_index,name,rating,url,tags) VALUES ($1::date,$2,$3,$4,$5,$6,$7) ON CONFLICT (assigned_date) DO NOTHING RETURNING id,assigned_date, contest_id, problem_index,name,rating,url,tags`

	var dailyproblem model.DailyProblem

	err := r.db.QueryRow(
		ctx,
		query,
		date,
		problem.ContestId,
		problem.Index,
		problem.Name,
		problem.Rating,
		problem.URL(),
		problem.Tags,
	).Scan(
		&dailyproblem.ID,
		&dailyproblem.AssignedDate,
		&dailyproblem.ContestID,
		&dailyproblem.ProblemIndex,
		&dailyproblem.Name,
		&dailyproblem.Rating,
		&dailyproblem.URL,
		&dailyproblem.Tags,
	)

	if err == nil {
		return &dailyproblem, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		exisitngProblem, err := r.GetByDate(ctx, date)

		if err != nil {
			return nil, fmt.Errorf("getting existing daily problem after conflict: %w", err)
		}

		if exisitngProblem == nil {
			return nil, fmt.Errorf("daily problem disappeared after conflict")
		}

		return exisitngProblem, nil
	}

	return nil, fmt.Errorf("creating daily problem: %w", err)
}

// GetAssignedKeys returns the set of problems that have already been handed
// out, keyed the same way codeforces.Problem.Key does, so a new assignment can
// avoid repeating one.
func (r *DailyProblemRepository) GetAssignedKeys(ctx context.Context) (map[string]bool, error) {
	query := `SELECT contest_id, problem_index FROM daily_problems`

	rows, err := r.db.Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("getting assigned problems: %w", err)
	}

	defer rows.Close()

	keys := make(map[string]bool)

	for rows.Next() {
		var contestID int
		var problemIndex string

		if err := rows.Scan(&contestID, &problemIndex); err != nil {
			return nil, fmt.Errorf("scanning assigned problem: %w", err)
		}

		keys[fmt.Sprintf("%d%s", contestID, problemIndex)] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating assigned problems: %w", err)
	}

	return keys, nil
}

func (r *DailyProblemRepository) GetByID(ctx context.Context, id int64) (*model.DailyProblem, error) {
	query := `SELECT id, assigned_date,contest_id,problem_index,name,rating,url,tags FROM daily_problems WHERE id = $1`

	var problem model.DailyProblem

	err := r.db.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&problem.ID,
		&problem.AssignedDate,
		&problem.ContestID,
		&problem.ProblemIndex,
		&problem.Name,
		&problem.Rating,
		&problem.URL,
		&problem.Tags,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"getting daily problem by id: %w",
			err,
		)
	}

	return &problem, nil
}
