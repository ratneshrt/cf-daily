package codeforces

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
)

type Service struct {
	client *Client
}

func NewService(client *Client) *Service {
	return &Service{
		client: client,
	}
}

// GetRandomProblem picks a problem inside the rating range. Problems whose Key
// is present in exclude are avoided, which is how the same problem stops being
// handed out twice; exclude may be nil. If every candidate has been used
// before, a repeat is allowed rather than failing.
func (s *Service) GetRandomProblem(ctx context.Context, minRating, maxRating int, exclude map[string]bool) (Problem, error) {
	problems, err := s.client.GetProblems(ctx)
	if err != nil {
		return Problem{}, err
	}

	var filtered []Problem
	var alreadyUsed []Problem

	for _, problem := range problems {
		// Without a contest id or a rating there is no usable problem URL.
		if problem.ContestId == 0 || problem.Rating == 0 {
			continue
		}

		if problem.Rating < minRating || problem.Rating > maxRating {
			continue
		}

		if exclude[problem.Key()] {
			alreadyUsed = append(alreadyUsed, problem)
			continue
		}

		filtered = append(filtered, problem)
	}

	if len(filtered) == 0 {
		if len(alreadyUsed) == 0 {
			return Problem{}, fmt.Errorf(
				"no problems found between ratings %d and %d",
				minRating,
				maxRating,
			)
		}

		slog.Warn(
			"every problem in range has been assigned before, allowing a repeat",
			"min_rating",
			minRating,
			"max_rating",
			maxRating,
			"candidates",
			len(alreadyUsed),
		)

		filtered = alreadyUsed
	}

	return filtered[rand.Intn(len(filtered))], nil
}
