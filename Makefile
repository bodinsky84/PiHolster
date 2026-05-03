.PHONY: dev test lint build build-web build-go reset image tidy docker-up docker-down docker-logs

# Default target
all: build

## dev: start backend (:8080) and frontend (:5173) in parallel
dev:
	@bash scripts/dev.sh

## test: run all tests (Go race detector + web)
test:
	go test -race ./...
	pnpm --filter web test

## lint: run all linters
lint:
	golangci-lint run
	pnpm --filter web lint

## build: build frontend then cross-compile for linux/arm/v7 (frontend is embedded via go:embed)
build: build-web build-go

## build-web: compile the SvelteKit app into apps/piholsterd/internal/api/dist/
build-web:
	pnpm --filter web build

## build-go: cross-compile piholsterd for linux/arm/v7 with embedded frontend
build-go:
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 \
		go build -trimpath -ldflags="-s -w" \
		-o dist/piholsterd-linux-arm-v7 \
		./apps/piholsterd/cmd/piholsterd

## reset: remove tmp/ and dist/
reset:
	rm -rf tmp/ dist/

## tidy: download dependencies and tidy go.mod/go.sum
tidy:
	cd apps/piholsterd && CGO_ENABLED=0 go mod tidy

## image: build Raspberry Pi SD-card image
image:
	@bash image/build.sh

## docker-up: build images and start all services
docker-up:
	docker compose up --build

## docker-down: stop and remove containers
docker-down:
	docker compose down

## docker-logs: follow logs from all running services
docker-logs:
	docker compose logs -f
