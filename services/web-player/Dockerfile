# ============================================================
# MediaHub Web Player - 多阶段构建
# （与 Admin 共用同样的 nginx 配置）
# ============================================================

FROM node:20-alpine AS builder

RUN npm config set registry https://registry.npmmirror.com

WORKDIR /app

COPY package*.json ./
# 显式安装 devDependencies：vue-tsc / vite / typescript 都在 devDependencies 里
# Docker 构建时 NODE_ENV 默认可能是 production，会跳过 devDeps 导致 build 失败
RUN npm ci --include=dev

COPY . .

ARG VITE_API_BASE_URL=http://localhost:3000
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

RUN npm run build

FROM nginx:1.27-alpine

RUN rm /etc/nginx/conf.d/default.conf
COPY nginx.conf /etc/nginx/conf.d/
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -q --spider http://localhost/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
