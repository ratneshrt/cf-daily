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
	"github.com/ratneshrt/cf-daily/internal/telegram"
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

	githubStateRepository := repository.NewGitHubStateRepository(db)
	githubService := service.NewGitHubService(
		cfg.GitHubAppID,
		cfg.GitHubClientID,
		cfg.GitHubClientSecret,
		cfg.GitHubPrivateKey,
		cfg.GitHubCallbackURL,
		cfg.GitHubRepositoryName,
	)

	// -------- telegram
	telegramClient := telegram.NewClient(cfg.TelegramBotToken)

	telegramUserRepository := repository.NewTelegramUserRepository(db)

	codeSubmissionRepository := repository.NewCodeSubmissionRepository(db)

	telegramProblemMessageRepository := repository.NewTelegramProblemMessageRepository(db)

	telegramService := service.NewTelegramService(
		telegramUserRepository,
		codeSubmissionRepository,
		telegramProblemMessageRepository,
		dailyProblemRepository,
		telegramClient,
		githubService,
		githubStateRepository,
	)

	telegramNotificationService := service.NewTelegramNotificationService(
		telegramUserRepository,
		dailyProblemService,
		telegramProblemMessageRepository,
		telegramClient,
		cfg.TelegramAllowedUserIDs,
	)

	telegramReminderService := service.NewTelegramReminderService(
		telegramUserRepository,
		codeSubmissionRepository,
		dailyProblemService,
		telegramClient,
		cfg.TelegramAllowedUserIDs,
	)

	telegramHandler := handler.NewTelegeamHandler(telegramService, cfg.TelegramWebhookSecret)

	telegramNotificationHandler := handler.NewTelegramNotificationHandler(telegramNotificationService, cfg.CronSecret, telegramReminderService)

	githubHandler := handler.NewGitHubHandler(
		githubService,
		githubStateRepository,
		telegramUserRepository,
	)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /problem", problemHandler.GetProblem)
	mux.HandleFunc("GET /problem/today", dailyProblemHandler.GetToday)
	mux.HandleFunc("POST /telegram/webhook", telegramHandler.Webhook)
	mux.HandleFunc("POST /telegram/send-daily-problem", telegramNotificationHandler.SendDailyProblem)
	mux.HandleFunc("POST /telegram/send-reminder", telegramNotificationHandler.SendReminder)
	mux.HandleFunc("GET /github/callback", githubHandler.Callback)

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
