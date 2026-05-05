# ---- Stage 1: build web UI ----
FROM node:20-alpine AS web-builder

RUN corepack enable && corepack prepare pnpm@9 --activate

WORKDIR /repo

# Copy workspace manifests for dependency caching
COPY package.json pnpm-workspace.yaml* turbo.json* ./
COPY apps/web/package.json ./apps/web/
COPY packages/ ./packages/

RUN pnpm install --frozen-lockfile

COPY apps/web ./apps/web

# svelte.config.js outputs to apps/piholsterd/internal/api/dist
COPY apps/piholsterd/internal/api/dist ./apps/piholsterd/internal/api/dist

RUN pnpm --filter web build

# ---- Stage 2: build Go binary ----
FROM golang:1.22-alpine AS go-builder

WORKDIR /src

# Copy go workspace and both modules for dependency caching
COPY go.work go.work.sum* ./
COPY apps/piholsterd/go.mod apps/piholsterd/go.sum ./apps/piholsterd/
COPY apps/piholster-arpd/go.mod apps/piholster-arpd/go.sum ./apps/piholster-arpd/

RUN cd apps/piholsterd && CGO_ENABLED=0 go mod download

COPY apps/piholsterd ./apps/piholsterd
COPY apps/piholster-arpd ./apps/piholster-arpd

# Copy the built web UI from stage 1
COPY --from=web-builder /repo/apps/piholsterd/internal/api/dist ./apps/piholsterd/internal/api/dist

RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" \
    -o /piholsterd \
    ./apps/piholsterd/cmd/piholsterd

# ---- Stage 3: minimal runtime ----
FROM alpine:3.19

RUN addgroup -g 999 piholster && \
    adduser -u 999 -G piholster -s /sbin/nologin -D piholster

WORKDIR /app

COPY --from=go-builder /piholsterd /app/piholsterd

RUN mkdir -p /app/blocklists /app/data && \
    chown -R piholster:piholster /app

USER piholster

# DNS
EXPOSE 5300/udp
EXPOSE 5300/tcp
# HTTP (dev mode, no TLS)
EXPOSE 8080

ENTRYPOINT ["/app/piholsterd"]
