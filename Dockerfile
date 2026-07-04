# --- Stage 1: Build ---
FROM golang:1.22-alpine AS builder

# Устанавливаем необходимые зависимости для сборки CGO (если нужно)
RUN apk add --no-cache git build-base

WORKDIR /app

# Кэшируем модули: копируем только go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник с флагами оптимизации (убираем отладочную информацию)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/api/main.go

# --- Stage 2: Final ---
FROM alpine:latest

# Добавляем сертификаты для общения по HTTPS
RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Копируем только скомпилированный бинарник из билдера
COPY --from=builder /app/main .

# Копируем конфигурационные файлы или миграции, если они нужны
# COPY --from=builder /app/migrations ./migrations

# Открываем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./main"]
