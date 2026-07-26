# Learning Service

Learning Service 是 TCM-History-AI 平台的学习模块底座，承担课程编排、学习记录、考试与错题管理职责。

## 职责

- **课程体系**：维护按中医发展史时间脉络编排的课程与课时，支持发布/取消发布
- **学习记录**：跟踪用户在每个课时的时长、位置与完成状态，自动计算整体进度
- **选课管理**：用户与课程的选课关系、进度更新、完成标记
- **考试系统**：考试与题目 CRUD + 自动评分（单选 / 多选 / 判断） + 考试记录
- **错题本**：考试提交时自动收集错题，支持标记已掌握
- **学习计划**：用户自定义学习计划，按关联课程进度自动计算完成百分比

## 模块结构

```
learning-service/
├── cmd/learning-service/     # 入口 main + wire
├── internal/
│   ├── application/           # usecase + dto
│   │   ├── dto/
│   │   └── usecase/
│   ├── conf/                  # 配置
│   ├── controller/            # Hertz HTTP handler
│   ├── domain/                # 实体、仓储接口、领域事件
│   │   ├── entity/
│   │   ├── event/
│   │   └── repository/
│   └── infrastructure/        # 适配器
│       ├── cache/             # Redis（学习进度、最近错题）
│       ├── eventbus/          # RabbitMQ
│       └── persistence/       # GORM 仓储
├── migrations/                # PostgreSQL 迁移（9 张表 + 种子数据）
└── Dockerfile
```

## 路由总览

| 资源 | 路径 |
| ---- | ---- |
| 课程 | `/api/v1/learning/courses` |
| 课时 | `/api/v1/learning/courses/:id/lessons`、`/api/v1/learning/lessons/:id` |
| 选课 | `/api/v1/learning/enrollments` |
| 学习记录 | `/api/v1/learning/records` |
| 考试 | `/api/v1/learning/exams` |
| 题目 | `/api/v1/learning/exams/:id/questions`、`/api/v1/learning/questions/:id` |
| 考试记录 | `/api/v1/learning/attempts` |
| 错题本 | `/api/v1/learning/wrong-questions` |
| 学习计划 | `/api/v1/learning/study-plans` |
| 健康检查 | `/health` |

## 设计依据

- 学习模块需求：[doc/01-产品需求(PRD).md](../../doc/01-产品需求(PRD).md) §5 学习模块
- AI 考试 / 错题 / 学习计划：[doc/01-产品需求(PRD).md](../../doc/01-产品需求(PRD).md) §6.3 / §6.4 / §6.6

## 运行

```bash
make run-learning-service
```

依赖 PostgreSQL（数据库 `tcm_learning`）/ Redis 7 / RabbitMQ，由 `deploy/docker/docker-compose.dev.yml` 一键拉起。

## 端口

- HTTP：8087
- app.node_id：7
