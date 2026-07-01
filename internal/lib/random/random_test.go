package random_test

import (
	"strings"
	"testing"
	"url-shortener/internal/lib/random"
)

func TestNewRandomString(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{
			name: "size = 1",
			size: 1,
		},
		{
			name: "size = 5",
			size: 5,
		},
		{
			name: "size = 10",
			size: 10,
		},
		{
			name: "size = 20",
			size: 20,
		},
		{
			name: "size = 30",
			size: 30,
		},
		{
			name: "edge case: size = 0",
			size: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str1, err := random.NewRandomString(tt.size)

			// 1. Проверяем, что нет ошибки генерации
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// 2. Проверяем, что длина совпадает с ожидаемой
			if len(str1) != tt.size {
				t.Errorf("expected size %d, got %d", tt.size, len(str1))
			}

			// Проверки если строка не пустая
			if tt.size > 0 {
				// 3. Проверяем, что используются только разрешенные символы
				charset := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
				for _, char := range str1 {
					if !strings.ContainsRune(charset, char) {
						t.Errorf("unexpected character %q in generated string", char)
					}
				}

				// 4. Проверяем "случайность"
				// Генерируем вторую строку и сравниваем с первой
				// Для длины 1 или 2 шанс случайного совпадения слишком велик, поэтому проверяем только для size > 2
				if tt.size > 2 {
					str2, err := random.NewRandomString(tt.size)
					if err != nil {
						t.Fatalf("unexpected error on second generation: %v", err)
					}

					if str1 == str2 {
						t.Errorf("generated strings are identical, expected randomness: %q == %q", str1, str2)
					}
				}
			}
		})
	}
}
