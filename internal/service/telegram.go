package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ratneshrt/cf-daily/internal/model"
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
		slog.Debug("telegram update has no message")
		return nil
	}

	if update.Message.From == nil {
		slog.Debug("telegram message has no sender")
		return nil
	}

	slog.Info(
		"telegram update received",
		"update_id",
		update.UpdateID,
		"message_id",
		update.Message.MessageID,
		"user_id",
		update.Message.From.ID,
		"has_reply",
		update.Message.ReplyToMessage != nil,
	)

	err := s.route(ctx, update.Message)

	if err == nil {
		return nil
	}

	// The webhook always acknowledges, so an internal failure would otherwise
	// be silent from the chat's point of view.
	slog.Error(
		"failed to handle telegram command",
		"update_id",
		update.UpdateID,
		"user_id",
		update.Message.From.ID,
		"error",
		err,
	)

	if notifyErr := s.sendMessage(
		ctx,
		update.Message.Chat.ID,
		"Something went wrong handling that command. Please try again.",
	); notifyErr != nil {
		slog.Error(
			"failed to report command failure to user",
			"error",
			notifyErr,
		)
	}

	return err
}

func (s *TelegramService) route(ctx context.Context, message *telegram.Message) error {
	text := strings.TrimSpace(message.Text)

	switch {
	case strings.HasPrefix(text, "/start"):
		return s.handleStart(ctx, message)

	case strings.HasPrefix(text, "/stop"):
		return s.handleStop(ctx, message)

	case strings.HasPrefix(text, "/help"):
		return s.handleHelp(ctx, message)

	case strings.HasPrefix(text, "/submit"):
		return s.handleSubmit(ctx, message)

	case strings.HasPrefix(text, "/edit"):
		return s.handleEdit(ctx, message)

	case strings.HasPrefix(text, "/delete"):
		return s.handleDelete(ctx, message)

	case strings.HasPrefix(text, "/connect"):
		return s.handleConnectGitHub(ctx, message)

	default:
		slog.Debug("unknown telegram command", "text", text)
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

	// The state is a single use credential, so it is never logged.
	err = s.githubStateRepository.Create(
		ctx,
		state,
		userID,
		expiresAt,
	)

	if err != nil {
		return fmt.Errorf(
			"saving github connection state: %w",
			err,
		)
	}

	slog.Info(
		"github connect state saved",
		"telegram_user_id", userID,
		"expires_at", expiresAt,
	)

	authURL := s.githubService.InstallationURL(state)

	return s.sendMessage(
		ctx,
		message.Chat.ID,
		"🔗 Connect your GitHub account\n\n"+
			"Install the app on your GitHub account and "+
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
		}
	} else {
		// Also refreshes the chat id, username and first name, and brings back
		// a user who previously ran /stop.
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

	text := fmt.Sprintf(
		"👋 Welcome %s!\n\n"+
			"You are now registered for CF Daily.\n\n"+
			"Use /help to see available commands.",
		user.FirstName,
	)

	return s.sendMessage(ctx, user.ChatID, text)
}

func (s *TelegramService) handleStop(ctx context.Context, message *telegram.Message) error {
	user, err := s.userRepository.GetByTelegramUserID(ctx, message.From.ID)

	if err != nil {
		return fmt.Errorf("checking telegram user: %w", err)
	}

	if user == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"You are not registered. Use /start first.",
		)
	}

	if err := s.userRepository.Deactivate(ctx, message.From.ID); err != nil {
		return fmt.Errorf("deactivating telegram user: %w", err)
	}

	return s.sendMessage(
		ctx,
		message.Chat.ID,
		"Daily problems paused. Use /start to resume.\n\n"+
			"Your saved solutions are untouched.",
	)
}

func (s *TelegramService) handleHelp(ctx context.Context, message *telegram.Message) error {
	text := `CF Daily Commands

/start - Register, or resume daily problems
/stop - Pause daily problems
/connect - Connect your GitHub account
/help - Show available commands

/submit
<your code> -> save your solution

/edit
<new code> -> replace your solution

/delete -> delete your solution

Reply to a daily problem message to target that
problem. Without a reply, the most recent problem
sent to you is used.
`

	return s.sendMessage(ctx, message.Chat.ID, text)
}

