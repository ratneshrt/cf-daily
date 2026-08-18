package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ratneshrt/cf-daily/internal/codeforces"
)

type ProblemHandler struct {
	service   *codeforces.Service
	minRating int
	maxRating int
}

func NewProblemHandler(service *codeforces.Service, minRating, maxRating int) *ProblemHandler {
	return &ProblemHandler{
		service:   service,
		minRating: minRating,
		maxRating: maxRating,
	}
}

func (h *ProblemHandler) GetProblem(w http.ResponseWriter, r *http.Request) {
	problem, err := h.service.GetRandomProblem(
		r.Context(),
		h.minRating,
		h.maxRating,
	)

	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	response := map[string]interface{}{
		"problem": problem,
		"url":     problem.URL(),
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}
