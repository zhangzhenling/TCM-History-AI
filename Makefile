SHELL := /bin/bash
.DEFAULT_GOAL := help

## 顶层编排：统一调度 backend / frontend / mobile / deploy 子目录的常用任务。

help: ## 显示帮助
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

up: ## 启动本地依赖基础设施（docker compose dev）
	cd deploy/docker && docker compose -f docker-compose.dev.yml up -d

down: ## 停止本地依赖基础设施
	cd deploy/docker && docker compose -f docker-compose.dev.yml down

down-v: ## 停止并删除数据卷（谨慎！会清空 PG/Redis/Neo4j 等数据）
	cd deploy/docker && docker compose -f docker-compose.dev.yml down -v

logs: ## 跟随基础设施日志
	cd deploy/docker && docker compose -f docker-compose.dev.yml logs -f --tail=200

ps: ## 查看基础设施容器状态
	cd deploy/docker && docker compose -f docker-compose.dev.yml ps

backend-build: ## 编译全部后端服务
	$(MAKE) -C backend build

backend-test: ## 运行后端单元测试
	$(MAKE) -C backend test

backend-lint: ## 后端静态检查
	$(MAKE) -C backend lint

backend-codegen: ## 生成 Wire 依赖注入产物
	$(MAKE) -C backend codegen

backend-run-gateway: ## 本地运行 gateway
	$(MAKE) -C backend run-gateway

backend-run-user-service: ## 本地运行 user-service
	$(MAKE) -C backend run-user-service

backend-run-history-service: ## 本地运行 history-service
	$(MAKE) -C backend run-history-service

frontend-install: ## 安装前端依赖
	cd frontend && pnpm install

frontend-dev: ## 启动前端开发服务器
	cd frontend && pnpm dev

frontend-build: ## 构建前端生产产物
	cd frontend && pnpm build

mobile-pub: ## Flutter 拉取依赖
	cd mobile && flutter pub get

mobile-run: ## Flutter 运行
	cd mobile && flutter run
