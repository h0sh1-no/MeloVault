# ── Stage 1: Build frontend ────────────────────────────────────────────────────
FROM node:20-alpine AS frontend
WORKDIR /web

COPY web/package.json web/package-lock.json* ./
RUN npm ci --no-audit --no-fund 2>/dev/null || npm install --no-audit --no-fund

COPY web/ ./
RUN npm run build

# ── Stage 2: Build Go backend ─────────────────────────────────────────────────
FROM golang:1.24-alpine3.21 AS builder
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum* ./
RUN go mod download || go mod tidy

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/melovault ./cmd/server

# ── Stage 3: Final image ──────────────────────────────────────────────────────
FROM alpine:3.21
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app && adduser -S app -G app

COPY --from=builder /out/melovault /app/melovault
COPY --from=frontend /web/dist /app/web

RUN mkdir -p /app/downloads \
    && touch /app/cookie.txt \
    && chown -R app:app /app

ENV TZ=Asia/Shanghai
ENV HOST=0.0.0.0
ENV PORT=5000
ENV DOWNLOADS_DIR=/app/downloads
ENV COOKIE_FILE=/app/cookie.txt
ENV STATIC_DIR=/app/web

EXPOSE 5000
USER app
CMD ["/app/melovault"]
