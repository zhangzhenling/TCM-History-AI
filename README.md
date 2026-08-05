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
├── doc/            # 软件设计说明书（00–23 章）与辅助文档
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
| RPC 框架 | Kitex（CloudWeGo，当前为预留，跨服务调用通过 Gateway HTTP 反代） |
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
| 21 | [开发路线图](./doc/21-开发路线图.md) | P0–P8 阶段规划与里程碑 |
| 22 | [Linux 部署](./doc/22-Linux部署.md) | Ubuntu 原生部署、systemd 服务管理 |
| 23 | [Windows 部署](./doc/23-Windows部署.md) | WSL2 与原生 Windows 部署方案 |

---

## 开发规范

- 提交信息遵循 [Conventional Commits](https://www.conventionalcommits.org/)，CI 会通过 commitlint 校验。
- 后端代码遵循 `golangci-lint` 规则，提交前执行 `make backend-lint`。
- 前端代码使用 ESLint + Prettier，提交前执行 `make frontend-build`。
- 详细规范见 [18-开发规范](./doc/18-开发规范.md)。

---

## 发布流程

平台级发布通过 Git 标签触发 GitHub Actions 自动化（[`.github/workflows/release.yml`](./.github/workflows/release.yml)）。

### 触发方式

推送形如 `v*` 的标签即可触发全平台发布：

```bash
git tag v1.0.0
git push origin v1.0.0
```

也可在 GitHub Actions 页面通过 `workflow_dispatch` 手动指定标签触发。

### 产物清单

| 产物 | 说明 |
| ---- | ---- |
| `tcm-backend-<ver>-linux-{amd64,arm64}.tar.gz` | 7 个后端服务二进制 + `.sha256` 校验 |
| `tcm-frontend-{learner,admin}-<ver>.tar.gz` | 前端生产构建 + `.sha256` 校验 |
| `tcm-mobile-<ver>.{apk,aab}` | Android 安装包 + `.sha256` 校验 |
| `ghcr.io/<owner>/<repo>/<service>:<ver>` | 7 个服务多架构 Docker 镜像（GHCR） |

### 标签约定

- `v<semver>`（如 `v1.0.0`、`v1.1.0-rc.1`）：全平台统一发布，由 `release.yml` 处理
- `mobile-v<semver>`：仅移动端补丁发布，由 `mobile-ci.yml` 处理

### 版本号规则

遵循 [Semantic Versioning](https://semver.org/lang/zh/) `MAJOR.MINOR.PATCH`：

- `MAJOR`：不兼容的 API 变更
- `MINOR`：向后兼容的功能新增
- `PATCH`：向后兼容的缺陷修复
- 预发布版附加 `-rc.1` / `-beta.1` 等后缀，自动标记为 GitHub prerelease

---

## 版本历史

| 版本 | 日期 | 变更说明 |
| ---- | ---- | -------- |
| V1.0 | — | 初始开发计划 |
| V2.0 | 2026-07 | 升级为企业架构版，补齐全栈设计 |
| V2.1 | 2026-07-25 | 升级为可执行项目蓝图，补充 Mermaid 架构图、ER 图、OpenAPI 定义、开发规范与任务拆分 |
| V2.2 | 2026-07-27 | P0–P4 阶段全部完成，P5 前端双端联调基本完成，P6 移动端 API 联调完成，P7 生产化部署基本完成（含生产化 checklist 演练：Secret 外部化 / 镜像 tag 固定 / 备份恢复演练），P8 商业化后端基本完成 |
| V2.3 | 2026-07-28 | 完整实现 MCP 协议支持（JSON-RPC 2.0 + SSE + 多模型适配），补全中医历史数据种子（新增药物与方剂-药物关联），关键服务性能基准测试（MCP / Embedding / 租户服务），新增 Linux 与 Windows 详细原生部署文档 |
| V2.4 | 2026-07-29 | 新增 GitHub 自动打包发布流程（`release.yml`）：`v*` 标签触发，构建后端多架构二进制 + 前端双端 + 移动端 APK/AAB + 7 服务 Docker 镜像推送 GHCR，自动创建 GitHub Release；发布首个版本 v1.0.0 |
