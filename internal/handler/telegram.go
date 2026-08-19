package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ratneshrt/cf-daily/internal/service"
	"github.com/ratneshrt/cf-daily/internal/telegram"
)

type TelegramHandler struct {
	service *service.TelegramService
	secret  string
}

func NewTelegeamHandler(service *service.TelegramService, secret string) *TelegramHandler {
	return &TelegramHandler{
		service: service,
		secret:  secret,
	}
}

func (h *TelegramHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	secret := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")

	if secret != h.secret {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	var update telegram.Update

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(
			w,
			"invalid request",
			http.StatusBadRequest,
		)
		return
	}

	if err := h.service.HandleUpdate(r.Context(), update); err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
}
