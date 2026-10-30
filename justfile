# https://just.systems

# 默认列出可用命令
default:
    just --list


# 输入参数（可用环境变量 OUTPUT 覆盖）
output := env_var_or_default("OUTPUT", "tmp/main")

# 从 git 收集构建期信息（与 core/version/version.go 配合，通过 ldflags 注入）
version    := `git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD`
commit     := `git rev-parse --short HEAD`
build_time := `date -u +%Y-%m-%dT%H:%M:%SZ`

module      := "cube"
version_pkg := module + "/internal/version"
ldflags     := "-X " + version_pkg + ".version=" + version + " -X " + version_pkg + ".commit=" + commit + " -X " + version_pkg + ".buildTime=" + build_time

ui_src := "./web"
concurrently := "web/node_modules/.bin/concurrently"

# 开发模式：concurrently 同时起 air 热重载与 vite
dev-all:
    pnpm --dir {{ ui_src }} install
    {{ concurrently }} \
            -n "api,web" -c "blue,green" \
            --kill-others \
            "just dev" \
            "just dev-web"

dev-web:
    pnpm --dir {{ ui_src }} dev

dev:
    air

fmt:
    go fmt ./...
    go mod tidy

lint:
    go vet ./...
    golangci-lint run ./...

fix: fmt
    golangci-lint run --fix ./...

test:
    go test -race ./...

# 编译前端产物
build-ui:
    @echo "==> 构建前端 ({{ ui_src }})"
    pnpm --dir {{ ui_src }} install
    pnpm --dir {{ ui_src }} build

# 编译到 {{output}} 并验证版本信息
build:
    @echo "==> go build ({{version}} @ {{commit}})"
    go build -ldflags "{{ldflags}}" -o {{output}} ./cmd/cube
    @echo "==> built {{output}} ({{version}} @ {{commit}}, {{build_time}})"
    @"{{output}}" version

# go install 安装到 GOBIN 并验证
install:
    @echo ">>> 编译安装 go install ... ({{version}} @ {{commit}} {{build_time}})"
    go install -ldflags "{{ldflags}}"

    @echo ">>> 已安装版本"
    @cube version


# 代码生成（生成逻辑在 cmd/gen/，模板在 cmd/gen/tpl/）
#   just gen domain <name>                        生成业务 domain 脚手架
#   just gen api <GET|POST> <domain> <base-name>  在已有 domain 的 handler 追加 api 端点
gen *args:
    go run ./cmd/gen {{args}}
    just fix

# 生成 api
gen-api:
    pnpm --dir {{ ui_src }} gen:api