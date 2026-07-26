# TCM-History-AI 中医发展史 AI 学习平台 软件设计说明书 V2.1

> **一句话定位**：AI 驱动的中医发展史知识图谱平台。

本仓库是 TCM-History-AI 的完整软件设计说明书（V2.1 可执行项目蓝图）。它不是一份普通的开发计划，而是一套可以直接进入开发与协作的工程规范，包含系统架构图、数据库表设计、OpenAPI 接口定义、部署方案与开发规范。

---

## 文档目录

| 编号 | 文档 | 内容概要 |
| ---- | ---- | -------- |
| 00 | [项目简介](./00-项目简介.md) | 背景、市场分析、竞品分析、产品定位 |
| 01 | [产品需求 PRD](./01-产品需求(PRD).md) | 首页、学习模块、AI 模块、用户旅程 |
| 02 | [系统架构设计](./02-系统架构设计.md) | Clean Architecture、DDD、微服务划分 |
| 03 | [技术选型](./03-技术选型.md) | CloudWeGo 生态、存储、基础设施 |
| 04 | [数据库设计](./04-数据库设计.md) | 40+ 表、字段、索引、ER 图 |
| 05 | [知识图谱设计](./05-知识图谱设计.md) | Neo4j 节点、关系、Cypher 查询 |
| 06 | [RAG 设计](./06-RAG设计.md) | 文献处理、向量化、检索增强 |
| 07 | [Agent 设计](./07-Agent设计.md) | Planner→Reasoner→Retriever→Answer |
| 08 | [MCP 设计](./08-MCP设计.md) | Tool 开放能力、多模型接入 |
| 09 | [AI Prompt 设计](./09-AI-Prompt设计.md) | Prompt Center、配置化管理 |
| 10 | [接口设计 OpenAPI](./10-接口设计(OpenAPI).md) | 200+ REST 接口、Swagger |
| 11 | [后台管理设计](./11-后台管理设计.md) | CMS、权限、日志 |
| 12 | [前端设计](./12-前端设计.md) | Vue3 + Vben Admin |
| 13 | [移动端设计](./13-移动端设计.md) | Flutter |
| 14 | [部署方案](./14-部署方案.md) | 整体部署架构 |
| 15 | [Docker 部署](./15-Docker部署.md) | Compose 一键启动 |
| 16 | [Kubernetes 部署](./16-Kubernetes部署.md) | 生产级 K8s |
| 17 | [CI/CD](./17-CI-CD.md) | GitHub Actions 流水线 |
| 18 | [开发规范](./18-开发规范.md) | 目录结构、代码规范、Git 流程 |
| 19 | [测试方案](./19-测试方案.md) | 单元、集成、E2E、性能测试 |
| 20 | [商业化方案](./20-商业化方案.md) | 会员、API、企业版、学校版 |

---

## 共享设计约定

以下约定贯穿全部章节，所有模块遵循同一套命名与技术基线。

### 技术基线

| 层 | 技术选型 |
| ---- | -------- |
| 后端语言 | Go 1.22+ |
| HTTP 框架 | Hertz（CloudWeGo） |
| RPC 框架 | Kitex（CloudWeGo） |
| 网络库 | Netpoll（CloudWeGo） |
| ORM | GORM v2 |
| 配置中心 | Viper |
| 日志 | Zap |
| 关系数据库 | PostgreSQL 16 |
| 缓存 | Redis 7 |
| 图数据库 | Neo4j 5 |
| 向量数据库 | Milvus 2 |
| 对象存储 | MinIO |
| 搜索引擎 | Meilisearch |
| PC 前端 | Vue3 + Vben Admin |
| 移动端 | Flutter |
| 容器 | Docker / Kubernetes |
| 监控 | Prometheus + Grafana + Loki + Tempo + Jaeger |

### 微服务划分

```
Gateway
  ├── User Service       用户、认证、权限
  ├── History Service    人物、经典、学派、朝代、事件
  ├── Knowledge Service  RAG、Embedding、文献检索
  ├── Graph Service      Neo4j 知识图谱
  ├── AI Service         LLM、Agent、Prompt
  └── Learning Service   课程、学习记录、考试、错题
```

### 核心领域实体

| 实体 | 数据库表 | 说明 |
| ---- | -------- | ---- |
| 用户 | `users` | 平台用户 |
| 历史人物 | `history_person` | 张仲景、华佗等 |
| 经典著作 | `history_book` | 黄帝内经、伤寒论等 |
| 学派 | `history_school` | 伤寒派、温病学派等 |
| 朝代 | `history_dynasty` | 先秦至近现代 |
| 历史事件 | `history_event` | 重大医学事件 |
| 方剂 | `prescription` | 经典方剂 |
| 药物 | `medicine` | 中药材 |
| 疾病 | `disease` | 病症 |
| 课程 | `course` | 学习课程 |
| 学习记录 | `learning_record` | 学习进度 |
| 考试 | `exam` | 试题与考试 |
| Prompt 模板 | `prompt_template` | AI 提示词 |
| Embedding 任务 | `embedding_task` | 向量化任务 |

### 知识图谱节点与关系

- **节点**：人物、经典、学派、方剂、药物、疾病、朝代、历史事件
- **关系**：著作、师承、影响、属于、发生于、引用、提出、反对、继承

### 文档规范

- 全部使用 Markdown 编写，可直接导入 GitBook、Docsify、VuePress。
- 架构图统一使用 Mermaid 语法，确保版本可控、可差异对比。
- 数据库表设计包含字段、类型、索引、外键。
- API 接口遵循 RESTful 规范，自动生成 Swagger 文档。

---

## 版本历史

| 版本 | 日期 | 变更说明 |
| ---- | ---- | -------- |
| V1.0 | — | 初始开发计划 |
| V2.0 | 2026-07 | 升级为企业架构版，补齐全栈设计 |
| V2.1 | 2026-07-25 | 升级为可执行项目蓝图，补充 Mermaid 架构图、ER 图、OpenAPI 定义、开发规范与任务拆分 |
