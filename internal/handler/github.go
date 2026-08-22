package handler

import (
	"fmt"
	"net/http"

	"github.com/ratneshrt/cf-daily/internal/repository"
	"github.com/ratneshrt/cf-daily/internal/service"
)

type GitHubHandler struct {
	githubService          *service.GitHubService
	githubStateRepository  *repository.GitHubStateRepository
	telegramUserRepository *repository.TelegramUserRepository
}

func NewGitHubHandler(githubservice *service.GitHubService, githubStateRepository *repository.GitHubStateRepository, telegramUserRepository *repository.TelegramUserRepository) *GitHubHandler {
	return &GitHubHandler{
		githubService:          githubservice,
		githubStateRepository:  githubStateRepository,
		telegramUserRepository: telegramUserRepository,
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

	telegramUserID, err := h.githubStateRepository.Get(ctx, state)

	if err != nil {
		http.Error(
			w,
			"invalid or expired connection",
			http.StatusBadRequest,
		)
		return
	}

	if err := h.githubStateRepository.Delete(ctx, state); err != nil {
		http.Error(
			w,
			"failed to consume connection state",
			http.StatusInternalServerError,
		)
		return
	}

	accessToken, err := h.githubService.ExchangeCode(ctx, code)

	if err != nil {
		http.Error(
			w,
			"failed to authorize GitHub",
			http.StatusInternalServerError,
		)
		return
	}

	githubUser, err := h.githubService.GetAuthenticatedUser(
		ctx,
		accessToken,
	)

	if err != nil {
		http.Error(
			w,
			"failed to get github user",
			http.StatusInternalServerError,
		)
		return
	}

	installationID, err := h.githubService.GetUserInstallation(ctx, githubUser.Login)

	if err != nil {
		http.Error(
			w,
			"8pieces installation not found",
			http.StatusBadRequest,
		)
		return
	}

	if err := h.telegramUserRepository.ConnectGithub(
		ctx,
		telegramUserID,
		githubUser.ID,
		githubUser.Login,
		installationID,
	); err != nil {
		http.Error(
			w,
			"failed to save github connection",
			http.StatusInternalServerError,
		)
		return
	}

	if err := h.githubService.CreateRepository(ctx, installationID); err != nil {
		http.Error(
			w,
			"Github connected, but failed to create flux-cf repository",
			http.StatusInternalServerError,
		)
		return
	}

	fmt.Fprintf(
		w,
		"✅ GitHub connected successfully!\n\n"+
			"GitHub: @%s\n"+
			"Installation ID: %d\n\n"+
			"You can close this page.",
		githubUser.Login,
		installationID,
	)
}
