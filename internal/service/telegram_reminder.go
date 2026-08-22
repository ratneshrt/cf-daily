package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ratneshrt/cf-daily/internal/repository"
	"github.com/ratneshrt/cf-daily/internal/telegram"
)

type TelegramReminderService struct {
	userRepository       *repository.TelegramUserRepository
	submissionRepository *repository.CodeSubmissionRepository
	dailyProblemService  *DailyProblemService
	telegramClient       *telegram.Client
	allowedUserIDs       map[int64]bool
}

func NewTelegramReminderService(userRepository *repository.TelegramUserRepository, submissionRepository *repository.CodeSubmissionRepository, dailyProblemService *DailyProblemService, telegramClient *telegram.Client, allowedUserIDs []int64) *TelegramReminderService {

	allowed := make(map[int64]bool)

	for _, id := range allowedUserIDs {
		allowed[id] = true
	}

	return &TelegramReminderService{
		userRepository:       userRepository,
		submissionRepository: submissionRepository,
		dailyProblemService:  dailyProblemService,
		telegramClient:       telegramClient,
		allowedUserIDs:       allowed,
	}
}

func (s *TelegramReminderService) SendNightlyReminder(ctx context.Context) error {
	problem, err := s.dailyProblemService.GetToday(ctx)
	if err != nil {
		return fmt.Errorf(
			"getting today's problem: %w",
			err,
		)
	}

	if problem == nil {
		return fmt.Errorf(
			"today's problem not found",
		)
	}

	users, err := s.userRepository.GetActiveUsers(ctx)

	if err != nil {
		return fmt.Errorf("getting active telegram users: %w", err)
	}

	for _, user := range users {

		if !s.allowedUserIDs[user.TelegramUserID] {
			continue
		}

		submission, err := s.submissionRepository.Get(ctx, user.TelegramUserID, problem.ID)

		if err != nil {
			slog.Error(
				"failed to check submission",
				"telegram_user_id",
				user.TelegramUserID,
				"daily_problem_id",
				problem.ID,
				"error",
				err,
			)
			continue
		}

		var message string

		if submission == nil {
			message = fmt.Sprintf(
				"⏰ 11:30 PM Reminder\n\n"+
					"🔥 You haven't solved today's problem yet!\n\n"+
					"📌 %s\n"+
					"⭐ Rating: %d\n"+
					"URL: %s\n\n"+
					"Give it a try before the day ends! 💪",
				problem.Name,
				problem.Rating,
				problem.URL,
			)
		} else {
			message = fmt.Sprintf(
				"🎉 Great job!\n\n"+
					"You already submitted today's problem:\n\n"+
					"📌 %s\n\n"+
					"Keep the streak going! 🔥",
				problem.Name,
			)
		}

		_, err = s.telegramClient.SendMessage(
			ctx,
			user.ChatID,
			message,
		)

		if err != nil {
			slog.Error(
				"failed to send nightly reminder",
				"telegram_user_id",
				user.TelegramUserID,
				"error",
				err,
			)
		}
		continue
	}

	return nil
}
