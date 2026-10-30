.DEFAULT_GOAL := build
.PHONY: build-ui build install tag dev-server dev-web dev-link-config last-proposal

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
