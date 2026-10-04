package handler

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"time"

	"github.com/ratneshrt/cf-daily/internal/service"
)

// cronTimeout bounds a scheduled send: fetching the problemset and messaging
// every user should take seconds, not minutes.
const cronTimeout = 2 * time.Minute

type TelegramNotificationHandler struct {
	notificationService *service.TelegramNotificationService
	reminderService     *service.TelegramReminderService
	cronSecret          string
}

func NewTelegramNotificationHandler(notificationService *service.TelegramNotificationService, cronSecret string, reminderService *service.TelegramReminderService) *TelegramNotificationHandler {
	return &TelegramNotificationHandler{
		notificationService: notificationService,
		reminderService:     reminderService,
		cronSecret:          cronSecret,
	}
}

func (h *TelegramNotificationHandler) authorized(r *http.Request) bool {
	provided := r.Header.Get("X-Cron-Secret")

	return subtle.ConstantTimeCompare([]byte(provided), []byte(h.cronSecret)) == 1
}

func (h *TelegramNotificationHandler) SendDailyProblem(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	if !h.authorized(r) {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)

		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cronTimeout)
	defer cancel()

	err := h.notificationService.SendTodayProblem(ctx)

	if err != nil {
		slog.Error("failed to send daily problem", "error", err)

		http.Error(
			w,
			"failed to send daily problem",
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusOK)

	_, _ = w.Write(
		[]byte("daily problem sent"),
	)
}

func (h *TelegramNotificationHandler) SendReminder(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	if !h.authorized(r) {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), cronTimeout)
	defer cancel()

	err := h.reminderService.SendNightlyReminder(ctx)

	if err != nil {
		slog.Error("failed to send reminder", "error", err)

		http.Error(
			w,
			"failed to send reminder",
			http.StatusInternalServerError,
		)

		return
	}

	w.WriteHeader(http.StatusOK)

	_, _ = w.Write(
		[]byte("reminder sent"),
	)
}
