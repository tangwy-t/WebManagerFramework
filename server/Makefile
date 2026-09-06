.PHONY: run build test lint vet fmt clean swagger docker-build docker-up docker-down

VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME  ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS = -X 'github.com/tangwy-t/webmanager-server/internal/pkg/version.Version=$(VERSION)' \
          -X 'github.com/tangwy-t/webmanager-server/internal/pkg/version.BuildTime=$(BUILD_TIME)' \
          -X 'github.com/tangwy-t/webmanager-server/internal/pkg/version.CommitHash=$(COMMIT_HASH)'

run:
	go run ./cmd/server

build:
	go build -ldflags "${LDFLAGS}" -o bin/server ./cmd/server

test:
	go test ./... -v -count=1

# 静态检查:优先 golangci-lint,未安装时降级 go vet。
# 注意 if/else 显式分支:不能写成 A && B || C —— golangci-lint 检出
# 问题(非零退出)也会触发 go vet,vet 通过会掩盖 lint 失败退出码。
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint 未安装,降级 go vet"; \
		go vet ./...; \
	fi

vet:
	go vet ./...

fmt:
	gofmt -l -w internal/ cmd/

# swag CLI 版本与 go.mod 的 swaggo/swag 依赖保持一致(当前 v1.16.6):
# 不同版本的 swag 生成的 docs 会有无意义差异,go run 钉版保证可复现。
SWAG_VERSION ?= v1.16.6

swagger:
	go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g cmd/server/main.go -o docs

# clean 会删除 docs/(git 跟踪且被 main.go 编译依赖的 swagger 包):
# clean 后构建需先 make swagger(需安装 swag CLI,见 README/开发文档)。
clean:
	rm -rf bin/ docs/

# ── Docker 编排(根目录 docker-compose.yml;server/ 自身不再有 compose 文件) ──
# docker-build: 计算版本三件套并注入 server 镜像(监控页「构建时间/提交 Hash」
# 数据源)。直接 docker compose build 不传参时,Dockerfile 兜底为
# BUILD_TIME=构建时刻、VERSION=docker、COMMIT_HASH=unknown。
# 本部署环境 BuildKit 不可用(buildx activity 目录只读),构建需 DOCKER_BUILDKIT=0:
#   DOCKER_BUILDKIT=0 make docker-build
docker-build:
	cd .. && docker compose build \
		--build-arg VERSION="$(VERSION)" \
		--build-arg BUILD_TIME="$(BUILD_TIME)" \
		--build-arg COMMIT_HASH="$(COMMIT_HASH)" \
		server

docker-up: docker-build
	cd .. && docker compose up -d

docker-down:
	cd .. && docker compose down