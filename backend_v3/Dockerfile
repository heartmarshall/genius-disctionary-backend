# Multi-stage build для оптимизации размера образа

# Stage 1: Builder - компиляция приложения
FROM golang:1.24-alpine AS builder

# Устанавливаем необходимые инструменты
RUN apk add --no-cache git make

# Устанавливаем рабочую директорию
WORKDIR /build

# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Генерируем GraphQL код (если нужно)
RUN make generate || true

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o server \
    ./cmd/server

# Stage 2: Runtime - минимальный образ для запуска
FROM alpine:latest

# Устанавливаем CA сертификаты и wget для healthcheck
RUN apk --no-cache add ca-certificates tzdata wget

# Создаем непривилегированного пользователя для безопасности
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Копируем бинарник из builder stage
COPY --from=builder /build/server .

# Копируем миграции (если нужно запускать их внутри контейнера)
COPY --from=builder /build/migrations ./migrations

# Меняем владельца файлов
RUN chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Открываем порт
EXPOSE 8080

# Healthcheck для мониторинга (можно переопределить в docker-compose)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Запускаем приложение
CMD ["./server"]

