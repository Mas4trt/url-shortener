package random

import (
	"crypto/rand"
	"fmt"
	"io"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

const maxRandomByte = 256 - (256 % len(charset))

type Generator struct {
	size          int
	entropyReader io.Reader
}

type Option func(*Generator)

func WithReader(r io.Reader) Option {
	return func(g *Generator) {
		g.entropyReader = r
	}
}

func New(size int, opts ...Option) *Generator {
	g := &Generator{
		size:          size,
		entropyReader: rand.Reader,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

func (g *Generator) Generate() (string, error) {
	result := make([]byte, g.size)
	random := make([]byte, g.size)

	for i := 0; i < g.size; {
		if _, err := io.ReadFull(g.entropyReader, random); err != nil {
			return "", fmt.Errorf("generate random bytes: %w", err)
		}

		for _, b := range random {
			if int(b) >= maxRandomByte {
				continue
			}

			result[i] = charset[int(b)%len(charset)]
			i++

			if i == g.size {
				break
			}
		}
	}

	return string(result), nil
}
