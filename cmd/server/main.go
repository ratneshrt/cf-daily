package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/ratneshrt/cf-daily/internal/codeforces"
	"github.com/ratneshrt/cf-daily/internal/config"
	"github.com/ratneshrt/cf-daily/internal/database"
	handler "github.com/ratneshrt/cf-daily/internal/handler"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

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

	codeforcesClient := codeforces.NewClient()

	codeforcesSerive := codeforces.NewService(
		codeforcesClient,
	)

	healthHandler := handler.Health

	problemHandler := handler.NewProblemHandler(
		codeforcesSerive,
		cfg.MinRating,
		cfg.MaxRating,
	)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /problem", problemHandler.GetProblem)

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
