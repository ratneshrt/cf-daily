package service

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type GitHubService struct {
	appID          int64
	clientID       string
	clientSecret   string
	privateKey     *rsa.PrivateKey
	callbackURL    string
	httpClient     *http.Client
	repositoryName string
}

type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

type GitHubOAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
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

func NewGitHubService(appID int64, clientID string, clientSecret string, privateKeyPEM string, callbackURL string, repositoryName string) *GitHubService {
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))

	if err != nil {
		return nil
	}

	return &GitHubService{
		appID:        appID,
		clientID:     clientID,
		clientSecret: clientSecret,
		privateKey:   key,
		callbackURL:  callbackURL,
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
		jwt.SigningMethodRS256,
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

func (s *GitHubService) ExchangeCode(
	ctx context.Context,
	code string,
) (string, error) {

	params := url.Values{}

	params.Set("client_id", s.clientID)
	params.Set("client_secret", s.clientSecret)
	params.Set("code", code)
	params.Set("redirect_uri", s.callbackURL)

	endpoint := "https://github.com/login/oauth/access_token?" +
		params.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		nil,
	)

	if err != nil {
		return "", fmt.Errorf(
			"creating github oauth request: %w",
			err,
		)
	}

	req.Header.Set(
		"Accept",
		"application/json",
	)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf(
			"exchanging github code: %w",
			err,
		)
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf(
			"reading github oauth response: %w",
			err,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"github oauth returned status %d: %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	var result GitHubOAuthTokenResponse

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf(
			"decoding github oauth response: %w",
			err,
		)
	}

	if result.AccessToken == "" {
		return "", fmt.Errorf(
			"github oauth returned empty access token",
		)
	}

	return result.AccessToken, nil
}

func (s *GitHubService) GetAuthenticatedUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://api.github.com/user",
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"creating github user request: %w",
			err,
		)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"getting gtihub user: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"github user returned status %d",
			resp.StatusCode,
		)
	}

	var user GitHubUser

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf(
			"decoding github user: %w",
			err,
		)
	}

	return &user, nil
}

func (s *GitHubService) CreateRepository(ctx context.Context, accessToken string) error {
	// token, err := s.GetInstallationToken(ctx, installationID)

	// if err != nil {
	// 	return err
	// }

	payload := map[string]any{
		"name":        s.repositoryName,
		"description": "Codeforces solutions managed by 8pieces",
		"private":     false,
		"auto_init":   true,
	}

	body, err := json.Marshal(payload)

	if err != nil {
		return fmt.Errorf(
			"marshalling repository name: %w",
			err,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.github.com/user/repos",
		bytes.NewReader(body),
	)

	if err != nil {
		return fmt.Errorf(
			"creating repository req: %w",
			err,
		)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")

	resp, err := s.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf(
			"creating github repository: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		return nil
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		bodyBytes, _ := io.ReadAll(resp.Body)

		return fmt.Errorf(
			"github repository already exists or validation failed: %s",
			string(bodyBytes),
		)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)

	return fmt.Errorf(
		"github repository creation failed: status=%d body=%s",
		resp.StatusCode,
		string(bodyBytes),
	)
}

func (s *GitHubService) GetRepository(ctx context.Context, installationID int64, owner string) error {

	token, err := s.GetInstallationToken(ctx, installationID)

	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s", url.PathEscape(owner), url.PathEscape(s.repositoryName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)

	if err != nil {
		return fmt.Errorf("creating github repository lookup req: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	req.Header.Set("Accept", "application/vnd.github+json")

	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")

	resp, err := s.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("checking github repository: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("github respository %s/%s does not exist", owner, s.repositoryName)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)

	return fmt.Errorf("github repository lookup failed: status=%d body=%s", resp.StatusCode, string(bodyBytes))

}
