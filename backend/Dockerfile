# ── Stage 1: build the server + seeder binaries ──────────────────────────────
FROM golang:1.23-alpine AS builder

# CGO_ENABLED=0 → fully static binary (runs on any base image, no libc deps).
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /build

# Cache deps first: copy only go.mod/go.sum, download, then copy the rest.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build both binaries; the canonical seed catalog is embedded in the seeder.
RUN go build -o /out/learnthor-server . \
 && go build -o /out/learnthor-seed ./cmd/seed \
 && go build -o /out/learnthor-account ./cmd/account

# ── Stage 2: minimal runtime image ───────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 appuser

WORKDIR /app

# Binaries
COPY --from=builder /out/learnthor-server /app/learnthor-server
COPY --from=builder /out/learnthor-seed  /app/learnthor-seed
COPY --from=builder /out/learnthor-account /app/learnthor-account


RUN chmod +x /app/learnthor-server /app/learnthor-seed /app/learnthor-account

USER appuser

EXPOSE 8080

# The server reads DATABASE_URL / JWT_SECRET / PORT from the environment
# (provided by docker-compose env_file or -e flags).
ENTRYPOINT ["/app/learnthor-server"]
