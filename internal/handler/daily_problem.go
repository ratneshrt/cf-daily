package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ratneshrt/cf-daily/internal/service"
)

type DailyProblemHandler struct {
	service *service.DailyProblemService
}

func NewDailyProblemHandler(service *service.DailyProblemService) *DailyProblemHandler {
	return &DailyProblemHandler{
		service: service,
	}
}

func (h *DailyProblemHandler) GetToday(w http.ResponseWriter, r *http.Request) {
	problem, err := h.service.GetToday(
		r.Context(),
	)

	if err != nil {
		http.Error(w, "failed to get today's problem", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(problem); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