// resolveProblemMessage works out which daily problem a command refers to.
//
// It first tries the message the user replied to, so replying to an older daily
// problem message still attaches the code to that older problem. When the
// replied-to message is not a recorded daily problem message - a reply to the
// nightly reminder, a reply to one of the bot's own answers, or no reply at all
// - it falls back to the most recent daily problem sent to that user.
func (s *TelegramService) resolveProblemMessage(ctx context.Context, message *telegram.Message) (*model.TelegramProblemMessage, error) {
	userID := message.From.ID

	if message.ReplyToMessage != nil {
		problemMessage, err := s.problemMessageRepository.GetByMessageID(
			ctx,
			userID,
			message.ReplyToMessage.MessageID,
		)

		if err != nil {
			return nil, fmt.Errorf("getting problem message: %w", err)
		}

		if problemMessage != nil {
			return problemMessage, nil
		}

		slog.Info(
			"replied-to message is not a daily problem message, falling back to latest",
			"telegram_user_id", userID,
			"reply_message_id", message.ReplyToMessage.MessageID,
		)
	}

	problemMessage, err := s.problemMessageRepository.GetLatestByUser(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("getting latest problem message: %w", err)
	}

	return problemMessage, nil
}

// connectedUser loads the sender and verifies their GitHub connection.
func (s *TelegramService) connectedUser(ctx context.Context, userID int64) (*model.TelegramUser, error) {
	user, err := s.userRepository.GetByTelegramUserID(ctx, userID)

	if err != nil {
		return nil, fmt.Errorf("getting telegram user: %w", err)
	}

	if user == nil || user.GithubUsername == nil || user.GithubInstallationID == nil {
		return nil, nil
	}

	return user, nil
}

// problemFor loads the problem a resolved problem message points at.
func (s *TelegramService) problemFor(ctx context.Context, problemMessage *model.TelegramProblemMessage) (*model.DailyProblem, error) {
	problem, err := s.dailyProblemRepository.GetByID(
		ctx,
		problemMessage.DailyProblemID,
	)

	if err != nil {
		return nil, fmt.Errorf("getting daily problem: %w", err)
	}

	return problem, nil
}

func (s *TelegramService) handleSubmit(ctx context.Context, message *telegram.Message) error {
	parts := strings.SplitN(message.Text, "\n", 2)

	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
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

	problemMessage, err := s.resolveProblemMessage(ctx, message)

	if err != nil {
		return err
	}

	if problemMessage == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"No daily problem has been sent to you yet, so there is nothing to submit against.",
		)
	}

	problem, err := s.problemFor(ctx, problemMessage)

	if err != nil {
		return err
	}

	if problem == nil {
		return s.sendError(ctx, message.Chat.ID, "Daily problem not found.")
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
			fmt.Sprintf(
				"⚠️ You already submitted a solution for %s.\n\n"+
					"Use /edit to replace it.",
				problem.Name,
			),
		)
	}

	user, err := s.connectedUser(ctx, userID)

	if err != nil {
		return err
	}

	if user == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"Please connect GitHub first using /connect.",
		)
	}

	language := detectLanguage(code)

	path := buildSolutionPath(
		problem.ContestID,
		problem.ProblemIndex,
		problem.Name,
		language,
	)

	// The submission is recorded first and rolled back if the push fails, so a
	// failed command never leaves a commit without a matching row.
	if _, err := s.submissionRepository.Create(
		ctx,
		userID,
		problemMessage.DailyProblemID,
		code,
		language,
	); err != nil {
		return fmt.Errorf("creating submission: %w", err)
	}

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

		if rollbackErr := s.submissionRepository.Delete(
			ctx,
			userID,
			problemMessage.DailyProblemID,
		); rollbackErr != nil {
			slog.Error(
				"failed to roll back submission after github failure",
				"telegram_user_id", userID,
				"daily_problem_id", problem.ID,
				"error", rollbackErr,
			)
		}

		return s.sendError(
			ctx,
			message.Chat.ID,
			"Failed to push your solution to GitHub. Nothing was saved, please try again.",
		)
	}

	return s.sendMessage(
		ctx,
		message.Chat.ID,
		fmt.Sprintf(
			"✅ Your solution for %s has been saved!",
			problem.Name,
		),
	)
}

