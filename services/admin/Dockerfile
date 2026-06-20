# ============================================================
# MediaHub CMS Admin - 多阶段构建
# ============================================================

# ---------- 阶段 1：构建 ----------
FROM node:20-alpine AS builder

# 国内镜像加速
RUN npm config set registry https://registry.npmmirror.com

WORKDIR /app

# 单独 COPY package.json 先安装依赖（利用 Docker 缓存）
COPY package*.json ./
# 显式安装 devDependencies：vue-tsc / vite / typescript 都在 devDependencies 里
# Docker 构建时 NODE_ENV 默认可能是 production，会跳过 devDeps 导致 build 失败
RUN npm ci --include=dev

COPY . .

# 构建参数：API 地址
# 默认留空：axios baseURL 为空时请求会发到当前 origin，由本容器的 nginx
# 反代 /api/ 到 api:3000。这样不管用户用 nas.local / 内网 IP / SSH 端口转发
# 访问都能正常工作（浏览器永远打相对路径，不会拼成 localhost:3000）。
#
# 仅本地开发（npm run dev）需要显式指定 API 地址：
#   docker build --build-arg VITE_API_BASE_URL=http://localhost:3000 .
ARG VITE_API_BASE_URL=
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

RUN npm run build

# ---------- 阶段 2：nginx 服务 ----------
FROM nginx:1.27-alpine

# 移除默认配置
RUN rm /etc/nginx/conf.d/default.conf

COPY nginx.conf /etc/nginx/conf.d/
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -q --spider http://localhost/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
