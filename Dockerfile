FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git build-base

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/url-shortener/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Копируем бинарник
COPY --from=builder /app/main .
# Копируем конфиг и миграции
COPY ./config/local.yaml ./config/local.yaml

COPY ./migrations ./migrations

# Устанавливаем путь по умолчанию для контейнера
ENV CONFIG_PATH=/root/config/local.yaml

EXPOSE 8080
CMD ["./main"]