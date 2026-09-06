# ============================================================
# web-admin — 多阶段构建
# 构建阶段：node:22-alpine + pnpm（vue-tsc 类型检查 + vite 构建）
# 运行阶段：nginx 托管 dist 静态资源，/api 反向代理到 compose 服务 server:8088
# （nginx.conf 见 ./nginx.conf，为生产镜像专用配置）
# ============================================================

# ---- 构建阶段 ----
FROM node:22-alpine AS builder

# npm/pnpm 镜像源；默认 npmmirror（本环境 proxy.golang.org 不可达，网络环境类似），
# 如需官方源：docker compose build --build-arg NPM_REGISTRY=https://registry.npmjs.org
ARG NPM_REGISTRY=https://registry.npmmirror.com

RUN npm config set registry "$NPM_REGISTRY" \
    && corepack enable

WORKDIR /app

# 先拷贝依赖清单，利用构建缓存
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

# 再拷贝源码；.dockerignore 已排除 node_modules/dist 等
COPY . .

# 生产构建：VITE_API_URL=/（与 .env.production 一致）。注意不能用 /api/v1：
# API 模块 url 自带 VITE_API_PREFIX，baseURL 再带前缀会拼成 /api/v1/api/v1/...
# nginx 会把 /api/ 原样转发给 server，路径以 /api/v1 开头即可命中 Go 路由。
ARG VITE_API_URL=/
ENV VITE_API_URL=$VITE_API_URL

RUN pnpm build

# ---- 运行阶段 ----
FROM nginx:1.27-alpine

# 生产镜像专用 nginx 配置（托管 dist + /api 反代 server:8088）
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80