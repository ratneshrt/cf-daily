package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ratneshrt/cf-daily/internal/codeforces"
	"github.com/ratneshrt/cf-daily/internal/model"
	"github.com/ratneshrt/cf-daily/internal/repository"
)

const dateLayout = "2006-01-02"

type DailyProblemService struct {
	repository *repository.DailyProblemRepository
	codeforces *codeforces.Service
	minRating  int
	maxRating  int
	location   *time.Location
}

func NewDailyProblemService(repository *repository.DailyProblemRepository, codeforces *codeforces.Service, minRating, maxRating int) *DailyProblemService {
	return &DailyProblemService{
		repository: repository,
		codeforces: codeforces,
		minRating:  minRating,
		maxRating:  maxRating,
		location:   time.FixedZone("IST", 5*60*60+30*60),
	}
}

// today returns the current IST calendar date as YYYY-MM-DD. It is passed to
// the database as a string so the stored date never depends on the database
// session timezone.
func (s *DailyProblemService) today() string {
	return time.Now().In(s.location).Format(dateLayout)
}

// GetToday reads today's problem. It never creates one: a nil problem means no
// problem has been assigned for today yet.
func (s *DailyProblemService) GetToday(ctx context.Context) (*model.DailyProblem, error) {
	problem, err := s.repository.GetByDate(ctx, s.today())

	if err != nil {
		return nil, fmt.Errorf("checking today's problem: %w", err)
	}

	return problem, nil
}

// EnsureToday returns today's problem, assigning a new one if today does not
// have a problem yet. Only the daily problem notification should call this.
func (s *DailyProblemService) EnsureToday(ctx context.Context) (*model.DailyProblem, error) {
	today := s.today()

	problem, err := s.repository.GetByDate(ctx, today)

	if err != nil {
		return nil, fmt.Errorf("checking today's problem: %w", err)
	}

	if problem != nil {
		return problem, nil
	}

	// Problems handed out before are excluded so the same one is not repeated.
	assigned, err := s.repository.GetAssignedKeys(ctx)

	if err != nil {
		slog.Warn(
			"could not load previously assigned problems, a repeat is possible",
			"error",
			err,
		)

		assigned = nil
	}

	cfProblem, err := s.codeforces.GetRandomProblem(ctx, s.minRating, s.maxRating, assigned)

	if err != nil {
		return nil, fmt.Errorf("generating daily problem: %w", err)
	}

	problem, err = s.repository.Create(ctx, cfProblem, today)

	if err != nil {
		return nil, fmt.Errorf("saving daily problem: %w", err)
	}

	slog.Info(
		"assigned daily problem",
		"date",
		today,
		"problem",
		cfProblem.Key(),
		"rating",
		cfProblem.Rating,
	)

	return problem, nil
}
