package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/ratneshrt/cf-daily/internal/codeforces"
	"github.com/ratneshrt/cf-daily/internal/config"
	"github.com/ratneshrt/cf-daily/internal/database"
	"github.com/ratneshrt/cf-daily/internal/handler"
	"github.com/ratneshrt/cf-daily/internal/repository"
	"github.com/ratneshrt/cf-daily/internal/service"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found, using env var")
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect db", "error", err)
		os.Exit(1)
	}

	defer db.Close()

	// ------- Codeforces
	codeforcesClient := codeforces.NewClient()

	codeforcesService := codeforces.NewService(
		codeforcesClient,
	)
	// ---------

	// --------- Daily Problem
	dailyProblemRepository := repository.NewDailyProblemRepository(db)

	dailyProblemService := service.NewDailyProblemService(dailyProblemRepository, codeforcesService, cfg.MinRating, cfg.MaxRating)

	dailyProblemHandler := handler.NewDailyProblemHandler(dailyProblemService)
	// -----------

	// ------------ health
	healthHandler := handler.Health
	// ------------

	problemHandler := handler.NewProblemHandler(
		codeforcesService,
		cfg.MinRating,
		cfg.MaxRating,
	)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /problem", problemHandler.GetProblem)
	mux.HandleFunc("GET /problem/today", dailyProblemHandler.GetToday)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	slog.Info("server started", "port", cfg.Port)

	if err := server.ListenAndServe(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
