# syntax=docker/dockerfile:1.7
#
# TCM-History-AI 前端 Dockerfile（learner + admin 双应用）
# 构建上下文：frontend/ 目录
#   docker build -f Dockerfile -t tcm-history-ai/frontend:latest .
#
# 产出镜像内含两个 Nginx server block：
#   - 80   端口 → learner（学习端）
#   - 8088 端口 → admin（管理后台）
# 两个端口均将 /api、/health 反代到 gateway 服务。

############################
# Builder：pnpm install + build learner & admin
############################
FROM node:20-alpine AS builder

# 安装 pnpm（与 package.json packageManager 版本对齐）
RUN corepack enable && corepack prepare pnpm@10 --activate

WORKDIR /app

# 先拷依赖清单，命中层缓存
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY apps/learner/package.json apps/learner/
COPY apps/admin/package.json apps/admin/
COPY packages/api/package.json packages/api/
COPY packages/shared/package.json packages/shared/
COPY packages/stores/package.json packages/stores/

RUN pnpm install --frozen-lockfile

# 拷全部源码
COPY . .

# 构建 learner + admin（VITE_API_BASE 默认 /api，由 Nginx 反代到 gateway）
ARG VITE_API_BASE=/api
ENV VITE_API_BASE=${VITE_API_BASE}
RUN pnpm build && pnpm build:admin

############################
# Runtime：Nginx 托管静态产物
############################
FROM nginx:1.25-alpine

# 安装 wget 供 healthcheck 使用
RUN apk add --no-cache wget

# 拷贝自定义 Nginx 配置
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 拷贝构建产物到 Nginx 静态目录
COPY --from=builder /app/apps/learner/dist /usr/share/nginx/html/learner
COPY --from=builder /app/apps/admin/dist /usr/share/nginx/html/admin

EXPOSE 80 8088

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:80/ || exit 1
