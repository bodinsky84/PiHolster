FROM golang:1.22-alpine AS builder

WORKDIR /src

# Download dependencies first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /piholsterd \
    ./cmd/piholsterd

# --- runtime stage ---
FROM alpine:3.19

# Create non-root user
RUN addgroup -g 999 piholster && \
    adduser -u 999 -G piholster -s /sbin/nologin -D piholster

WORKDIR /app

COPY --from=builder /piholsterd /app/piholsterd

# Blocklists are mounted at runtime via volume
RUN mkdir -p /app/blocklists && \
    chown -R piholster:piholster /app

USER piholster

EXPOSE 5300/udp
EXPOSE 5300/tcp

ENTRYPOINT ["/app/piholsterd"]
