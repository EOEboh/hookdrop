# Build stage. Architecture is left to the builder: `docker build` follows the
# host, and buildx follows --platform. Pinning it here (it was linux/amd64 for
# Hetzner) silently overrides the platform the deploy workflow asks for.
FROM golang:1.23-alpine AS builder

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
FROM alpine:3.21
RUN apk add --no-cache ca-certificates sqlite

WORKDIR /app
COPY --from=builder /app/hookdrop .

RUN mkdir -p /data

EXPOSE 8080
CMD ["./hookdrop"]