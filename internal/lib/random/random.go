package random

import (
	"crypto/rand"
	"fmt"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func NewRandomString(size int) (string, error) {
	alias := make([]byte, size)

	if _, err := rand.Read(alias); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	for i := range alias {
		alias[i] = charset[alias[i]%byte(len(charset))]
	}
	return string(alias), nil
}
