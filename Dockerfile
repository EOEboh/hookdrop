# Build stage — explicitly target AMD64 (Hetzner's architecture)
FROM --platform=linux/amd64 golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build the package, not a single file. `go build ./main.go` compiles only
# that one file, so any other file in package main (repair.go, for example)
# is left out and main.go fails on the symbols it defines.
RUN CGO_ENABLED=1 GOOS=linux go build -o hookdrop .

# Run stage
FROM --platform=linux/amd64 alpine:3.21
RUN apk add --no-cache ca-certificates sqlite

WORKDIR /app
COPY --from=builder /app/hookdrop .

RUN mkdir -p /data

EXPOSE 8080
CMD ["./hookdrop"]