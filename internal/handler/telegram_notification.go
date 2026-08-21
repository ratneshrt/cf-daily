package handler

import (
	"net/http"

	"github.com/ratneshrt/cf-daily/internal/service"
)

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

func (h *TelegramNotificationHandler) SendDailyProblem(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	if r.Header.Get("X-Cron-Secret") != h.cronSecret {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)

		return
	}

	err := h.notificationService.SendTodayProblem(
		r.Context(),
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

	if r.Header.Get("X-Cron-Secret") != h.cronSecret {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	err := h.reminderService.SendNightlyReminder(
		r.Context(),
	)

	if err != nil {
		http.Error(
			w,
			"failed to send daily reminder",
			http.StatusInternalServerError,
		)

		return
	}

	w.WriteHeader(http.StatusOK)

	_, _ = w.Write(
		[]byte("daily problem sent"),
	)
}
