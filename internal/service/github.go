package service

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

type GitHubContentResponse struct {
	SHA     string `json:"sha"`
	Content struct {
		SHA string `json:"sha"`
	} `json:"content"`
}

type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

type GitHubRepository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
	CloneURL string `json:"clone_url"`
	Private  bool   `json:"private"`
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
			"getting github user: %w",
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

func (s *GitHubService) CreateRepository(ctx context.Context, accessToken string) (*GitHubRepository, error) {

	payload := map[string]any{
		"name":        s.repositoryName,
		"description": "Codeforces solutions managed by 8pieces",
		"private":     false,
		"auto_init":   true,
	}

	body, err := json.Marshal(payload)

	if err != nil {
		return nil, fmt.Errorf(
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
		return nil, fmt.Errorf(
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
		return nil, fmt.Errorf(
			"creating github repository: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {

		var repo GitHubRepository

		if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
			return nil, fmt.Errorf("decodind created github repo: %w", err)
		}

		return &repo, nil
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		bodyBytes, _ := io.ReadAll(resp.Body)

		var githubError struct {
			Message string `json:"message"`
			Errors  []struct {
				Resource string `json:"resource"`
				Code     string `json:"code"`
				Field    string `json:"field"`
				Message  string `json:"message"`
			} `json:"errors"`
		}

		if err := json.Unmarshal(bodyBytes, &githubError); err == nil {
			for _, githubErr := range githubError.Errors {
				if githubErr.Field == "name" && githubErr.Message == "name already exists on this account" {
					return s.GetExistingRepository(ctx, accessToken)
				}
			}
		}

		return nil, fmt.Errorf(
			"github repository already exists or validation failed: %s",
			string(bodyBytes),
		)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)

	return nil, fmt.Errorf(
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

func (s *GitHubService) GetExistingRepository(ctx context.Context, access_token string) (*GitHubRepository, error) {
	user, err := s.GetAuthenticatedUser(ctx, access_token)

	if err != nil {
		return nil, fmt.Errorf("getting authenticated github user: %w", err)
	}

	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s", url.PathEscape(user.Login), url.PathEscape(s.repositoryName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)

	if err != nil {
		return nil, fmt.Errorf("creating existing repository request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+access_token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")

	resp, err := s.httpClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("getting existing gtihub repo: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodybytes, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("existing gtihub repo lookup failed: status=%d body=%s", resp.StatusCode, string(bodybytes))
	}

	var repo GitHubRepository

	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return nil, fmt.Errorf("decoding existing github repo: %w", err)
	}

	return &repo, nil
}

func buildSolutionPath(contestID int, problemIndex string, problemName string) string {
	name := sanitizerProblemName(problemName)

	return fmt.Sprintf(
		"%d/%d%s-%s/solution.cpp",
		contestID,
		contestID,
		problemIndex,
		name,
	)
}

func sanitizerProblemName(name string) string {
	name = strings.TrimSpace(name)

	var builder strings.Builder

	lastWasDash := false

	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastWasDash = false

		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
			lastWasDash = false

		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastWasDash = false

		default:
			if !lastWasDash && builder.Len() > 0 {
				builder.WriteRune('-')
				lastWasDash = true
			}
		}
	}

	return strings.Trim(builder.String(), "-")
}

func (s *GitHubService) CreateOrUpdateFile(ctx context.Context, installationID int64, owner string, path string, content string, commitMessage string) error {
	token, err := s.GetInstallationToken(ctx, installationID)

	if err != nil {
		return fmt.Errorf("getting github installation token: %w", err)
	}

	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", url.PathEscape(owner), url.PathEscape(s.repositoryName), path)

	var existingSHA string

	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)

	if err != nil {
		return fmt.Errorf("creating github file lookup request: %w", err)
	}

	getReq.Header.Set("Authorization", "Bearer "+token)
	getReq.Header.Set("Accept", "application/vnd.github+json")
	getReq.Header.Set("X-GitHub-Api-Version", "2026-03-10")

	getResp, err := s.httpClient.Do(getReq)

	if err != nil {
		return fmt.Errorf("checking github file: %w", err)
	}

	if getResp.StatusCode == http.StatusOK {

		var existing GitHubContentResponse

		err := json.NewDecoder(getReq.Body).Decode(&existing)

		getResp.Body.Close()

		if err != nil {
			return fmt.Errorf("decoding existing github file: %w", err)
		}

		existingSHA = existing.SHA
	} else if getResp.StatusCode == http.StatusNotFound {
		getResp.Body.Close()
	} else {
		bodyBytes, _ := io.ReadAll(getResp.Body)

		getResp.Body.Close()

		return fmt.Errorf("github file lookup failed: status=%d body=%s", getResp.StatusCode, string(bodyBytes))
	}

	encodedContent := base64.StdEncoding.EncodeToString([]byte(content))

	payload := map[string]any{
		"message": commitMessage,
		"content": encodedContent,
	}

	if existingSHA != "" {
		payload["sha"] = existingSHA
	}

	body, err := json.Marshal(payload)

	if err != nil {
		return fmt.Errorf("marshalling github file payload: %w", err)
	}

	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))

	if err != nil {
		return fmt.Errorf("creating github file req: %w", err)
	}

	putReq.Header.Set("Authorization", "Bearer "+token)
	putReq.Header.Set("Accept", "application/vnd.github+json")
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("X-GitHub-Api-Version", "2026-03-10")

	putResp, err := s.httpClient.Do(putReq)

	if err != nil {
		return fmt.Errorf("writing github file: %w", err)
	}

	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusCreated && putResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(putResp.Body)

		return fmt.Errorf("github file write failed: status=%d body=%s", putResp.StatusCode, string(bodyBytes))
	}

	return nil
}
