package service

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateGitHubState() (string, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
