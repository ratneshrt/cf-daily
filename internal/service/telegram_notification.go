package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ratneshrt/cf-daily/internal/model"
	"github.com/ratneshrt/cf-daily/internal/repository"
	"github.com/ratneshrt/cf-daily/internal/telegram"
)

type TelegramNotificationService struct {
	userRepository           *repository.TelegramUserRepository
	dailyProblemService      *DailyProblemService
	problemMessageRepository *repository.TelegramProblemMessageRepository
	telegramClient           *telegram.Client
	allowedUserIDs           map[int64]bool
}

func NewTelegramNotificationService(userRepository *repository.TelegramUserRepository, dailyProblemService *DailyProblemService, problemMessageRepository *repository.TelegramProblemMessageRepository, telegramClient *telegram.Client, allowedUserIDs []int64) *TelegramNotificationService {

	allowed := make(map[int64]bool)

	for _, id := range allowedUserIDs {
		allowed[id] = true
	}

	return &TelegramNotificationService{
		userRepository:           userRepository,
		dailyProblemService:      dailyProblemService,
		problemMessageRepository: problemMessageRepository,
		telegramClient:           telegramClient,
		allowedUserIDs:           allowed,
	}
}

func (s *TelegramNotificationService) sendProblemToUsers(ctx context.Context, problem *model.DailyProblem) error {

	users, err := s.userRepository.GetActiveUsers(ctx)

	if err != nil {
		return fmt.Errorf(
			"getting active telegram users: %w",
			err,
		)
	}

	message := buildDailyProblemMessage(problem)

	for _, user := range users {

		if !s.allowedUserIDs[user.TelegramUserID] {
			slog.Debug(
				"telegram user is not allowed",
				"telegram_user_id",
				user.TelegramUserID,
			)
			continue
		}

		existingMessage, err := s.problemMessageRepository.GetByUserAndProblem(
			ctx,
			user.TelegramUserID,
			problem.ID,
		)

		if err != nil {
			slog.Error(
				"failed to check existing telegram problem message",
				"telegram_user_id",
				user.TelegramUserID,
				"daily_problem_id",
				problem.ID,
				"error",
				err,
			)
			continue
		}

		if existingMessage != nil {
			slog.Info(
				"daily problem already sent",
				"telegram_user_id",
				user.TelegramUserID,
				"daily_problem_id",
				problem.ID,
				"telegram_message_id",
				existingMessage.TelegramMessageID,
			)
			continue
		}

		sentMessage, err := s.telegramClient.SendMessage(
			ctx,
			user.ChatID,
			message,
		)

		if err != nil {
			slog.Error(
				"failed to send daily problem",
				"telegram_user_id",
				user.TelegramUserID,
				"error",
				err,
			)
			continue
		}

		_, err = s.problemMessageRepository.Create(
			ctx,
			user.TelegramUserID,
			problem.ID,
			sentMessage.MessageID,
		)

		if err != nil {
			slog.Error(
				"failed to save telegram problem message",
				"telegram_user_id",
				user.TelegramUserID,
				"error",
				err,
			)
			continue
		}
	}

	return nil

}

func buildDailyProblemMessage(problem *model.DailyProblem) string {
	var builder strings.Builder

	builder.WriteString("🔥 TODAY'S CODEFORCES PROBLEM\n\n")

	builder.WriteString("📌 Problem: ")
	builder.WriteString(problem.Name)
	builder.WriteString("\n\n")

	builder.WriteString("⭐ Rating: ")
	builder.WriteString(fmt.Sprintf("%d", problem.Rating))
	builder.WriteString("\n\n")

	if len(problem.Tags) > 0 {
		builder.WriteString("🏷 Tags:\n")
		builder.WriteString(strings.Join(problem.Tags, " • "))
		builder.WriteString("\n\n")
	}

	builder.WriteString("🔗 ")
	builder.WriteString(problem.URL)
	builder.WriteString("\n\n")

	builder.WriteString("\n\n")

	builder.WriteString(
		"💻 Submit your solution by replying to THIS message:\n\n",
	)

	builder.WriteString("/submit\n")
	builder.WriteString("<your code>\n\n")

	builder.WriteString("✏️ Edit:\n")
	builder.WriteString("/edit\n")
	builder.WriteString("<new code>\n\n")

	builder.WriteString("🗑 Delete:\n")
	builder.WriteString("/delete")

	return builder.String()
}

func (s *TelegramNotificationService) SendTodayProblem(ctx context.Context) error {
	problem, err := s.dailyProblemService.GetToday(ctx)

	if err != nil {
		return fmt.Errorf(
			"getting today's problem: %w",
			err,
		)
	}

	if problem == nil {
		return fmt.Errorf(
			"today's problem is nil",
		)
	}

	return s.sendProblemToUsers(ctx, problem)
}
