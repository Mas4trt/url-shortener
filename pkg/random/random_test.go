package random_test

import (
	"errors"
	"strings"
	"testing"
	"url-shortener/pkg/random"
)

// TestGenerator_Generate_Lengths проверяет генерацию строк различной длины.
func TestGenerator_Generate_Lengths(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"zero size", 0},
		{"size 1", 1},
		{"size 10", 10},
		{"large size", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := random.New(tt.size)

			got, err := g.Generate()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.size {
				t.Errorf("expected length %d, got %d", tt.size, len(got))
			}
		})
	}
}

// TestGenerator_Generate_Charset убеждается, что все символы сгенерированной строки
// действительно принадлежат заданному алфавиту (charset)
func TestGenerator_Generate_Charset(t *testing.T) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	size := 1000
	g := random.New(size)

	got, err := g.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, char := range got {
		if !strings.ContainsRune(charset, char) {
			t.Errorf("generated string contains invalid character: %q", char)
		}
	}
}

// TestGenerator_Generate_Uniqueness проверяет, что генератор не выдает дубликатов
func TestGenerator_Generate_Uniqueness(t *testing.T) {
	iterations := 1000
	size := 16
	g := random.New(size)
	seen := make(map[string]struct{}, iterations)

	for i := 0; i < iterations; i++ {
		got, err := g.Generate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, exists := seen[got]; exists {
			t.Fatalf("collision detected: %s was generated more than once", got)
		}
		seen[got] = struct{}{}
	}
}

// TestGenerator_Generate_NegativeSize проверяет ожидаемое поведение при передаче отрицательного размера
func TestGenerator_Generate_NegativeSize(t *testing.T) {
	g := random.New(-1)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on negative size, but code didn't panic")
		}
	}()

	_, _ = g.Generate()
}

var errMockRead = errors.New("mock read error")

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errMockRead
}

// TestGenerator_Generate_RandError проверяет обработку ошибки чтения из crypto/rand
func TestGenerator_Generate_RandError(t *testing.T) {
	g := random.New(10, random.WithReader(&errorReader{}))

	_, err := g.Generate()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	if !errors.Is(err, errMockRead) {
		t.Errorf("expected error to wrap errMockRead, got: %v", err)
	}
}
