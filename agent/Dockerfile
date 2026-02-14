# Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o agent .

# Użyj base zamiast nonroot
FROM gcr.io/distroless/static-debian12:latest

WORKDIR /app

COPY --from=builder /build/agent /app/agent

ENTRYPOINT ["/app/agent"]
CMD ["stats"]