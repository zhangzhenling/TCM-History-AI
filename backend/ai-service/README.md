# AI Service

AI Service 是 TCM-History-AI 平台的 LLM/Agent/Prompt/MCP 服务底座，承担对话编排、Agent 多阶段执行、Prompt 模板治理与 MCP Tool 注册与调用职责。

## 职责

- **对话管理**：单轮/多轮对话（chat），持久化消息历史，发布 `ai.message.created` 事件
- **Agent 编排**：Plan→Execute→Answer 最小骨架（支持 Knowledge Service / Graph Service / Tool 调用扩展点）
- **Prompt Center**：模板 CRUD + 按场景获取激活模板 + `{{variable}}` 变量渲染
- **MCP Tool 注册表**：Tool CRUD + 启用/禁用 + 通过 HTTP 调用注册的 endpoint
- **Agent 运行记录**：plan/steps/final_answer 全量留存，便于中断恢复与离线分析

## 模块结构

```
ai-service/
├── cmd/ai-service/            # 入口 main + wire
├── internal/
│   ├── application/            # usecase + dto
│   │   ├── dto/
│   │   └── usecase/
│   ├── conf/                  # 配置
│   ├── controller/            # Hertz HTTP handler
│   ├── domain/                # 实体、仓储接口、领域事件、端口
│   │   ├── entity/
│   │   ├── event/
│   │   ├── repository/
│   │   └── service/           # LLMProvider / ToolExecutor / PromptRenderer 端口
│   └── infrastructure/        # 适配器
│       ├── eventbus/          # RabbitMQ 发布（延迟连接）
│       ├── llm/               # LLM 适配（OpenAI / Anthropic HTTP + stub 回退）
│       ├── persistence/       # GORM 仓储
│       ├── prompt/            # Prompt 渲染器
│       ├── retrieval/         # Knowledge Service RAG 客户端
│       └── tool/              # Tool 执行器（HTTP 调用注册的 endpoint）
├── migrations/                # PostgreSQL 迁移（含 Prompt 模板种子）
└── Dockerfile
```

## 当前实现状态

为保持离线可构建性，LLM 通过 `net/http` 直连 OpenAI / Anthropic 的 Chat Completions 协议，不引入官方 Go SDK。

- **LLM Provider**：`internal/infrastructure/llm/provider.go` 实现 OpenAI（`/v1/chat/completions`）与 Anthropic（`/v1/messages`）两种 HTTP 客户端，按 `llm.provider` 配置选择。`llm.provider=stub` 时返回回显响应，便于离线联调。
- **Tool Executor**：通过 HTTP 调用注册的 endpoint，未注册或调用失败返回 `degraded=true` 的桩结果。
- **Retrieval Client**：`internal/infrastructure/retrieval/` 通过 HTTP 调用 knowledge-service 的 RAG 检索接口。
- **Prompt Renderer**：`{{variable}}` 模板渲染，纯本地实现。
- **Event Publisher**：连接延迟到首次 Publish，避免 broker 未就绪时阻塞启动。

## 设计依据

- Agent 设计：[doc/07-Agent设计.md](../../doc/07-Agent设计.md)
- MCP 设计：[doc/08-MCP设计.md](../../doc/08-MCP设计.md)
- Prompt 设计：[doc/09-AI-Prompt设计.md](../../doc/09-AI-Prompt设计.md)
- 数据库设计：[doc/04-数据库设计.md](../../doc/04-数据库设计.md) §6

## 运行

```bash
make run-ai-service
```

依赖 PostgreSQL / RabbitMQ，由 `deploy/docker/docker-compose.dev.yml` 一键拉起。

## HTTP 端口

- `8086` （`http.port`，与 docker-compose.dev.yml 对齐）
