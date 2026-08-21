package service

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type GitHubService struct {
	appID          int64
	clientID       string
	privateKey     *rsa.PrivateKey
	callbackURL    string
	httpClient     *http.Client
	repositoryName string
}

type githubInstallationResponse struct {
	ID      int64 `json:"id"`
	Account struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
	} `json:"account"`
}

type installationTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewGitHubService(appID int64, clientID string, privateKeyPEM string, callbackURL string, repositoryName string) *GitHubService {
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))

	if err != nil {
		return nil
	}

	return &GitHubService{
		appID:       appID,
		clientID:    clientID,
		privateKey:  key,
		callbackURL: callbackURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		repositoryName: repositoryName,
	}
}

func (s *GitHubService) generateAppJWT() (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"iat": now.Add(-30 * time.Second).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": s.appID,
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodES256,
		claims,
	)

	signed, err := token.SignedString(s.privateKey)

	if err != nil {
		return "", fmt.Errorf(
			"signing github app jwt. %w",
			err,
		)
	}

	return signed, nil
}

func (s *GitHubService) GetUserInstallation(ctx context.Context, username string) (int64, error) {

	appJWT, err := s.generateAppJWT()
	if err != nil {
		return 0, err
	}

	endpoint := fmt.Sprintf(
		"https://api.github.com/users/%s/installation",
		url.PathEscape(username),
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)

	if err != nil {
		return 0, fmt.Errorf("creating github installation request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	req.Header.Set("Authorization", "Bearer "+appJWT)

	req.Header.Set("X-Github-Api-Version", "2026-03-10")

	resp, err := s.httpClient.Do(req)

	if err != nil {
		return 0, fmt.Errorf(
			"getting github installation: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf(
			"github installation lookup returned status %d",
			resp.StatusCode,
		)
	}

	var result githubInstallationResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf(
			"decoding github installation: %w",
			err,
		)
	}

	return result.ID, nil
}

func (s *GitHubService) GetInstallationToken(ctx context.Context, installationID int64) (string, error) {
	appJWT, err := s.generateAppJWT()
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("https://api/github.com/app/installations/%d/access_tokens", installationID)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		nil,
	)

	if err != nil {
		return "", fmt.Errorf(
			"creating installation token request: %w",
			err,
		)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("X-Github-Api-Version", "2026-03-10")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf(
			"getting installtion token: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf(
			"github installation token returned status %d",
			resp.StatusCode,
		)
	}

	var result installationTokenResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf(
			"decoding installation token: %w",
			err,
		)
	}

	return result.Token, nil
}

func (s *GitHubService) InstallationURL(state string) string {
	values := url.Values{}

	values.Set(
		"state",
		state,
	)

	return "https://github.com/apps/8pieces/installations/new?" + values.Encode()
}
