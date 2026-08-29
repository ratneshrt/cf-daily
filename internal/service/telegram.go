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
	dailyProblemRepository   *repository.DailyProblemRepository
	telegramClient           *telegram.Client
	githubService            *GitHubService
	githubStateRepository    *repository.GitHubStateRepository
}

func NewTelegramService(userRepository *repository.TelegramUserRepository, submissionRepository *repository.CodeSubmissionRepository, problemMessageRepository *repository.TelegramProblemMessageRepository, dailyProblemRepository *repository.DailyProblemRepository, telegramClient *telegram.Client, githubService *GitHubService, githubStateRepository *repository.GitHubStateRepository) *TelegramService {
	return &TelegramService{
		userRepository:           userRepository,
		submissionRepository:     submissionRepository,
		problemMessageRepository: problemMessageRepository,
		dailyProblemRepository:   dailyProblemRepository,
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

	slog.Info("github connect state generated", "telegram_user_id", userID, "state", state, "expires_at", expiresAt)

	err = s.githubStateRepository.Create(
		ctx,
		state,
		userID,
		expiresAt,
	)

	if err != nil {
		slog.Error("github connect state create failed", "telegram_user_id", userID, "state", state, "error", err)

		return fmt.Errorf(
			"saving github connection state: %w",
			err,
		)
	}

	slog.Info("github connect state saved", "telegram_user_id", userID, "state", state)

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
<your code> -> to edit your code

/edit
<new code> -> to edit your code

/delete -> to delete your code

`

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
			"Please reply to the daily problem message.\n\n"+
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
			"No code provided.\n\n"+
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
			"I couldn't find the problem you replied to.\n\n"+
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

	problem, err := s.dailyProblemRepository.GetByID(
		ctx,
		problemMessage.DailyProblemID,
	)

	if err != nil {
		return fmt.Errorf(
			"getting daily problem: %w",
			err,
		)
	}

	if problem == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"Daily problem not found.",
		)
	}

	user, err := s.userRepository.GetByTelegramUserID(
		ctx,
		userID,
	)

	if err != nil {
		return fmt.Errorf(
			"getting telegram user: %w",
			err,
		)
	}

	if user == nil ||
		user.GithubUsername == nil ||
		user.GithubInstallationID == nil {

		return s.sendError(
			ctx,
			message.Chat.ID,
			"Please connect GitHub first using /connect.",
		)
	}

	path := buildSolutionPath(
		problem.ContestID,
		problem.ProblemIndex,
		problem.Name,
	)

	err = s.githubService.CreateOrUpdateFile(
		ctx,
		*user.GithubInstallationID,
		*user.GithubUsername,
		path,
		code,
		fmt.Sprintf(
			"Add solution for %d%s - %s",
			problem.ContestID,
			problem.ProblemIndex,
			problem.Name,
		),
	)

	if err != nil {
		slog.Error(
			"failed to push solution to github",
			"telegram_user_id", userID,
			"daily_problem_id", problem.ID,
			"path", path,
			"error", err,
		)

		return s.sendError(
			ctx,
			message.Chat.ID,
			"Failed to push your solution to GitHub.",
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
			"Please reply to the daily problem message.",
		)
	}

	parts := strings.SplitN(message.Text, "\n", 2)

	if len(parts) < 2 || parts[1] == "" {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"No new code provided.\n\n"+
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
			"You don't have a submission for this problem yet.\n\n"+
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
			"You don't have a submission for this problem yet.\n\n"+
				"Use /submit first.",
		)
	}

	problem, err := s.dailyProblemRepository.GetByID(ctx, problemMessage.DailyProblemID)

	if err != nil {
		return fmt.Errorf("getting daily problem: %w", err)
	}

	if problem == nil {
		return s.sendError(ctx, message.Chat.ID, "Daily problem not found")
	}

	user, err := s.userRepository.GetByTelegramUserID(ctx, userID)

	if err != nil {
		return fmt.Errorf("getting telegram user: %w", err)
	}

	if user == nil || user.GithubUsername == nil || user.GithubInstallationID == nil {
		return s.sendError(ctx, message.Chat.ID, "Please connect Github first using /connect")
	}

	path := buildSolutionPath(problem.ContestID, problem.ProblemIndex, problem.Name)

	err = s.githubService.CreateOrUpdateFile(ctx, *user.GithubInstallationID, *user.GithubUsername, path, code, fmt.Sprintf("update solution for %d%s - %s", problem.ContestID, problem.ProblemIndex, problem.Name))

	if err != nil {
		slog.Error(
			"failed to update solution on github",
			"telegram_user_id", userID,
			"daily_problem_id", problem.ID,
			"path", path,
			"error", err,
		)

		return s.sendError(
			ctx,
			message.Chat.ID,
			"Failed to update your solution on GitHub.",
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
			"Please reply to the daily problem message.\n\n"+
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
			"I couldn't find the problem you replied to.\n\n"+
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
			"You don't have a submitted solution for this problem.",
		)
	}

	problem, err := s.dailyProblemRepository.GetByID(
		ctx,
		problemMessage.DailyProblemID,
	)

	if err != nil {
		return fmt.Errorf(
			"getting daily problem: %w",
			err,
		)
	}

	if problem == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"Daily problem not found.",
		)
	}

	user, err := s.userRepository.GetByTelegramUserID(
		ctx,
		userID,
	)

	if err != nil {
		return fmt.Errorf(
			"getting telegram user: %w",
			err,
		)
	}

	if user == nil ||
		user.GithubUsername == nil ||
		user.GithubInstallationID == nil {

		return s.sendError(
			ctx,
			message.Chat.ID,
			"Please connect GitHub first using /connect.",
		)
	}

	path := buildSolutionPath(
		problem.ContestID,
		problem.ProblemIndex,
		problem.Name,
	)

	err = s.githubService.DeleteFile(
		ctx,
		*user.GithubInstallationID,
		*user.GithubUsername,
		path,
		fmt.Sprintf(
			"Delete solution for %d%s - %s",
			problem.ContestID,
			problem.ProblemIndex,
			problem.Name,
		),
	)

	if err != nil {
		slog.Error(
			"failed to delete solution from github",
			"telegram_user_id", userID,
			"daily_problem_id", problem.ID,
			"path", path,
			"error", err,
		)

		return s.sendError(
			ctx,
			message.Chat.ID,
			"Failed to delete your solution from GitHub.",
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
