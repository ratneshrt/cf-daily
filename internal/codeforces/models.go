package codeforces

import "fmt"

type Problem struct {
	ContestId int      `json:"contestId"`
	Index     string   `json:"index"`
	Name      string   `json:"name"`
	Rating    int      `json:"rating"`
	Tags      []string `json:"tags"`
}

func (p Problem) URL() string {
	return fmt.Sprintf(
		"https://codeforces.com/contest/%d/problem/%s", p.ContestId,
		p.Index,
	)
}

type ProblemSetResult struct {
	Problems   []Problem `json:"problems"`
	Statistics []struct {
		ContestID int    `json:"contestId"`
		Index     string `json:"index"`
		Solved    int    `json:"solvedCount"`
	} `json:"problemStatistics"`
}

type APIResponse struct {
	Status  string           `json:"status"`
	Comment string           `json:"comment"`
	Result  ProblemSetResult `json:"result"`
}
