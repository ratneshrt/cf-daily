package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ratneshrt/cf-daily/internal/codeforces"
	"github.com/ratneshrt/cf-daily/internal/model"
	"github.com/ratneshrt/cf-daily/internal/repository"
)

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

func (s *DailyProblemService) GetToday(ctx context.Context) (*model.DailyProblem, error) {
	now := time.Now().In(s.location)

	today := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0, 0, 0, 0,
		s.location,
	)

	problem, err := s.repository.GetByDate(ctx, today)

	if err != nil {
		return nil, fmt.Errorf("checking today's problem: %w", err)
	}

	if problem != nil {
		return problem, nil
	}

	cfProblem, err := s.codeforces.GetRandomProblem(ctx, s.minRating, s.maxRating)

	if err != nil {
		return nil, fmt.Errorf("generating daily problem: %w", err)
	}

	problem, err = s.repository.Create(ctx, cfProblem, today)

	if err != nil {
		return nil, fmt.Errorf("saving daily problem: %w", err)
	}

	return problem, nil
}