func (s *TelegramService) handleEdit(ctx context.Context, message *telegram.Message) error {
	parts := strings.SplitN(message.Text, "\n", 2)

	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
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

	problemMessage, err := s.resolveProblemMessage(ctx, message)

	if err != nil {
		return err
	}

	if problemMessage == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"No daily problem has been sent to you yet, so there is nothing to edit.",
		)
	}

	problem, err := s.problemFor(ctx, problemMessage)

	if err != nil {
		return err
	}

	if problem == nil {
		return s.sendError(ctx, message.Chat.ID, "Daily problem not found.")
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
			fmt.Sprintf(
				"You don't have a submission for %s yet.\n\n"+
					"Use /submit first.",
				problem.Name,
			),
		)
	}

	user, err := s.connectedUser(ctx, userID)

	if err != nil {
		return err
	}

	if user == nil {
		return s.sendError(ctx, message.Chat.ID, "Please connect GitHub first using /connect.")
	}

	previousLanguage := submissionLanguage(existing.Language)

	previousPath := buildSolutionPath(
		problem.ContestID,
		problem.ProblemIndex,
		problem.Name,
		previousLanguage,
	)

	language := detectLanguage(code)

	path := buildSolutionPath(
		problem.ContestID,
		problem.ProblemIndex,
		problem.Name,
		language,
	)

	if _, err := s.submissionRepository.Update(
		ctx,
		userID,
		problemMessage.DailyProblemID,
		code,
		language,
	); err != nil {
		return fmt.Errorf("updating submission: %w", err)
	}

	err = s.githubService.CreateOrUpdateFile(
		ctx,
		*user.GithubInstallationID,
		*user.GithubUsername,
		path,
		code,
		fmt.Sprintf(
			"Update solution for %d%s - %s",
			problem.ContestID,
			problem.ProblemIndex,
			problem.Name,
		),
	)

	if err != nil {
		slog.Error(
			"failed to update solution on github",
			"telegram_user_id", userID,
			"daily_problem_id", problem.ID,
			"path", path,
			"error", err,
		)

		if _, rollbackErr := s.submissionRepository.Update(
			ctx,
			userID,
			problemMessage.DailyProblemID,
			existing.Code,
			previousLanguage,
		); rollbackErr != nil {
			slog.Error(
				"failed to roll back submission after github failure",
				"telegram_user_id", userID,
				"daily_problem_id", problem.ID,
				"error", rollbackErr,
			)
		}

		return s.sendError(
			ctx,
			message.Chat.ID,
			"Failed to update your solution on GitHub. Your saved solution was left unchanged.",
		)
	}

	// A different language means a different file name, so the old file would
	// otherwise be left behind.
	if path != previousPath {
		if err := s.githubService.DeleteFile(
			ctx,
			*user.GithubInstallationID,
			*user.GithubUsername,
			previousPath,
			fmt.Sprintf(
				"Remove superseded solution for %d%s - %s",
				problem.ContestID,
				problem.ProblemIndex,
				problem.Name,
			),
		); err != nil {
			slog.Warn(
				"failed to remove superseded solution file",
				"telegram_user_id", userID,
				"path", previousPath,
				"error", err,
			)
		}
	}

	return s.sendMessage(
		ctx,
		message.Chat.ID,
		fmt.Sprintf(
			"✏️ Your solution for %s has been updated!",
			problem.Name,
		),
	)
}

func (s *TelegramService) handleDelete(ctx context.Context, message *telegram.Message) error {
	userID := message.From.ID

	problemMessage, err := s.resolveProblemMessage(ctx, message)

	if err != nil {
		return err
	}

	if problemMessage == nil {
		return s.sendError(
			ctx,
			message.Chat.ID,
			"No daily problem has been sent to you yet, so there is nothing to delete.",
		)
	}

	problem, err := s.problemFor(ctx, problemMessage)

	if err != nil {
		return err
	}

	if problem == nil {
		return s.sendError(ctx, message.Chat.ID, "Daily problem not found.")
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
			fmt.Sprintf(
				"You don't have a submitted solution for %s.",
				problem.Name,
			),
		)
	}

	user, err := s.connectedUser(ctx, userID)

	if err != nil {
		return err
	}

	if user == nil {
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
		submissionLanguage(submission.Language),
	)

	// The file is removed first: if the database delete then fails the code is
	// still recorded, which is the recoverable order.
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
		fmt.Sprintf(
			"🗑️ Your solution for %s has been deleted.",
			problem.Name,
		),
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
