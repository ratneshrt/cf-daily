package handler

import (
	"log/slog"
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
	installationIDParam := r.URL.Query().Get("installation_id")
	setupAction := r.URL.Query().Get("setup_action")

	slog.Info(
		"github callback received",
		"installation_id", installationIDParam,
		"setup_action", setupAction,
	)

	if code == "" || state == "" {
		slog.Warn("github callback missing code or state")

		http.Error(w, "missing code or state", http.StatusBadRequest)
		return
	}

	// Consumed up front: validating first and consuming later left the state
	// replayable whenever a later step failed.
	telegramUserID, err := h.githubStateRepository.Consume(ctx, state)

	if err != nil {
		slog.Warn("github state validation failed", "error", err)

		http.Error(
			w,
			"invalid or expired connection",
			http.StatusBadRequest,
		)
		return
	}

	slog.Info("github state consumed", "telegram_user_id", telegramUserID)

	accessToken, err := h.githubService.ExchangeCode(ctx, code)

	if err != nil {
		slog.Error("github oauth exchange failed", "error", err)

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
		slog.Error("github user lookup failed", "error", err)

		http.Error(
			w,
			"failed to get github user",
			http.StatusInternalServerError,
		)
		return
	}

	// installationOwner is the account the app is installed on, which owns the
	// repository. For a personal install it equals the user's login; for an
	// organisation install it does not.
	installationID, installationOwner, err := h.githubService.GetUserInstallation(ctx, githubUser.Login)

	if err != nil {
		slog.Error(
			"github installation lookup failed",
			"github_user", githubUser.Login,
			"error", err,
		)

		http.Error(
			w,
			"app installation not found",
			http.StatusBadRequest,
		)
		return
	}

	if installationOwner == "" {
		installationOwner = githubUser.Login
	}

	if err := h.telegramUserRepository.ConnectGithub(
		ctx,
		telegramUserID,
		githubUser.ID,
		installationOwner,
		installationID,
	); err != nil {
		slog.Error("saving github connection failed", "error", err)

		http.Error(
			w,
			"failed to save github connection",
			http.StatusInternalServerError,
		)
		return
	}

	repo, err := h.githubService.CreateRepository(ctx, accessToken)

	if err != nil {
		slog.Error(
			"github repository creation failed",
			"error", err,
		)

		http.Error(
			w,
			"GitHub connected, but the solutions repository could not be created",
			http.StatusInternalServerError,
		)
		return
	}

	slog.Info(
		"github connected",
		"github_user", githubUser.Login,
		"owner", installationOwner,
		"installation_id", installationID,
		"repository", repo.FullName,
	)

	http.Redirect(
		w,
		r,
		repo.HTMLURL,
		http.StatusFound,
	)
}
