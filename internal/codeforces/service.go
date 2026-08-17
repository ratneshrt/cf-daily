package codeforces

import (
	"context"
	"fmt"
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

func (s *Service) GetRandomProblem(ctx context.Context, minRating, maxRating int) (Problem, error) {
	problems, err := s.client.GetProblems(ctx)
	if err != nil {
		return Problem{}, err
	}

	var filtered []Problem

	for _, problem := range problems {
		if problem.Raitng >= minRating && problem.Raitng <= maxRating {
			filtered = append(filtered, problem)
		}
	}

	if len(filtered) == 0 {
		return Problem{}, fmt.Errorf(
			"no problems found between ratings %d and %d",
			minRating,
			maxRating,
		)
	}

	randomIndex := rand.Intn(len(filtered))

	return filtered[randomIndex], nil
}
