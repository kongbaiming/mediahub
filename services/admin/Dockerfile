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
RUN npm ci

COPY . .

# 构建参数：API 地址
ARG VITE_API_BASE_URL=http://localhost:3000
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
