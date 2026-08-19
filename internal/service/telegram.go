package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ratneshrt/cf-daily/internal/repository"
	"github.com/ratneshrt/cf-daily/internal/telegram"
)

type TelegramService struct {
	userRepository           *repository.TelegramUserRepository
	submissionRepository     *repository.CodeSubmissionRepository
	problemMessageRepository *repository.TelegramProblemMessageRepository
	telegramClient           *telegram.Client
}

func NewTelegramService(userRepository *repository.TelegramUserRepository, submissionRepository *repository.CodeSubmissionRepository, problemMessageRepository *repository.TelegramProblemMessageRepository, telegramClient *telegram.Client) *TelegramService {
	return &TelegramService{
		userRepository:           userRepository,
		submissionRepository:     submissionRepository,
		problemMessageRepository: problemMessageRepository,
		telegramClient:           telegramClient,
	}
}

func (s *TelegramService) HandleUpdate(ctx context.Context, update telegram.Update) error {
	if update.Message == nil {
		return nil
	}

	if update.Message.From == nil {
		return nil
	}

	text := update.Message.Text

	switch {
	case strings.HasPrefix(text, "/start"):
		return s.handleStart(ctx, update.Message)

	case strings.HasPrefix(text, "/help"):
		return s.handleHelp(ctx, update.Message)

	case strings.HasPrefix(text, "/edit"):
		return s.handleEdit(ctx, update.Message)

	case strings.HasPrefix(text, "/delete"):
		return s.handleDelete(ctx, update.Message)

	case strings.HasPrefix(text, "submit"):
		return s.handleSubmit(ctx, update.Message)

	default:
		return nil
	}
}

func (s *TelegramService) handleStart(ctx context.Context, message *telegram.Message) error {
	user, err := s.userRepository.GetByTelegramUserID(
		ctx,
		message.From.ID,
	)

	if err != nil {
		return fmt.Errorf(
			"checking telegram user: %w",
			err,
		)
	}

	if user == nil {
		user, err = s.userRepository.Create(
			ctx,
			message.From.ID,
			message.Chat.ID,
			message.From.Username,
			message.From.FirstName,
		)

		if err != nil {
			return fmt.Errorf(
				"creating telegram user: %w",
				err,
			)
		} else {
			user, err = s.userRepository.Activate(
				ctx,
				message.From.ID,
				message.Chat.ID,
				message.From.Username,
				message.From.FirstName,
			)

			if err != nil {
				return fmt.Errorf(
					"activating telegram user: %w",
					err,
				)
			}
		}
	}

	text := fmt.Sprintf(
		"👋 Welcome %s!\n\n"+
			"You are now registered for CF Daily.\n\n"+
			"Use /help to see available commands.",
		user.FirstName,
	)

	_, err = s.telegramClient.SendMessage(
		ctx,
		user.ChatID,
		text,
	)

	if err != nil {
		return fmt.Errorf(
			"sendinf welcome message: %w",
			err,
		)
	}

	return nil
}

func (s *TelegramService) handleHelp(ctx context.Context, message *telegram.Message) error {
	text := `CF Daily Commands

/start - Register or activate your account
/help - Show available commands

Reply to a daily problem:

/submit
<your code>

 /edit
<new code>

 /delete

Your code formatting, including spaces,
tabs and newlines, is preserved.`

	_, err := s.telegramClient.SendMessage(
		ctx,
		message.Chat.ID,
		text,
	)

	if err != nil {
		return fmt.Errorf(
			"sending help message: %w",
			err,
		)
	}

	return nil
}

func (s *TelegramService) handleSubmit(ctx context.Context, message *telegram.Message) error {
	return nil
}

func (s *TelegramService) handleEdit(ctx context.Context, message *telegram.Message) error {
	return nil
}

func (s *TelegramService) handleDelete(ctx context.Context, message *telegram.Message) error {
	return nil
}
