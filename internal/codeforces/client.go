package codeforces

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const problemsetURL = "https://codeforces.com/api/problemset.problems"

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
	}
}

func (c *Client) GetProblems(ctx context.Context) ([]Problem, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		problemsetURL,
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
			"Codeforces API returned status %d",
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
			"Codeforces API error: %s",
			result.Comment,
		)
	}

	return result.Result.Problems, nil
}
