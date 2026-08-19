package service

import (
	"context"

	"github.com/ratneshrt/cf-daily/internal/repository"
	"github.com/ratneshrt/cf-daily/internal/telegram"
)

type TelegramService struct {
	userRepository           *repository.TelegramUserRepository
	submissionRepository     *repository.CodeSubmissionRepository
	problemMessageRepository *repository.TelegramProblemMessageRepository
	telegramClient           *telegram.Client
}

func NewTelegramService(userRepository *repository.TelegramUserRepository, telegramClient *telegram.Client) *TelegramService {
	return &TelegramService{
		userRepository: userRepository,
		telegramClient: telegramClient,
	}
}

func (s *TelegramService) HandleUpdate(ctx context.Context, update telegram.Update) error {
	if update.Message == nil {
		return nil
	}

	if update.Message.From == nil {
		return nil
	}

	return nil
}
