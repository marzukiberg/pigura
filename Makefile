BIN        := bin/pigura
DIST       := internal/web/dist
PKG        := ./cmd/pigura
SCAN_BASE  ?= /srv/photos
LIBRARY    ?=

# Release-style flags, shared by every Go build target (static, no cgo).
GOFLAGS_BUILD := -trimpath -ldflags="-s -w"
export CGO_ENABLED ?= 0

.DEFAULT_GOAL := help
.PHONY: help web dist-clean build build-go cross arm64 run test fmt fmt-check dup check docker clean

help: ## List targets
	@awk -F':.*## ' '/^[a-zA-Z0-9_-]+:.*## /{printf "  \033[1m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

web: ## Build the frontend (web/ -> internal/web/dist)
	cd web && npm ci --no-audit --no-fund && npm run build

dist-clean: ## Empty the embedded frontend dir (keeps .gitkeep)
	find $(DIST) -mindepth 1 ! -name .gitkeep -delete

# Sequential on purpose (dist-clean must finish before web), so it is safe with `make -j`.
build: ## Build a local binary (frontend embedded) -> bin/pigura
	@$(MAKE) --no-print-directory dist-clean
	@$(MAKE) --no-print-directory web
	@$(MAKE) --no-print-directory build-go

build-go: ## Go only; embeds whatever is in internal/web/dist right now
	go build $(GOFLAGS_BUILD) -o $(BIN) $(PKG)

cross: ## Cross-compile: make cross GOOS=linux GOARCH=arm64 -> bin/pigura-<os>-<arch>
	@test -n "$(GOOS)" -a -n "$(GOARCH)" || { echo "usage: make cross GOOS=linux GOARCH=arm64"; exit 1; }
	@$(MAKE) --no-print-directory dist-clean
	@$(MAKE) --no-print-directory web
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS_BUILD) -o $(BIN)-$(GOOS)-$(GOARCH) $(PKG)

arm64: ## Shortcut: static Linux ARM64 binary -> bin/pigura-linux-arm64
	@$(MAKE) --no-print-directory cross GOOS=linux GOARCH=arm64

run: build ## Run locally: make run LIBRARY=/path/to/photos SCAN_BASE=/path/to
	$(BIN) $(if $(LIBRARY),-library "$(LIBRARY)") -scan-base "$(SCAN_BASE)" -scan

test: ## go vet + go test
	go vet ./...
	go test ./...

fmt: ## Format Go code
	gofmt -w cmd internal

fmt-check: ## Fail if Go code is not gofmt-formatted
	@out="$$(gofmt -l cmd internal)"; \
	if [ -n "$$out" ]; then echo "Not gofmt-formatted:"; echo "$$out"; exit 1; fi

dup: ## Code duplication check (jscpd; threshold in .jscpd.json)
	npx --yes jscpd@5.4.3 --config .jscpd.json

check: fmt-check test dup ## Everything CI runs

docker: ## Build the Docker image locally (pigura:dev)
	docker build -t pigura:dev .

clean: ## Remove build output
	rm -rf bin
	@$(MAKE) --no-print-directory dist-clean
