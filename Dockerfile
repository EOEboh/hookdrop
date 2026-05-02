# Build stage
FROM golang:1.23-alpine AS builder

# gcc needed for go-sqlite3 (cgo)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Added GOARCH=amd64 — explicitly targets CPX22's x86-64 architecture
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o hookdrop ./main.go

# Run stage
# Pinned alpine version — matches the builder, no surprise updates on redeploy
FROM alpine:3.21
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/hookdrop .

RUN mkdir -p /data

EXPOSE 8080
CMD ["./hookdrop"]