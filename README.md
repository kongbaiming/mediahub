# MediaHub

> 自建家庭媒资中心 — 从下载、刮削、布局编辑到多端播放，全部掌控在自己手里。

[![License](https://img.shields.io/badge/license-Apache_2.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://golang.org)
[![Vue](https://img.shields.io/badge/Vue-3-42b883.svg)](https://vuejs.org)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED.svg)](docker-compose.yml)
[![GitHub release](https://img.shields.io/github/v/release/kongbaiming/mediahub)](https://github.com/kongbaiming/mediahub/releases)

简体中文

---

## 特色

- **全自建栈**：Go 后端 + Vue 3 CMS / Web Player + Android TV 客户端（Kotlin）
- **可视化布局编辑器**：拖拽配置首页行（Hero、Shelf、猜你喜欢等），Web / TV 多端预览与发布
- **猜你喜欢**：观影历史 + TMDB 推荐 + 库内兜底，可写入布局 Feed
- **智能推荐**：Content-based + Hybrid；Feed Redis 缓存，布局变更自动失效
- **多端播放**：Web（hls.js）/ Android TV（ExoPlayer）；MKV 自动 HLS 转码，支持边转边播
- **NAS 友好**：群晖 DS920+ 优化（Quick Sync 硬转、`/media` + `/downloads` 双路径）
- **下载入库**：qBittorrent 集成，完成下载后自动扫描入库
- **Apache 2.0**：可商用

## 快速开始

### 前置条件

| 组件 | 要求 |
|---|---|
| **NAS / 服务器** | x86_64，推荐 4GB+ RAM |
| **Docker** | Compose v2 |
| **存储** | 媒资目录 + 下载目录（可分开挂载） |
| **TMDB API Key** | https://www.themoviedb.org/settings/api |

### 5 分钟启动

```bash
git clone https://github.com/kongbaiming/mediahub.git
cd mediahub

cp .env.example .env
# 编辑 .env：TMDB_API_KEY、各密码、NAS 挂载路径

docker compose up -d --build
```

默认端口（可在 `.env` 修改）：

| 服务 | 默认端口 | 说明 |
|---|---|---|
| CMS Admin | `8080` | 布局 / 媒资 / 下载管理 |
| Web Player | `8081` | 家庭播放端 |
| API | `3000` | REST API + Swagger |
| qBittorrent | `8082` | 下载 WebUI |

浏览器访问：

- CMS：`http://<NAS-IP>:8080`（默认 `admin` / `admin123`，请尽快改密）
- Web Player：`http://<NAS-IP>:8081`
- API 文档：`http://<NAS-IP>:3000/swagger/index.html`

### 完整验证指南

👉 **[docs/GETTING-STARTED.md](docs/GETTING-STARTED.md)** — 从部署到布局发布、播放、Profile、下载的逐步 checklist

### 从源码开发

```bash
make api-deps && make api-dev      # 后端 :3000
make admin-deps && make admin-dev  # CMS  :5173
make web-deps && make web-dev      # Web  :5174
make ci                            # 本地跑与 CI 相同的检查
```

## 架构概览

```
Downloader (qBit) → /downloads ─┐
                                 ├→ Scanner → PostgreSQL
Media 目录 /media ──────────────┘       ↓
                              MediaHub API (Go)
                    ┌───────────┼───────────┐
                    ▼           ▼           ▼
              CMS Admin    Web Player   Android TV
              (布局编辑)    (hls.js)    (Leanback)
```

## 项目结构

```
mediahub/
├── services/
│   ├── api/           # Go + Gin 后端
│   ├── admin/         # CMS Admin (Vue 3 + Element Plus)
│   ├── web-player/    # Web Player (Vue 3 + hls.js)
│   └── android-tv/    # Android TV 客户端
├── docs/              # 部署与方案文档
├── .github/workflows/ # CI + Release
├── docker-compose.yml
└── Makefile
```

## 版本与路线图

当前最新版本：**[v0.3.0](https://github.com/kongbaiming/mediahub/releases/tag/v0.3.0)** — 详见 [CHANGELOG.md](CHANGELOG.md)

| 阶段 | 状态 | 内容 |
|---|---|---|
| Phase 1 服务端 + CMS | ✅ | CRUD、布局编辑器、JWT、刮削 |
| Phase 2 推荐 + Feed | ✅ | 猜你喜欢、Feed 缓存、模板继承 |
| Phase 3 Web Player | ✅ | Hero、续播、HLS、Profile |
| Phase 4 下载 / 扫描 | ✅ | qBit、自动入库、库扫描 |
| Phase 5 Android TV | 🚧 | 骨架可用，持续完善 |
| Phase 6 tvOS | 📋 | 计划中 |

## 升级

```bash
cd /volume1/docker/mediahub   # 或你的部署目录
git pull
docker compose build api admin web-player
docker compose up -d api admin web-player
```

布局或 Feed 有缓存时，重启 `api` 会自动刷新首页 Feed；也可手动清 Redis：`KEYS mediahub:feed:*`

## 贡献

详见 [CONTRIBUTING.md](CONTRIBUTING.md)。Bug / 功能建议请走 [Issues](https://github.com/kongbaiming/mediahub/issues)。

## 许可证

[Apache License 2.0](LICENSE)

## 致谢

- [TMDB](https://www.themoviedb.org/) · [Gin](https://github.com/gin-gonic/gin) · [Vue.js](https://vuejs.org/) · [FFmpeg](https://ffmpeg.org/) · [Asynq](https://github.com/hibiken/asynq)
