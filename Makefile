.DEFAULT_GOAL := build
.PHONY: build-ui build install tag dev-server dev-web dev-link-config last-proposal

# dist 即完整产物，touch 补 .keep 后直接整体 mv 换入（同文件系统 mv 是原子 rename）：
# 任何时刻 web/ui 要么旧要么新，没有半成品窗口；.keep 是提交进仓库的（保 embed 可编译
# 的空目录兜底），在 dist 里补回就不会弄脏 git 工作区。代价：web/dist 被消费，下次构建重建
build-ui:
	pnpm -C ./web build
	touch ./web/dist/.keep
	rm -rf ./server/web/ui
	mv ./web/dist ./server/web/ui

build: build-ui
	$(MAKE) -C server build

install: build-ui
	$(MAKE) -C server install

tag: ## 在当前位置打一个新版本 tag（上个版本末位 +1，如 v3.0.6 -> v3.0.7）
	@set -e; \
	export LC_ALL="${LC_ALL:-en_US.UTF-8}" LANG="${LANG:-en_US.UTF-8}"; \
	prev=$$(git describe --tags --abbrev=0 2>/dev/null); \
	if [ -z "$$prev" ]; then echo "==> 仓库还没有任何 tag" >&2; exit 1; fi; \
	prev_commit=$$(git rev-parse "$$prev^{commit}"); \
	curr_commit=$$(git rev-parse HEAD); \
	if [ "$$prev_commit" = "$$curr_commit" ]; then \
	  echo "==> 当前位置 $${curr_commit:0:7} 已是上个版本 $$prev, 无需打新 tag" >&2; exit 1; \
	fi; \
	major=$$(echo "$$prev" | sed -E 's/^v([0-9]+)\.([0-9]+)\.([0-9]+).*$$/\1/'); \
	minor=$$(echo "$$prev" | sed -E 's/^v([0-9]+)\.([0-9]+)\.([0-9]+).*$$/\2/'); \
	patch=$$(echo "$$prev" | sed -E 's/^v([0-9]+)\.([0-9]+)\.([0-9]+).*$$/\3/'); \
	patch=$$((patch + 1)); \
	new_tag="v$$major.$$minor.$$patch"; \
	echo "==> 上个版本: $$prev ($${prev_commit:0:7})"; \
	echo "==> 新版本  : $$new_tag ($${curr_commit:0:7})"; \
	git tag -a "$$new_tag" -m "release $$new_tag"; \
	echo "==> 已打 tag $$new_tag, 如需推送: git push origin $$new_tag"

last-proposal:
	@ls -d docs/proposals/*/ docs/proposals/archived/*/ docs/proposals/parked/*/ 2>/dev/null \
		| awk -F/ '/\/1[0-9]{3}-/ {print $$(NF-1)}' | sort | tail -1

# -------

dev-server:
	$(MAKE) -C server dev

dev-web:
	cd web && pnpm dev

dev-link-config:
	mkdir -p ./tmp/.config
	ln -s ~/.config/cube-dev ./tmp/.config/cube-dev
	ln -s ~/.config/cube ./tmp/.config/cube