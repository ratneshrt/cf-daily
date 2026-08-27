package handler

import (
	"fmt"
	"log"
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

	log.Printf(
		"github callback: installation_id=%s setup_action=%s",
		installationIDParam,
		setupAction,
	)

	if code == "" || state == "" {
		log.Printf("github callback missing code or state")

		http.Error(w, "missing code or state", http.StatusBadRequest)
		return
	}

	telegramUserID, err := h.githubStateRepository.Get(ctx, state)

	if err != nil {
		log.Printf(
			"github state validation failed: %v",
			err,
		)

		http.Error(
			w,
			"invalid or expired connection",
			http.StatusBadRequest,
		)
		return
	}

	log.Printf("github state consumed: telegram_user_id=%d", telegramUserID)

	accessToken, err := h.githubService.ExchangeCode(ctx, code)

	if err != nil {
		log.Printf("github oauth exchange failed: %v", err)

		http.Error(
			w,
			"failed to authorize GitHub",
			http.StatusInternalServerError,
		)
		return
	}

	consumedTelegramUserID, err := h.githubStateRepository.Consume(ctx, state)

	if err != nil {
		log.Printf("github state consume failed: state=%s error=%v", state, err)

		http.Error(w, "failed to finalize github connection", http.StatusInternalServerError)

		return
	}

	if consumedTelegramUserID != telegramUserID {
		log.Printf("github state user mismatch: validated=%d consumed=%d", telegramUserID, consumedTelegramUserID)

		http.Error(w, "invalid GitHub connection state", http.StatusBadRequest)

		return
	}

	log.Printf("github state validated: state=%s telegram_user_id=%d", state, telegramUserID)

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

	if err := h.githubService.CreateRepository(ctx, accessToken); err != nil {
		log.Printf(
			"github repository creation failed: %v",
			err,
		)

		http.Error(
			w,
			"Github connected, but failed to create flux-cf repository",
			http.StatusInternalServerError,
		)
		return
	}

	log.Printf("github repository created successfully: github_user=%s installation_id=%d", githubUser.Login, installationID)

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
