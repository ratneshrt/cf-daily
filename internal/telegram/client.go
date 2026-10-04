package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// maxMessageLength is the Telegram limit for a single sendMessage text.
const maxMessageLength = 4096

const maxSendAttempts = 3

type Client struct {
	token      string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// SendMessage sends text to a chat, splitting it across several messages when
// it exceeds the Telegram length limit. The returned SentMessage is the first
// chunk, which is the message users reply to.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) (*SentMessage, error) {
	chunks := splitMessage(text, maxMessageLength)

	var first *SentMessage

	for index, chunk := range chunks {
		sent, err := c.sendOne(ctx, chatID, chunk)

		if err != nil {
			return nil, fmt.Errorf(
				"sending telegram message chunk %d/%d: %w",
				index+1,
				len(chunks),
				err,
			)
		}

		if first == nil {
			first = sent
		}
	}

	if first == nil {
		return nil, errors.New("telegram message was empty")
	}

	return first, nil
}

func (c *Client) sendOne(ctx context.Context, chatID int64, text string) (*SentMessage, error) {
	payload := struct {
		ChatID int64  `json:"chat_id"`
		Text   string `json:"text"`
	}{
		ChatID: chatID,
		Text:   text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling telegram message: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token)

	var lastErr error

	for attempt := 1; attempt <= maxSendAttempts; attempt++ {
		sent, retryAfter, err := c.attemptSend(ctx, url, body)

		if err == nil {
			return sent, nil
		}

		lastErr = err

		if retryAfter < 0 || attempt == maxSendAttempts {
			return nil, err
		}

		if retryAfter == 0 {
			retryAfter = time.Duration(attempt) * time.Second
		}

		slog.Warn(
			"telegram send failed, retrying",
			"attempt",
			attempt,
			"retry_in",
			retryAfter,
			"error",
			err,
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryAfter):
		}
	}

	return nil, lastErr
}

// attemptSend performs one request. A retryAfter of -1 means the failure is
// permanent and must not be retried; 0 means retry with a default backoff.
func (c *Client) attemptSend(ctx context.Context, url string, body []byte) (*SentMessage, time.Duration, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)

	if err != nil {
		return nil, -1, fmt.Errorf("creating telegram request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Network level failures are worth another attempt.
		return nil, 0, fmt.Errorf("sending telegram req: %w", err)
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("reading telegram response: %w", err)
	}

	var result APIResponse[SentMessage]

	// A non-JSON body is possible for gateway errors, so decode failures are
	// reported using the status code instead.
	decodeErr := json.Unmarshal(responseBody, &result)

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := time.Duration(result.Parameters.RetryAfter) * time.Second

		if retryAfter <= 0 {
			retryAfter = time.Second
		}

		return nil, retryAfter, fmt.Errorf(
			"telegram rate limited: %s",
			result.Description,
		)
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, 0, fmt.Errorf(
			"telegram returned status %d",
			resp.StatusCode,
		)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, -1, fmt.Errorf(
			"telegram returned status %d: %s",
			resp.StatusCode,
			result.Description,
		)
	}

	if decodeErr != nil {
		return nil, -1, fmt.Errorf(
			"decoding telegram response: %w",
			decodeErr,
		)
	}

	if !result.OK {
		return nil, -1, fmt.Errorf(
			"telegram API error: %s",
			result.Description,
		)
	}

	return &result.Result, 0, nil
}

// splitMessage breaks text into chunks of at most limit characters, preferring
// to break on a newline so code blocks stay readable.
func splitMessage(text string, limit int) []string {
	runes := []rune(text)

	if len(runes) <= limit {
		return []string{text}
	}

	var chunks []string

	for len(runes) > limit {
		cut := limit

		for i := limit - 1; i > limit/2; i-- {
			if runes[i] == '\n' {
				cut = i + 1
				break
			}
		}

		chunks = append(chunks, string(runes[:cut]))
		runes = runes[cut:]
	}

	if len(runes) > 0 {
		chunks = append(chunks, string(runes))
	}

	return chunks
}
