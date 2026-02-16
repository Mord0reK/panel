# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Install build dependencies for CGO (required by go-sqlite3)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server/main.go

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations

# Create data directory for SQLite
RUN mkdir /data

EXPOSE 8080

ENTRYPOINT ["./server"]
