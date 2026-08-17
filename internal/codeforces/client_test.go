package codeforces

import (
	"context"
	"testing"
)

func TestGetProblem(t *testing.T) {
	client := NewClient()

	problems, err := client.GetProblems(context.Background())
	if err != nil {
		t.Fatalf("failed to get problems: %v", err)
	}

	if len(problems) == 0 {
		t.Fatal("expected problems, got zero")
	}

	t.Logf("received %d problems", len(problems))
}
