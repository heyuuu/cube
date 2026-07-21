.DEFAULT_GOAL := build

# 从 git 收集构建期信息（与 version/version.go 配合，通过 ldflags 注入）
VERSION    := $(shell git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD)
COMMIT     := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

VERSION_PKG := github.com/heyuuu/cube/version
LDFLAGS := \
  -X $(VERSION_PKG).Version=$(VERSION) \
  -X $(VERSION_PKG).Commit=$(COMMIT) \
  -X $(VERSION_PKG).BuildTime=$(BUILD_TIME)

OUTPUT ?= tmp/cube

.PHONY: build install

build: ## 构建到 OUTPUT（默认 tmp/cube）
	@echo "==> go build ($(VERSION) @ $(COMMIT))"
	go build -ldflags "$(LDFLAGS)" -o $(OUTPUT)
	@echo "==> built $(OUTPUT) ($(VERSION) @ $(COMMIT), $(BUILD_TIME))"

install: ## go install 到 GOBIN（默认 ~/go/bin）
	@echo "==> go install ($(VERSION) @ $(COMMIT))"
	go install -ldflags "$(LDFLAGS)"
	@echo "==> installed cube ($(VERSION) @ $(COMMIT), $(BUILD_TIME))"

install-zsh-completion:
	cube completion zsh > ~/.cube/zsh.sh