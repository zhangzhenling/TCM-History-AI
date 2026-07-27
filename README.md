# TCM-History-AI 中医发展史 AI 学习平台

> **一句话定位**：AI 驱动的中医发展史知识图谱平台。

本仓库是 TCM-History-AI 的工程实现 monorepo，涵盖后端微服务、PC 前端、Flutter 移动端、本地/生产部署配置与 CI/CD 流水线，并附带一份完整的软件设计说明书（00–20 章）。设计说明书是项目的可执行蓝图，包含系统架构图、数据库表设计、OpenAPI 接口定义、部署方案与开发规范。

---

## Monorepo 结构

```
tcm-history-ai/
├── backend/        # 后端：Go 微服务（gateway + 6 业务服务 + 共享 pkg）
├── frontend/       # PC 前端：Vue3 + Vben Admin
├── mobile/         # 移动端：Flutter
├── deploy/         # 部署配置：docker-compose / k8s / helm
├── doc/            # 软件设计说明书（00–20 章）
├── docs/           # 辅助文档（架构图导出、ADR 等）
├── .github/        # CI/CD：GitHub Actions workflows + PR 模板
├── Makefile        # 顶层编排
└── README.md       # 本文件
```

---

## 快速开始

### 环境要求

- Go 1.22+
- Node 20 + pnpm 9
- Flutter 3.x
- Docker + Docker Compose v2
- GNU Make

### 1. 启动本地依赖基础设施

```bash
make up
```

该命令会在 `deploy/docker/` 下通过 `docker compose -f docker-compose.dev.yml up -d` 拉起 PostgreSQL、Redis、Neo4j、Milvus、MinIO、Meilisearch、RabbitMQ、etcd 共 8 个服务。查看状态：

```bash
make ps        # 容器状态
make logs      # 跟随日志
make down      # 停止
make down-v    # 停止并清空数据卷（谨慎）
```

环境变量从 `deploy/docker/.env.example` 复制为 `deploy/docker/.env`（已通过 compose `env_file` 自动加载，未提供时使用内置默认值）。

### 2. 编译后端服务

```bash
make backend-build       # 编译 7 个服务到 backend/bin/
make backend-run-history-service   # 本地运行 history-service
```

更多后端任务见 `backend/Makefile`：`build / test / lint / codegen / fmt / tidy / vet / run-*`。

### 3. 启动前端

```bash
make frontend-install    # pnpm install
make frontend-dev        # 开发服务器
make frontend-build      # 生产构建
```

### 4. 移动端

```bash
make mobile-pub          # flutter pub get
make mobile-run          # flutter run
```

---

## 技术栈

| 层 | 技术选型 |
| ---- | -------- |
| 后端语言 | Go 1.22+ |
| HTTP 框架 | Hertz（CloudWeGo） |
| RPC 框架 | Kitex（CloudWeGo） |
| ORM | GORM v2 |
| 依赖注入 | Google Wire |
| 关系数据库 | PostgreSQL 16 |
| 缓存 | Redis 7 |
| 图数据库 | Neo4j 5 |
| 向量数据库 | Milvus 2 |
| 对象存储 | MinIO |
| 搜索引擎 | Meilisearch |
| 消息队列 | RabbitMQ 3.13 |
| PC 前端 | Vue3 + Vben Admin |
| 移动端 | Flutter |
| 容器与编排 | Docker / Kubernetes |
| CI/CD | GitHub Actions |

### 微服务划分

```
Gateway
  ├── User Service       用户、认证、权限
  ├── History Service    人物、经典、学派、朝代、事件（已完成 P2）
  ├── Knowledge Service  RAG、Embedding、文献检索
  ├── Graph Service      Neo4j 知识图谱
  ├── AI Service         LLM、Agent、Prompt
  └── Learning Service   课程、学习记录、考试、错题
```

---

## 文档目录

设计说明书位于 `doc/` 目录，共 21 章（00–20）：

| 编号 | 文档 | 内容概要 |
| ---- | ---- | -------- |
| 00 | [项目简介](./doc/00-项目简介.md) | 背景、市场分析、竞品分析、产品定位 |
| 01 | [产品需求 PRD](./doc/01-产品需求(PRD).md) | 首页、学习模块、AI 模块、用户旅程 |
| 02 | [系统架构设计](./doc/02-系统架构设计.md) | Clean Architecture、DDD、微服务划分 |
| 03 | [技术选型](./doc/03-技术选型.md) | CloudWeGo 生态、存储、基础设施 |
| 04 | [数据库设计](./doc/04-数据库设计.md) | 40+ 表、字段、索引、ER 图 |
| 05 | [知识图谱设计](./doc/05-知识图谱设计.md) | Neo4j 节点、关系、Cypher 查询 |
| 06 | [RAG 设计](./doc/06-RAG设计.md) | 文献处理、向量化、检索增强 |
| 07 | [Agent 设计](./doc/07-Agent设计.md) | Planner→Reasoner→Retriever→Answer |
| 08 | [MCP 设计](./doc/08-MCP设计.md) | Tool 开放能力、多模型接入 |
| 09 | [AI Prompt 设计](./doc/09-AI-Prompt设计.md) | Prompt Center、配置化管理 |
| 10 | [接口设计 OpenAPI](./doc/10-接口设计(OpenAPI).md) | 200+ REST 接口、Swagger |
| 11 | [后台管理设计](./doc/11-后台管理设计.md) | CMS、权限、日志 |
| 12 | [前端设计](./doc/12-前端设计.md) | Vue3 + Vben Admin |
| 13 | [移动端设计](./doc/13-移动端设计.md) | Flutter |
| 14 | [部署方案](./doc/14-部署方案.md) | 整体部署架构 |
| 15 | [Docker 部署](./doc/15-Docker部署.md) | Compose 一键启动 |
| 16 | [Kubernetes 部署](./doc/16-Kubernetes部署.md) | 生产级 K8s |
| 17 | [CI/CD](./doc/17-CI-CD.md) | GitHub Actions 流水线 |
| 18 | [开发规范](./doc/18-开发规范.md) | 目录结构、代码规范、Git 流程 |
| 19 | [测试方案](./doc/19-测试方案.md) | 单元、集成、E2E、性能测试 |
| 20 | [商业化方案](./doc/20-商业化方案.md) | 会员、API、企业版、学校版 |

---

## 开发规范

- 提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/)，CI 会通过 commitlint 校验。
- 后端代码遵循 `golangci-lint` 规则，提交前执行 `make backend-lint`。
- 前端代码使用 ESLint + Prettier，提交前执行 `make frontend-build`。
- 详细规范见 [18-开发规范](./doc/18-开发规范.md)。

---

## 版本历史

| 版本 | 日期 | 变更说明 |
| ---- | ---- | -------- |
| V1.0 | — | 初始开发计划 |
| V2.0 | 2026-07 | 升级为企业架构版，补齐全栈设计 |
| V2.1 | 2026-07-25 | 升级为可执行项目蓝图，补充 Mermaid 架构图、ER 图、OpenAPI 定义、开发规范与任务拆分 |
| V2.2 | 2026-07-27 | P0–P4 阶段全部完成，P5 前端双端联调基本完成，P6 移动端 API 联调完成，P7 生产化部署基本完成，P8 商业化后端基本完成 |
