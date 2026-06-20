# ============================================================
# MediaHub API - 多阶段构建（DS920+ J4125 优化）
# ============================================================

# ---------- 阶段 1：编译 ----------
FROM golang:1.22-alpine AS builder

# 国内镜像加速（DS920+ 在国内环境下需要）
ENV GOPROXY=https://goproxy.cn,direct
# 注意：不要设 GOSUMDB=off！
# GOTOOLCHAIN=auto 下载工具链时**强制**需要 sumdb 校验，
# 关掉 GOSUMDB 会导致工具链下载失败（即使有 GOPROXY 镜像也不行）。
# 默认的 sum.golang.org 在国内偶有延迟，但能保证工具链二进制不被篡改。
#
# 自动下载工具链（go.mod 里如果声明了 >= 1.25，会自动下 Go 1.25 工具链来编译）
# 镜像里的 Go 1.22 只是 bootstrap，运行时按 go.mod 要求自动升级
ENV GOTOOLCHAIN=auto

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

# 以 root 运行：ffmpeg QSV 需访问 /dev/dri；HLS 缓存目录需可写
# docker-compose 可通过 user: "1000:1000" 覆盖（软解-only 时可非 root）

WORKDIR /app
COPY --from=builder /app/server /app/server

EXPOSE 3000

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -fsS http://localhost:3000/health || exit 1

ENTRYPOINT ["/app/server"]
