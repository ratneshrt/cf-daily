package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ratneshrt/cf-daily/internal/telegram"
)

type TelegramHandler struct {
	client *telegram.Client
	secret string
}

func NewTelegeamHandler(client *telegram.Client, secret string) *TelegramHandler {
	return &TelegramHandler{
		client: client,
		secret: secret,
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

	w.WriteHeader(http.StatusOK)
}
