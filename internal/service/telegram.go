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
	if message.ReplyToMessage == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"❌ Please reply to the daily problem message.\n\n"+
				"Example:\n\n"+
				"/submit\n"+
				"<your code>",
		)
	}

	parts := strings.SplitN(message.Text, "\n", 2)

	if len(parts) < 2 || parts[1] == "" {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"❌ No code provided.\n\n"+
				"Use:\n\n"+
				"/submit\n"+
				"<your code>",
		)
	}

	code := parts[1]

	userID := message.From.ID

	problemMessage, err := s.problemMessageRepository.GetByMessageID(
		ctx,
		userID,
		message.ReplyToMessage.MessageID,
	)

	if err != nil {
		return fmt.Errorf("getting problem message: %w", err)
	}

	if problemMessage == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"❌ I couldn't find the problem you replied to.\n\n"+
				"Please reply directly to the daily problem message.",
		)
	}

	existing, err := s.submissionRepository.Get(
		ctx,
		userID,
		problemMessage.DailyProblemID,
	)

	if err != nil {
		return fmt.Errorf("checking existing submission: %w", err)
	}

	if existing != nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"⚠️ You already submitted a solution for this problem.\n\n"+
				"Use /edit to replace it.",
		)
	}

	_, err = s.submissionRepository.Create(
		ctx,
		userID,
		problemMessage.DailyProblemID,
		code,
	)

	if err != nil {
		return fmt.Errorf("creating submission: %w", err)
	}

	return s.sendMessage(
		ctx,
		message.Chat.ID,
		"✅ Your solution has been saved!",
	)
}

func (s *TelegramService) handleEdit(ctx context.Context, message *telegram.Message) error {
	if message.ReplyToMessage == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"❌ Please reply to the daily problem message.",
		)
	}

	parts := strings.SplitN(message.Text, "\n", 2)

	if len(parts) < 2 || parts[1] == "" {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"❌ No new code provided.\n\n"+
				"Use:\n\n"+
				"/edit\n"+
				"<new code>",
		)
	}

	code := parts[1]

	userID := message.From.ID

	problemMessage, err := s.problemMessageRepository.GetByMessageID(
		ctx,
		userID,
		message.ReplyToMessage.MessageID,
	)

	if err != nil {
		return fmt.Errorf(
			"getting problem message: %w",
			err,
		)
	}

	if problemMessage == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"❌ You don't have a submission for this problem yet.\n\n"+
				"Use /submit first.",
		)
	}

	_, err = s.submissionRepository.Update(
		ctx,
		userID,
		problemMessage.DailyProblemID,
		code,
	)

	if err != nil {
		return fmt.Errorf(
			"updating submission: %w",
			err,
		)
	}

	return s.sendMessage(
		ctx,
		message.Chat.ID,
		"✏️ Your solution has been updated!",
	)
}

func (s *TelegramService) handleDelete(ctx context.Context, message *telegram.Message) error {
	return nil
}

func (s *TelegramService) sendMessage(ctx context.Context, chatID int64, text string) error {
	_, err := s.telegramClient.SendMessage(ctx, chatID, text)

	if err != nil {
		return fmt.Errorf(
			"sending telegram message: %w",
			err,
		)
	}

	return nil
}

func (s *TelegramService) sendError(ctx context.Context, chatID int64, text string) error {
	return s.sendMessage(ctx, chatID, text)
}
