.PHONY: dev client-build client-dev server-build server-dev build docker-build docker-up clean

GO := /opt/homebrew/bin/go
NPM := source ~/.nvm/nvm.sh && npm
SHELL := /bin/zsh

# Build client
client-build:
	cd client && $(NPM) run build

# Run client build in watch mode
client-dev:
	cd client && $(NPM) run dev

# Build server
server-build:
	cd server && $(GO) build -o bin/game-server ./cmd/server

# Run server in dev mode
server-dev:
	cd server && $(GO) run ./cmd/server

# Build everything
build: client-build server-build

# Dev mode: build client then run server
dev: client-build server-dev

# Docker
docker-build:
	docker compose build

docker-up:
	docker compose up

# Clean
clean:
	rm -rf client/dist server/bin
