# Build stage — explicitly target AMD64 (Hetzner's architecture)
FROM --platform=linux/amd64 golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# No GOARCH flag needed — the platform declaration above handles it
RUN CGO_ENABLED=1 GOOS=linux go build -o hookdrop ./main.go

# Run stage
FROM --platform=linux/amd64 alpine:3.21
RUN apk add --no-cache ca-certificates sqlite

WORKDIR /app
COPY --from=builder /app/hookdrop .

RUN mkdir -p /data

EXPOSE 8080
CMD ["./hookdrop"]