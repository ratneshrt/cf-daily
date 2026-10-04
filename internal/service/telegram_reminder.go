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

type TelegramReminderService struct {
	userRepository           *repository.TelegramUserRepository
	problemMessageRepository *repository.TelegramProblemMessageRepository
	dailyProblemRepository   *repository.DailyProblemRepository
	telegramClient           *telegram.Client
	allowedUserIDs           map[int64]bool
}

func NewTelegramReminderService(userRepository *repository.TelegramUserRepository, problemMessageRepository *repository.TelegramProblemMessageRepository, dailyProblemRepository *repository.DailyProblemRepository, telegramClient *telegram.Client, allowedUserIDs []int64) *TelegramReminderService {

	allowed := make(map[int64]bool)

	for _, id := range allowedUserIDs {
		allowed[id] = true
	}

	return &TelegramReminderService{
		userRepository:           userRepository,
		problemMessageRepository: problemMessageRepository,
		dailyProblemRepository:   dailyProblemRepository,
		telegramClient:           telegramClient,
		allowedUserIDs:           allowed,
	}
}

// SendNightlyReminder reminds every active user about the last problem that was
// actually sent to them. It never assigns a new problem, so a late trigger can
// only ever remind about a problem the user has already seen.
func (s *TelegramReminderService) SendNightlyReminder(ctx context.Context) error {
	users, err := s.userRepository.GetActiveUsers(ctx)

	if err != nil {
		return fmt.Errorf("getting active telegram users: %w", err)
	}

	sent := 0
	skipped := 0

	for _, user := range users {

		if !s.allowedUserIDs[user.TelegramUserID] {
			skipped++
			continue
		}

		problemMessage, err := s.problemMessageRepository.GetLatestByUser(
			ctx,
			user.TelegramUserID,
		)

		if err != nil {
			slog.Error(
				"failed to get latest problem message",
				"telegram_user_id",
				user.TelegramUserID,
				"error",
				err,
			)
			continue
		}

		if problemMessage == nil {
			slog.Info(
				"no problem has been sent to user yet, skipping reminder",
				"telegram_user_id",
				user.TelegramUserID,
			)
			skipped++
			continue
		}

		problem, err := s.dailyProblemRepository.GetByID(
			ctx,
			problemMessage.DailyProblemID,
		)

		if err != nil {
			slog.Error(
				"failed to get daily problem for reminder",
				"telegram_user_id",
				user.TelegramUserID,
				"daily_problem_id",
				problemMessage.DailyProblemID,
				"error",
				err,
			)
			continue
		}

		if problem == nil {
			slog.Error(
				"daily problem referenced by problem message is missing",
				"telegram_user_id",
				user.TelegramUserID,
				"daily_problem_id",
				problemMessage.DailyProblemID,
			)
			continue
		}

		_, err = s.telegramClient.SendMessage(
			ctx,
			user.ChatID,
			buildReminderMessage(problem),
		)

		if err != nil {
			slog.Error(
				"failed to send nightly reminder",
				"telegram_user_id",
				user.TelegramUserID,
				"error",
				err,
			)

			continue
		}

		sent++
	}

	slog.Info(
		"reminder send finished",
		"active_users",
		len(users),
		"sent",
		sent,
		"skipped",
		skipped,
	)

	return nil
}

func buildReminderMessage(problem *model.DailyProblem) string {
	var builder strings.Builder

	builder.WriteString("REMINDER: solve and submit today's problem\n")
	builder.WriteString("(ignore if already submitted)\n\n")

	builder.WriteString("Problem: ")
	builder.WriteString(problem.Name)
	builder.WriteString("\n")

	builder.WriteString("Rating: ")
	builder.WriteString(fmt.Sprintf("%d", problem.Rating))
	builder.WriteString("\n")

	builder.WriteString("URL: ")
	builder.WriteString(problem.URL)
	builder.WriteString("\n\n")

	builder.WriteString("Submit by replying to this message:\n\n")
	builder.WriteString("/submit\n")
	builder.WriteString("<your code>")

	return builder.String()
}
