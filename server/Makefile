# ============================================================
# 该文件已迁移到根目录 Makefile（统一管理 server + web 两端）。
# 保留此薄委托仅为兼容旧习惯：在 server/ 目录下敲 `make <目标>` 时
# 转发到根 Makefile，避免逻辑分散重复维护。
# 推荐改用根目录 `make server-*` / `make web-*` / `make docker-*`。
# ============================================================

ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/..)

# server/ 下的历史无前缀目标映射到根 Makefile 的 server-* 目标：
#   `make build` → 根 `make server-build`，`make run` → 根 `make server-run`
.PHONY: run build test lint vet fmt swagger clean
run build test lint vet fmt swagger clean:
	@$(MAKE) -C $(ROOT) server-$@

# 其余任意目标（help / docker-* / web-* / all / ...）同名转发到根 Makefile。
# %: 为隐式规则，仅在无显式规则匹配时生效，不会覆盖上面那一组。
%:
	@$(MAKE) -C $(ROOT) $@