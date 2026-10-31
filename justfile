# https://just.systems

# 默认列出可用命令
default:
    just --list


# 输入参数（可用环境变量 OUTPUT 覆盖）
output := env_var_or_default("OUTPUT", "tmp/main")

# 从 git 收集构建期信息（与 internal/version/version.go 配合，通过 ldflags 注入）
version    := `git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD`
commit     := `git rev-parse --short HEAD`
build_time := `date -u +%Y-%m-%dT%H:%M:%SZ`

module      := "cube"
version_pkg := module + "/internal/version"
ldflags     := "-X " + version_pkg + ".version=" + version + " -X " + version_pkg + ".commit=" + commit + " -X " + version_pkg + ".buildTime=" + build_time

zsh_completion_file := "~/.config/cube/zsh.sh"

# 主命令入口包
main_cmd := "./cmd/cube"
# 工具命令入口包
devtool_cmd := "./cmd/devtool"

ui_src := "./web"
concurrently := "web/node_modules/.bin/concurrently"

# 开发模式：concurrently 同时起 air 热重载与 vite
dev:
    pnpm --dir {{ ui_src }} install
    {{ concurrently }} \
            -n "api,web" -c "blue,green" \
            --kill-others \
            "just dev-web" \
            "just dev-server"

dev-web:
    pnpm --dir {{ ui_src }} dev

dev-server:
    @# 启动前清除 6001 上的残留：air 被异常退出后孤儿 server 会一直占着端口，导致后续 air 热重启端口冲突起不来。
    @# 必须带 -sTCP:LISTEN 只杀监听进程——不带过滤会把连着 6001 的客户端（vite 代理、浏览器连接等）一起杀掉
    @lsof -ti :6001 -sTCP:LISTEN | xargs kill 2>/dev/null || true
    air

# link 配置文件到项目临时目录，方便查看
dev-link-config:
	mkdir -p ./tmp/.config
	ln -s ~/.config/cube-dev ./tmp/.config/cube-dev
	ln -s ~/.config/cube ./tmp/.config/cube

fmt:
    golangci-lint fmt

# 依赖对账：按实际 import 重算 go.mod / go.sum（加删依赖后手动跑，不属于日常 fmt）
tidy:
    go mod tidy

lint:
    golangci-lint run

fix: fmt
    golangci-lint run --fix

test:
    go test -race ./...

# 编译前端产物
build-ui:
    @echo "==> 构建前端 ({{ ui_src }})"
    pnpm --dir {{ ui_src }} install
    pnpm --dir {{ ui_src }} build

# 编译到 {{output}} 并验证版本信息
build: build-ui
    @echo "==> go build ({{version}} @ {{commit}})"
    go build -ldflags "{{ldflags}}" -o {{output}} ./cmd/cube
    @echo "==> built {{output}} ({{version}} @ {{commit}}, {{build_time}})"
    @"{{output}}" version

# go install 安装到 GOBIN 并验证
install: build-ui
    @echo ">>> 编译安装 go install ... ({{version}} @ {{commit}} {{build_time}})"
    go install -ldflags "{{ldflags}}"

    @# 安装完成，确认生效：PATH 上的 cube 必须是刚构建的版本（输出含本次BUILD_TIME），否则视为安装未生效（GOBIN 不在 PATH / 旧版本在前等），中止
    @cube version | grep -qF "{{build_time}}" || { echo "!! 安装校验失败：PATH 上的 cube 不是刚构建的版本（检查 GOBIN 是否在 PATH 且优先于旧安装）" >&2; exit 1; }

    @echo ">>> 已安装版本"
    @cube version

    @echo ">>> 停止旧服务"
    @# 关闭旧版本 server（服务未运行时 stop 会非零退出，- 忽略）
    -@cube server stop 2>/dev/null

    @echo ">>> 写入 zsh 提示"
    @# 安装 zsh completion + shell 扩展（p / pz 函数及其 TAB 补全，见 ./scripts/zsh-append.sh）
    @# completion 生成是覆盖写，重复 install 不会累积
    @cube completion zsh > {{zsh_completion_file}}
    @cat ./scripts/zsh-append.sh >> {{zsh_completion_file}}

# 执行一次主命令
run *args:
    go run {{ main_cmd }} {{ args }}

# 代码生成（生成逻辑在 cmd/devtool/internal/generate/，模板在其 tpl/ 下）
#   just gen domain <name>                        生成业务 domain 脚手架
#   just gen api <GET|POST> <domain> <base-name>  在已有 domain 的 handler 追加 api 端点
gen *args:
    go run {{ devtool_cmd }} generate {{args}}
    just fix

# 生成 api
gen-api:
    pnpm --dir {{ ui_src }} gen:api
