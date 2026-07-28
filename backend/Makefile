SHELL := /bin/bash
.DEFAULT_GOAL := help

GO := go
BIN_DIR := bin
WIRE := wire
SERVICES := gateway user-service history-service knowledge-service graph-service ai-service learning-service

help: ## 显示帮助
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## 编译所有服务到 bin/
	@mkdir -p $(BIN_DIR)
	@for svc in $(SERVICES); do \
		if [ ! -d ./$$svc/cmd/$$svc ]; then \
			echo "==> skip $$svc (cmd dir not found)"; \
			continue; \
		fi; \
		echo "==> building $$svc"; \
		$(GO) build -o $(BIN_DIR)/$$svc ./$$svc/cmd/$$svc || exit 1; \
	done
	@echo "==> all services built into $(BIN_DIR)"

build-gateway: ## 仅编译 gateway
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/gateway ./gateway/cmd/gateway

test: ## 运行后端单测
	$(GO) test -race -coverprofile=coverage.txt ./...

e2e: ## 运行后端 E2E 测试（带 e2e build tag）
	$(GO) test -tags=e2e -race -timeout=120s ./...

lint: ## 后端静态检查
	golangci-lint run ./...

codegen: ## 生成 Wire 产物
	@for svc in $(SERVICES); do \
		echo "==> wire $$svc"; \
		( cd ./$$svc/cmd/$$svc && $(WIRE) ) || exit 1; \
	done

fmt: ## 格式化
	$(GO) fmt ./...
	goimports -w .

tidy: ## 整理依赖
	$(GO) mod tidy

vet: ## go vet
	$(GO) vet ./...

run-gateway: ## 本地运行 gateway
	$(GO) run ./gateway/cmd/gateway

run-user-service: ## 本地运行 user-service
	$(GO) run ./user-service/cmd/user-service

run-history-service: ## 本地运行 history-service
	$(GO) run ./history-service/cmd/history-service

run-knowledge-service:
	$(GO) run ./knowledge-service/cmd/knowledge-service

run-graph-service:
	$(GO) run ./graph-service/cmd/graph-service

run-ai-service:
	$(GO) run ./ai-service/cmd/ai-service

run-learning-service:
	$(GO) run ./learning-service/cmd/learning-service

clean:
	rm -rf $(BIN_DIR) coverage.txt
