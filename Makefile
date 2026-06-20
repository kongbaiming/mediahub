# MediaHub Makefile
# 一站式开发命令

SHELL := /bin/bash
.DEFAULT_GOAL := help

# ---------- 变量 ----------
API_DIR       := services/api
ADMIN_DIR     := services/admin
WEB_DIR       := services/web-player

# ---------- 帮助 ----------
.PHONY: help
help:  ## 显示帮助
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ---------- Docker ----------
.PHONY: up
up:  ## 启动所有 Docker 服务
	docker compose up -d

.PHONY: down
down:  ## 停止所有 Docker 服务
	docker compose down

.PHONY: logs
logs:  ## 查看所有服务日志
	docker compose logs -f

.PHONY: logs-api
logs-api:  ## 仅查看 API 日志
	docker compose logs -f api

.PHONY: restart
restart:  ## 重启所有服务
	docker compose restart

.PHONY: clean
clean:  ## 停止 + 清理所有数据卷（⚠️ 会删除数据库）
	docker compose down -v

.PHONY: rebuild
rebuild:  ## 重新构建所有镜像
	docker compose build --no-cache

# ---------- 后端 (Go) ----------
.PHONY: api-deps
api-deps:  ## 下载 Go 依赖
	cd $(API_DIR) && go mod tidy

.PHONY: api-dev
api-dev:  ## 本地运行 API（需本地 postgres + redis）
	cd $(API_DIR) && go run ./cmd/server

.PHONY: api-build
api-build:  ## 编译 API 二进制
	cd $(API_DIR) && CGO_ENABLED=0 go build -o bin/server ./cmd/server

.PHONY: api-test
api-test:  ## 运行 Go 测试
	cd $(API_DIR) && go test -race -coverprofile=coverage.out ./...

.PHONY: api-test-coverage
api-test-coverage: api-test  ## 运行测试 + 打开覆盖率报告
	cd $(API_DIR) && go tool cover -html=coverage.out

.PHONY: api-lint
api-lint:  ## Go Lint
	cd $(API_DIR) && \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
		golangci-lint run ./...

.PHONY: api-fmt
api-fmt:  ## Go 格式化
	cd $(API_DIR) && gofmt -s -w .

# ---------- 前端 Admin ----------
.PHONY: admin-deps
admin-deps:  ## 安装 Admin 依赖
	cd $(ADMIN_DIR) && npm install

.PHONY: admin-dev
admin-dev:  ## 启动 Admin 开发服务器
	cd $(ADMIN_DIR) && npm run dev

.PHONY: admin-build
admin-build:  ## 构建 Admin 生产包
	cd $(ADMIN_DIR) && npm run build

.PHONY: admin-lint
admin-lint:  ## Admin Lint
	cd $(ADMIN_DIR) && npm run lint

.PHONY: admin-type-check
admin-type-check:  ## Admin 类型检查
	cd $(ADMIN_DIR) && npm run type-check

# ---------- 前端 Web Player ----------
.PHONY: web-deps
web-deps:  ## 安装 Web Player 依赖
	cd $(WEB_DIR) && npm install

.PHONY: web-dev
web-dev:  ## 启动 Web Player 开发服务器
	cd $(WEB_DIR) && npm run dev

.PHONY: web-build
web-build:  ## 构建 Web Player 生产包
	cd $(WEB_DIR) && npm run build

.PHONY: web-type-check
web-type-check:  ## Web Player 类型检查
	cd $(WEB_DIR) && npm run type-check

# ---------- 一键全套 ----------
.PHONY: all
all: api-test admin-type-check web-type-check  ## 跑完整套测试

.PHONY: ci
ci: api-fmt api-lint api-test admin-lint admin-type-check admin-build web-type-check web-build  ## 模拟 CI 全流程

# ---------- 清理 ----------
.PHONY: clean-build
clean-build:  ## 清理构建产物
	rm -rf $(API_DIR)/bin $(API_DIR)/coverage.out
	rm -rf $(ADMIN_DIR)/dist $(WEB_DIR)/dist
