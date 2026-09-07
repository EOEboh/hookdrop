# Build stage. No cgo, so no C toolchain and no architecture pinning: the
# binary is pure Go and cross-compiles for free. This is only true while the
# database is libSQL/Turso over the network — the local-file driver
# (mattn/go-sqlite3) needs cgo, and a CGO_ENABLED=0 binary given a file path
# fails at boot with "go-sqlite3 requires cgo to work. This is a stub".
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
# Build the package, not a single file. `go build ./main.go` compiles only
# that one file, so any other file in package main (repair.go, for example)
# is left out and main.go fails on the symbols it defines.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
	go build -ldflags="-s -w" -o hookdrop .

# Run stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/hookdrop .

RUN mkdir -p /data

EXPOSE 8080
CMD ["./hookdrop"]
