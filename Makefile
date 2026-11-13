BIN := bin/pigura
DIST := internal/web/dist
SCAN_BASE ?= /srv/photos

.DEFAULT_GOAL := help
.PHONY: help web dist-clean build arm64 arm64-bin run test dup check docker clean

help: ## List targets
	@awk -F':.*## ' '/^[a-zA-Z0-9_-]+:.*## /{printf "  \033[1m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

web: ## Build the frontend (web/ -> internal/web/dist)
	cd web && npm install --no-audit --no-fund && npm run build

# Empty dist (keeps .gitkeep) so stale hashed assets never get embedded.
dist-clean:
	find $(DIST) -mindepth 1 ! -name .gitkeep -delete

build: dist-clean web ## Build a local binary (frontend embedded) -> bin/pigura
	go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/pigura

# Go only (static Linux ARM64). The frontend embedded is whatever is in $(DIST) right now.
arm64-bin:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN)-linux-arm64 ./cmd/pigura

arm64: dist-clean web arm64-bin ## Build a static Linux ARM64 binary -> bin/pigura-linux-arm64

run: build ## Run locally: make run LIBRARY=/path/to/photos SCAN_BASE=/path/to
	$(BIN) -library "$(LIBRARY)" -scan-base "$(SCAN_BASE)" -scan

test: ## go vet + go test
	go vet ./... && go test ./...

dup: ## Code duplication check (jscpd; threshold in .jscpd.json)
	npx --yes jscpd@5.4.3 --config .jscpd.json

check: test dup ## Everything CI runs

docker: ## Build the Docker image locally (pigura:dev)
	docker build -t pigura:dev .

clean:
	rm -rf bin
