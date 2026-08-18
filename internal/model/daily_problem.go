package model

import "time"

type DailyProblem struct {
	ID           int64
	AssignedDate time.Time
	ContestID    int
	ProblemIndex string
	Name         string
	Rating       int
	URL          string
}
