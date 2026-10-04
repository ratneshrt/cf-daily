package handler

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ratneshrt/cf-daily/internal/service"
	"github.com/ratneshrt/cf-daily/internal/telegram"
)

// updateRetention is how long a processed update id is remembered, which only
// needs to outlast Telegram's redelivery window.
const updateRetention = 15 * time.Minute

// maxTrackedUpdates bounds the dedupe map in case cleanup never triggers.
const maxTrackedUpdates = 10000

type TelegramHandler struct {
	service *service.TelegramService
	secret  string

	mu          sync.Mutex
	seenUpdates map[int64]time.Time
}

func NewTelegeamHandler(service *service.TelegramService, secret string) *TelegramHandler {
	return &TelegramHandler{
		service:     service,
		secret:      secret,
		seenUpdates: make(map[int64]time.Time),
	}
}

// alreadySeen records an update id and reports whether it had been handled
// before. Telegram can deliver the same update more than once, and handling a
// /submit twice would mean a second commit.
func (h *TelegramHandler) alreadySeen(updateID int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()

	if _, seen := h.seenUpdates[updateID]; seen {
		return true
	}

	if len(h.seenUpdates) >= maxTrackedUpdates {
		for id, at := range h.seenUpdates {
			if now.Sub(at) > updateRetention {
				delete(h.seenUpdates, id)
			}
		}

		// Still full: the map is all recent entries, so start over rather than
		// growing without bound.
		if len(h.seenUpdates) >= maxTrackedUpdates {
			h.seenUpdates = make(map[int64]time.Time)
		}
	}

	h.seenUpdates[updateID] = now

	return false
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

	if subtle.ConstantTimeCompare([]byte(secret), []byte(h.secret)) != 1 {
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

	if h.alreadySeen(update.UpdateID) {
		slog.Info("ignoring duplicate telegram update", "update_id", update.UpdateID)

		w.WriteHeader(http.StatusOK)

		return
	}

	// Telegram redelivers any update that is not acknowledged with a 2xx, which
	// would re-run the command. The error is logged, and HandleUpdate tells the
	// user, so the update is acknowledged either way.
	if err := h.service.HandleUpdate(r.Context(), update); err != nil {
		slog.Error(
			"telegram update handling failed",
			"update_id",
			update.UpdateID,
			"error",
			err,
		)
	}

	w.WriteHeader(http.StatusOK)
}
