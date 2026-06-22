# syntax=docker/dockerfile:1

# ============================================================
# CodeTasker Backend — Multi-stage Dockerfile
# Stage 1: Build the Go binary
# Stage 2: Minimal production image
# ============================================================

# --- Stage 1: Builder ---
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache module downloads separately from source
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /codetasker ./cmd/server/main.go

# --- Stage 2: Production runtime ---
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=builder /codetasker /codetasker

EXPOSE 8080

ENTRYPOINT ["/codetasker"]
