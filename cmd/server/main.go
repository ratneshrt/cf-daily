package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/ratneshrt/cf-daily/internal/codeforces"
	"github.com/ratneshrt/cf-daily/internal/config"
	"github.com/ratneshrt/cf-daily/internal/database"
	"github.com/ratneshrt/cf-daily/internal/handler"
	"github.com/ratneshrt/cf-daily/internal/repository"
	"github.com/ratneshrt/cf-daily/internal/service"
	"github.com/ratneshrt/cf-daily/internal/telegram"
)

// shutdownTimeout gives in-flight work - a GitHub push, say - time to finish
// when the container is replaced.
const shutdownTimeout = 20 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found, using env var")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})))

	if len(cfg.TelegramAllowedUserIDs) == 0 {
		slog.Warn(
			"TELEGRAM_ALLOWED_USER_IDS is empty, so no daily problems or reminders will be delivered to anyone",
		)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
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

	githubService, err := service.NewGitHubService(
		cfg.GitHubAppID,
		cfg.GitHubAppSlug,
		cfg.GitHubClientID,
		cfg.GitHubClientSecret,
		cfg.GitHubPrivateKey,
		cfg.GitHubCallbackURL,
		cfg.GitHubRepositoryName,
	)

	if err != nil {
		return err
	}

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
		telegramProblemMessageRepository,
		dailyProblemRepository,
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
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		// Long enough for a scheduled send to message every user.
		WriteTimeout: 3 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}

	serverErrors := make(chan error, 1)

	go func() {
		slog.Info("server started", "port", cfg.Port)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
			return
		}

		serverErrors <- nil
	}()

	select {
	case err := <-serverErrors:
		return err

	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)

	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	slog.Info("server stopped cleanly")

	return nil
}
