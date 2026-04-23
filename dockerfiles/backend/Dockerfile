# ── Stage 1: зависимости (общий слой для dev и prod) ─────────────────────────
FROM golang:1.24-alpine AS deps

WORKDIR /app
COPY backend/go.mod ./
RUN GONOSUMDB=* go mod download all

# ── Stage 2: dev — hot reload через air ──────────────────────────────────────
FROM deps AS dev

ENV GONOSUMDB=* GOFLAGS=-mod=mod

RUN go install github.com/air-verse/air@v1.61.7
EXPOSE 8080
CMD ["air"]

# ── Stage 3: сборка prod-бинарника ───────────────────────────────────────────
FROM deps AS builder

COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# ── Stage 4: prod — минимальный образ ────────────────────────────────────────
FROM alpine:3.21 AS prod

# ca-certificates — HTTPS к Telegram API
# tzdata — корректная таймзона
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /server ./server
EXPOSE 8080
CMD ["./server"]
