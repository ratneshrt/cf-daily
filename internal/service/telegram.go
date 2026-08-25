package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ratneshrt/cf-daily/internal/repository"
	"github.com/ratneshrt/cf-daily/internal/telegram"
)

type TelegramService struct {
	userRepository           *repository.TelegramUserRepository
	submissionRepository     *repository.CodeSubmissionRepository
	problemMessageRepository *repository.TelegramProblemMessageRepository
	telegramClient           *telegram.Client
	githubService            *GitHubService
	githubStateRepository    *repository.GitHubStateRepository
}

func NewTelegramService(userRepository *repository.TelegramUserRepository, submissionRepository *repository.CodeSubmissionRepository, problemMessageRepository *repository.TelegramProblemMessageRepository, telegramClient *telegram.Client, githubService *GitHubService, githubStateRepository *repository.GitHubStateRepository) *TelegramService {
	return &TelegramService{
		userRepository:           userRepository,
		submissionRepository:     submissionRepository,
		problemMessageRepository: problemMessageRepository,
		telegramClient:           telegramClient,
		githubService:            githubService,
		githubStateRepository:    githubStateRepository,
	}
}

func (s *TelegramService) HandleUpdate(
	ctx context.Context,
	update telegram.Update,
) error {
	if update.Message == nil {
		slog.Info("telegram update has no message")
		return nil
	}

	if update.Message.From == nil {
		slog.Info("telegram message has no sender")
		return nil
	}

	slog.Info(
		"telegram update received",
		"message_id",
		update.Message.MessageID,
		"user_id",
		update.Message.From.ID,
		"text",
		update.Message.Text,
		"has_reply",
		update.Message.ReplyToMessage != nil,
	)

	text := strings.TrimSpace(update.Message.Text)

	switch {
	case strings.HasPrefix(text, "/start"):
		slog.Info("routing to start")
		return s.handleStart(ctx, update.Message)

	case strings.HasPrefix(text, "/help"):
		slog.Info("routing to help")
		return s.handleHelp(ctx, update.Message)

	case strings.HasPrefix(text, "/submit"):
		slog.Info("routing to submit")
		return s.handleSubmit(ctx, update.Message)

	case strings.HasPrefix(text, "/edit"):
		slog.Info("routing to edit")
		return s.handleEdit(ctx, update.Message)

	case strings.HasPrefix(text, "/delete"):
		slog.Info("routing to delete")
		return s.handleDelete(ctx, update.Message)

	case strings.HasPrefix(text, "/connect"):
		return s.handleConnectGitHub(ctx, update.Message)

	default:
		slog.Info(
			"unknown telegram command",
			"text",
			update.Message.Text,
		)
		return nil
	}
}

func (s *TelegramService) handleConnectGitHub(ctx context.Context, message *telegram.Message) error {
	userID := message.From.ID

	state, err := GenerateGitHubState()

	if err != nil {
		return fmt.Errorf(
			"creating github connection state: %w",
			err,
		)
	}

	expiresAt := time.Now().Add(10 * time.Minute)

	err = s.githubStateRepository.Create(
		ctx,
		state,
		userID,
		expiresAt,
	)

	authURL := s.githubService.InstallationURL(state)

	return s.sendMessage(
		ctx,
		message.Chat.ID,
		"🔗 Connect your GitHub account\n\n"+
			"Install Flux on your GitHub account and "+
			"authorize it to manage your CF solutions.\n\n"+
			authURL,
	)
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

	slog.Info(
		"handleSubmit called",
		"message_id",
		message.MessageID,
		"user_id",
		message.From.ID,
		"text",
		message.Text,
		"has_reply",
		message.ReplyToMessage != nil,
	)

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

	replyMessageID := message.ReplyToMessage.MessageID

	slog.Info(
		"looking up telegram problem message",
		"telegram_user_id", userID,
		"reply_message_id", replyMessageID,
	)

	problemMessage, err := s.problemMessageRepository.GetByMessageID(
		ctx,
		userID,
		replyMessageID,
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
	replyMessageID := message.ReplyToMessage.MessageID

	problemMessage, err := s.problemMessageRepository.GetByMessageID(
		ctx,
		userID,
		replyMessageID,
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

	existing, err := s.submissionRepository.Get(
		ctx,
		userID,
		problemMessage.DailyProblemID,
	)

	if err != nil {
		return fmt.Errorf(
			"checking exisiting submission: %w",
			err,
		)
	}

	if existing == nil {
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
	if message.ReplyToMessage == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"❌ Please reply to the daily problem message.\n\n"+
				"Use:\n\n"+
				"/delete",
		)
	}

	userID := message.From.ID

	replyMessageID := message.ReplyToMessage.MessageID

	problemMessage, err := s.problemMessageRepository.GetByMessageID(
		ctx,
		userID,
		replyMessageID,
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
			"❌ I couldn't find the problem you replied to.\n\n"+
				"Please reply directly to the daily problem message.",
		)
	}

	submission, err := s.submissionRepository.Get(
		ctx,
		userID,
		problemMessage.DailyProblemID,
	)

	if err != nil {
		return fmt.Errorf(
			"checking existing submission: %w",
			err,
		)
	}

	if submission == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"❌ You don't have a submitted solution for this problem.",
		)
	}

	err = s.submissionRepository.Delete(
		ctx,
		userID,
		problemMessage.DailyProblemID,
	)

	if err != nil {
		return fmt.Errorf(
			"deleting submission: %w",
			err,
		)
	}

	return s.sendMessage(
		ctx,
		message.Chat.ID,
		"🗑️ Your solution has been deleted successfully.",
	)
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
