package random

import (
	"crypto/rand"
	"fmt"
)

type Generator struct{}

func New() *Generator {
	return &Generator{}
}

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func (g *Generator) Generate(size int) (string, error) {
	alias := make([]byte, size)

	if _, err := rand.Read(alias); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	for i := range alias {
		alias[i] = charset[alias[i]%byte(len(charset))]
	}
	return string(alias), nil
}
