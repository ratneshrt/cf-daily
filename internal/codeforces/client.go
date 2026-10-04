package codeforces

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const defaultProblemsetURL = "https://codeforces.com/api/problemset.problems"

// cacheTTL is how long a fetched problemset stays usable. The problemset only
// changes when a new contest is added, so a few hours is plenty.
const cacheTTL = 6 * time.Hour

type Client struct {
	httpClient *http.Client
	baseURL    string

	// mu also serializes fetches, which keeps us within the Codeforces rate
	// limit of roughly one call every two seconds per IP.
	mu       sync.Mutex
	cached   []Problem
	cachedAt time.Time
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: defaultProblemsetURL,
	}
}

// GetProblems returns the Codeforces problemset, cached for cacheTTL. The
// response is several megabytes, so it must not be fetched per request. If a
// refresh fails but a previous response is cached, the stale copy is served
// rather than failing the caller.
func (c *Client) GetProblems(ctx context.Context) ([]Problem, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached != nil && time.Since(c.cachedAt) < cacheTTL {
		return c.cached, nil
	}

	problems, err := c.fetchProblems(ctx)

	if err != nil {
		if c.cached != nil {
			slog.Warn(
				"codeforces fetch failed, serving stale problemset",
				"cached_at",
				c.cachedAt,
				"error",
				err,
			)

			return c.cached, nil
		}

		return nil, err
	}

	c.cached = problems
	c.cachedAt = time.Now()

	return problems, nil
}

func (c *Client) fetchProblems(ctx context.Context) ([]Problem, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("creating cf req: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling codeforces api: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"codeforces API returned status %d",
			resp.StatusCode,
		)
	}

	var result APIResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf(
			"decoding Codeforces response: %w",
			err,
		)
	}

	if result.Status != "OK" {
		return nil, fmt.Errorf(
			"codeforces API error: %s",
			result.Comment,
		)
	}

	return result.Result.Problems, nil
}
