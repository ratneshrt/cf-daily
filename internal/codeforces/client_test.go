package codeforces

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const sampleResponse = `{
  "status": "OK",
  "result": {
    "problems": [
      {"contestId": 4, "index": "A", "name": "Watermelon", "rating": 800, "tags": ["brute force", "math"]},
      {"contestId": 71, "index": "A", "name": "Way Too Long Words", "rating": 800, "tags": ["strings"]},
      {"contestId": 1, "index": "B", "name": "Spreadsheets", "rating": 1600, "tags": ["math"]},
      {"index": "Z", "name": "No Contest Id", "rating": 800, "tags": []},
      {"contestId": 99, "index": "C", "name": "Unrated", "tags": []}
    ]
  }
}`

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)

	t.Cleanup(server.Close)

	client := NewClient()
	client.baseURL = server.URL

	return client
}

func TestGetProblems(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleResponse))
	})

	problems, err := client.GetProblems(context.Background())
	if err != nil {
		t.Fatalf("failed to get problems: %v", err)
	}

	if len(problems) != 5 {
		t.Fatalf("expected 5 problems, got %d", len(problems))
	}

	if problems[0].Name != "Watermelon" {
		t.Errorf("unexpected first problem: %q", problems[0].Name)
	}
}

func TestGetProblemsIsCached(t *testing.T) {
	calls := 0

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(sampleResponse))
	})

	for i := 0; i < 3; i++ {
		if _, err := client.GetProblems(context.Background()); err != nil {
			t.Fatalf("call %d failed: %v", i, err)
		}
	}

	if calls != 1 {
		t.Fatalf("expected 1 upstream call, got %d", calls)
	}
}

func TestGetProblemsServesStaleCacheOnFailure(t *testing.T) {
	fail := false

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		_, _ = w.Write([]byte(sampleResponse))
	})

	if _, err := client.GetProblems(context.Background()); err != nil {
		t.Fatalf("warming cache failed: %v", err)
	}

	// Expire the cache, then make the upstream fail.
	client.cachedAt = client.cachedAt.Add(-2 * cacheTTL)
	fail = true

	problems, err := client.GetProblems(context.Background())
	if err != nil {
		t.Fatalf("expected stale cache, got error: %v", err)
	}

	if len(problems) != 5 {
		t.Fatalf("expected 5 stale problems, got %d", len(problems))
	}
}

func TestGetProblemsReturnsErrorWithoutCache(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	if _, err := client.GetProblems(context.Background()); err == nil {
		t.Fatal("expected an error when nothing is cached")
	}
}

func TestGetRandomProblemFiltersRatingAndUnusable(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleResponse))
	})

	service := NewService(client)

	for i := 0; i < 20; i++ {
		problem, err := service.GetRandomProblem(context.Background(), 800, 800, nil)
		if err != nil {
			t.Fatalf("getting random problem: %v", err)
		}

		if problem.Rating != 800 {
			t.Fatalf("rating %d is outside the requested range", problem.Rating)
		}

		if problem.ContestId == 0 {
			t.Fatal("picked a problem with no contest id")
		}
	}
}

func TestGetRandomProblemExcludesUsedProblems(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleResponse))
	})

	service := NewService(client)

	exclude := map[string]bool{"4A": true}

	for i := 0; i < 20; i++ {
		problem, err := service.GetRandomProblem(context.Background(), 800, 800, exclude)
		if err != nil {
			t.Fatalf("getting random problem: %v", err)
		}

		if problem.Key() != "71A" {
			t.Fatalf("expected the only unused 800 problem, got %q", problem.Key())
		}
	}
}

func TestGetRandomProblemAllowsRepeatWhenAllUsed(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleResponse))
	})

	service := NewService(client)

	exclude := map[string]bool{"4A": true, "71A": true}

	problem, err := service.GetRandomProblem(context.Background(), 800, 800, exclude)
	if err != nil {
		t.Fatalf("expected a repeat rather than an error: %v", err)
	}

	if problem.Rating != 800 {
		t.Fatalf("unexpected problem %q", problem.Key())
	}
}

func TestGetRandomProblemErrorsWhenRangeIsEmpty(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleResponse))
	})

	service := NewService(client)

	if _, err := service.GetRandomProblem(context.Background(), 3500, 3600, nil); err == nil {
		t.Fatal("expected an error for an empty rating range")
	}
}

func TestProblemURLUsesProblemsetForm(t *testing.T) {
	problem := Problem{ContestId: 4, Index: "A"}

	want := "https://codeforces.com/problemset/problem/4/A"

	if got := problem.URL(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
