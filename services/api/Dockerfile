# ============================================================
# MediaHub API - 多阶段构建（DS920+ J4125 优化）
# ============================================================

# ---------- 阶段 1：编译 ----------
FROM golang:1.22-alpine AS builder

# 国内镜像加速（DS920+ 在国内环境下需要）
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=off

WORKDIR /app

# 单独 COPY go.mod 先下载依赖（利用 Docker 缓存）
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# CGO 关闭 → 静态二进制，Docker 镜像里不需要 glibc
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=$(date +%Y%m%d)" \
    -o /app/server \
    ./cmd/server

# ---------- 阶段 2：运行 ----------
FROM alpine:3.19

# 必要的运行时依赖 + Intel Quick Sync 用户态库（J4125 硬转码）
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    ffmpeg \
    libva-utils \
    intel-media-driver \
    bash \
    curl

# 时区
ENV TZ=Asia/Shanghai

# Intel GPU 设备（DS920+ 的 /dev/dri 会被 docker-compose 挂进来）
# 应用本身不直接用，应用容器内调用 ffmpeg 走 /dev/dri

WORKDIR /app
COPY --from=builder /app/server /app/server

# 非 root 运行（NAS 上避免权限问题）
RUN addgroup -S mediahub && adduser -S mediahub -G mediahub
USER mediahub

EXPOSE 3000

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -fsS http://localhost:3000/health || exit 1

ENTRYPOINT ["/app/server"]
