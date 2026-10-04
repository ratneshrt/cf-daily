package codeforces

import "fmt"

type Problem struct {
	ContestId int      `json:"contestId"`
	Index     string   `json:"index"`
	Name      string   `json:"name"`
	Rating    int      `json:"rating"`
	Tags      []string `json:"tags"`
}

// Key identifies a problem uniquely within the problemset.
func (p Problem) Key() string {
	return fmt.Sprintf("%d%s", p.ContestId, p.Index)
}

// URL returns the problemset form of the problem URL. The
// /contest/{id}/problem/{index} form returns 404 for problems whose contest is
// not publicly viewable, while the problemset form works for every entry.
func (p Problem) URL() string {
	return fmt.Sprintf(
		"https://codeforces.com/problemset/problem/%d/%s",
		p.ContestId,
		p.Index,
	)
}

type ProblemSetResult struct {
	Problems []Problem `json:"problems"`
}

type APIResponse struct {
	Status  string           `json:"status"`
	Comment string           `json:"comment"`
	Result  ProblemSetResult `json:"result"`
}
