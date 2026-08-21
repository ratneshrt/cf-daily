package handler

import (
	"context"
	"net/http"

	"github.com/ratneshrt/cf-daily/internal/service"
)

type CronHandler struct {
	telegramNotificationService *service.TelegramNotificationService
	telegramReminderService     *service.TelegramReminderService
	cronSecret                  string
}

func NewCronHandler(telegramNotification *service.TelegramNotificationService, telegramReminderService *service.TelegramReminderService, cronSecret string) *CronHandler {
	return &CronHandler{
		telegramNotificationService: telegramNotification,
		telegramReminderService:     telegramReminderService,
		cronSecret:                  cronSecret,
	}
}

func (h *CronHandler) authenticate(r *http.Request) bool {
	return r.Header.Get("X-Cron_Secret") == h.cronSecret
}

func (h *CronHandler) DailyProblem(w http.ResponseWriter, r *http.Request) {
	if !h.authenticate(r) {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)

		return
	}

	err := h.telegramNotificationService.SendTodayProblem(
		context.Background(),
	)

	if err != nil {
		http.Error(
			w,
			"failed to send daily problem",
			http.StatusInternalServerError,
		)

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("daily problem sent"))
}

func (h *CronHandler) NightlyReminder(w http.ResponseWriter, r *http.Request) {
	if !h.authenticate(r) {
		http.Error(
			w,
			"unauthorized",
			http.StatusInternalServerError,
		)
		return
	}

	err := h.telegramReminderService.SendNightlyReminder(context.Background())

	if err != nil {
		http.Error(
			w,
			"failed to send nightly reminder",
			http.StatusInternalServerError,
		)

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("nightly reminder sent"))
}
