package codeforces

type Problem struct {
	ContestId int      `json:"contestId"`
	Index     string   `json:"index"`
	Name      string   `json:"name"`
	Raitng    int      `json:"rating"`
	Tags      []string `json:"tags"`
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
