# syntax=docker/dockerfile:1

FROM golang:1.24.4-bookworm AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY claudecode ./claudecode
COPY cmd ./cmd

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /out/claudecode-runner ./cmd/claudecode-runner

FROM node:20-bookworm-slim

# Install Claude Code CLI (runtime only, minimize cache size)
RUN npm install -g --omit=dev @anthropic-ai/claude-code \
    && npm cache clean --force

COPY --from=builder /out/claudecode-runner /usr/local/bin/claudecode-runner

ENTRYPOINT ["claudecode-runner"]
