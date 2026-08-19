package handler

import (
	"net/http"
	"time"

	"github.com/ratneshrt/cf-daily/internal/service"
)

type TelegramNotificationHandler struct {
	service *service.TelegramNotificationService
}

func NewTelegramNotificationHandler(service *service.TelegramNotificationService) *TelegramNotificationHandler {
	return &TelegramNotificationHandler{
		service: service,
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

	err := h.service.SendDailyProblem(
		r.Context(),
		time.Now(),
	)

	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusOK)

	_, _ = w.Write(
		[]byte("daily problem sent"),
	)
}
