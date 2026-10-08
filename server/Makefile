.DEFAULT_GOAL := build
.PHONY: dev build install

# 输入参数
OUTPUT ?= tmp/cube

# 从 git 收集构建期信息（与 core/version/version.go 配合，通过 ldflags 注入）
VERSION    := $(shell git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD)
COMMIT     := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

VERSION_PKG := cube/core/version
LDFLAGS := \
  -X $(VERSION_PKG).version=$(VERSION) \
  -X $(VERSION_PKG).commit=$(COMMIT) \
  -X $(VERSION_PKG).buildTime=$(BUILD_TIME)

ZSH_COMPLETION_FILE := ~/.config/cube/zsh.sh

# 启动前清除 6001 上的残留：air 被异常退出后孤儿 server 会一直占着端口，导致后续 air 热重启端口冲突起不来。
# 注意必须 -sTCP:LISTEN 只杀监听进程——不带过滤会把连着 6001 的客户端（vite 代理、浏览器连接等）一起杀掉
dev:
	-@lsof -ti :6001 -sTCP:LISTEN | xargs kill 2>/dev/null || true
	air

build:
	@echo "==> go build ($(VERSION) @ $(COMMIT))"
	go build -ldflags "$(LDFLAGS)" -o $(OUTPUT)
	@echo "==> built $(OUTPUT) ($(VERSION) @ $(COMMIT), $(BUILD_TIME))"
	@$(OUTPUT) version

install:
	@echo ">>> 编译安装 go install ... ($(VERSION) @ $(COMMIT) $(BUILD_TIME))"
	go install -ldflags "$(LDFLAGS)"

	@echo ">>> 验证是否正确安装..."
	@# 安装完成，确认生效：PATH 上的 cube 必须是刚构建的版本（输出含本次BUILD_TIME），否则视为安装未生效（GOBIN 不在 PATH / 旧版本在前等），中止
	@cube version | grep -qF "$(BUILD_TIME)" || { echo "!! 安装校验失败：PATH 上的 cube 不是刚构建的版本（检查 GOBIN 是否在 PATH 且优先于旧安装）" >&2; exit 1; }

	@echo ">>> 已安装版本"
	@cube version

	@echo ">>> 停止旧服务"
	@# 关闭旧版本 server（服务未运行时 stop 会非零退出，- 忽略）
	-@cube server stop 2>/dev/null

	@echo ">>> 写入 zsh 提示"
	@# 安装 zsh completion + shell 扩展（p / pz 函数及其 TAB 补全，见 ./scripts/zsh-append.sh）
	@# completion 生成是覆盖写，重复 install 不会累积
	@cube completion zsh > $(ZSH_COMPLETION_FILE)
	@cat ./scripts/zsh-append.sh >> $(ZSH_COMPLETION_FILE)