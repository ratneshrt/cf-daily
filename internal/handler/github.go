package handler

import (
	"fmt"
	"net/http"

	"github.com/ratneshrt/cf-daily/internal/service"
)

type GitHubHandler struct {
	githubService *service.GitHubService
}

func NewGitHubHandler(githubservice *service.GitHubService) *GitHubHandler {
	return &GitHubHandler{
		githubService: githubservice,
	}
}

func (h *GitHubHandler) Callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		http.Error(w, "missing code or state", http.StatusBadRequest)
		return
	}

	fmt.Fprintln(w, "Gtihub callback received. Connection flow is being completed.")

	_ = ctx
}
