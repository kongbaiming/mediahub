# MediaHub

> 🏠 自建家庭媒资中心 — 从下载到推荐到播放端布局，全部掌控在自己手里。

[![License](https://img.shields.io/badge/license-Apache_2.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://golang.org)
[![Vue](https://img.shields.io/badge/Vue-3-42b883.svg)](https://vuejs.org)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED.svg)](docker-compose.yml)
[![GitHub release](https://img.shields.io/github/release/mediahub/mediahub.svg)](https://github.com/mediahub/mediahub/releases)

[English](./README.en.md) | 简体中文

---

## ✨ 特色

- **🎬 全自建栈**：Go 后端 + Vue 3 前端 + 原生 Android TV / tvOS 客户端，无中间商
- **🎨 可视化布局编辑器**：像 Netflix / Apple TV+ 一样编辑播放端 UI（横滑海报墙、Banner、专题合集）
- **🤖 智能推荐**：Content-based + 协同过滤 + Hybrid 算法，跨 Profile 个性化
- **📺 多端原生播放**：Web / Android TV（Kotlin Leanback）/ tvOS（SwiftUI），各自焦点系统
- **🔍 自动刮削**：TMDB 元数据 + 海报 + 演员 + 季/集信息
- **📡 灵活分发**：模板继承（基础 → 家庭版 → 儿童版）、AB 测试、动态规则（按时段/星期）
- **💾 NAS 友好**：专为群晖 DS920+ (J4125) 优化（Quick Sync 硬转 + 单二进制）
- **🔓 Apache 2.0**：宽松许可，明确专利授权，可商用

## 🖼️ 截图

<!-- TODO: 添加截图 -->
<p align="center">
  <i>占位 - 后续补充：CMS 编辑器、Web Player 首页、播放页</i>
</p>

## 🚀 快速开始

### 前置条件

| 组件 | 要求 |
|---|---|
| **NAS** | 群晖 DS920+ 或同级（推荐 8GB+ RAM） |
| **Docker** | Container Manager 套件 |
| **存储** | 媒资盘 + 下载盘 |
| **TMDB API Key** | 免费申请：https://www.themoviedb.org/settings/api |

### 5 分钟启动

```bash
# 1. 克隆仓库
git clone https://github.com/mediahub/mediahub.git
cd mediahub

# 2. 复制环境变量模板
cp .env.example .env
# 编辑 .env，填入 TMDB_API_KEY 和各密码

# 3. 启动所有服务
docker compose up -d

# 4. 打开浏览器
# CMS Admin:   http://nas-ip:8080   (admin / admin123)
# Web Player:  http://nas-ip:8081
# API 文档:    http://nas-ip:3000/swagger/index.html
```

### 完整启动验证（含功能测试）

👉 **[docs/GETTING-STARTED.md](docs/GETTING-STARTED.md)** — 1.5 小时从零跑通所有 W1-W9 功能的完整指南

包含：环境检查 → 部署 → 健康验证 → TMDB 刮削 → 布局编辑器 → Web Player → Profile/儿童模式 → 下载/字幕 → Android TV → 性能验证 → 12 张截图清单 → 常见问题。

### 从源码开发

```bash
# 后端
make api-deps && make api-dev

# 前端 Admin
make admin-deps && make admin-dev    # http://localhost:5173

# 前端 Web Player
make web-deps && make web-dev        # http://localhost:5174

# 跑全套测试
make ci
```

## 📐 架构

```
┌─────────────────────── Synology NAS ────────────────────────┐
│                                                              │
│  ┌────────────┐  ┌────────────┐  ┌─────────────────────┐   │
│  │ Downloader │→ │ Media Store │→ │ Media Engine        │   │
│  │ (qBit/DS)  │  │ /volume1/  │  │ (Scraper + FFmpeg)  │   │
│  └────────────┘  │   media/    │  └──────────┬──────────┘   │
│                  └────────────┘             │              │
│                                               ▼              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  MediaHub API (Go + Gin)                              │  │
│  │  - 媒资 CRUD + 搜索                                   │  │
│  │  - 布局 JSON 模型（10 种数据源）                       │  │
│  │  - 推荐引擎（CF + Content + Hybrid）                  │  │
│  │  - JWT 鉴权 + 多 Profile + 续播                       │  │
│  └──────────────────────┬────────────────────────────────┘  │
│                         │                                    │
│         ┌───────────────┼───────────────┐                   │
│         ▼               ▼               ▼                   │
│  ┌──────────┐    ┌──────────┐    ┌──────────────┐          │
│  │ Web App  │    │ Android  │    │   tvOS       │          │
│  │ (Vue 3)  │    │   TV     │    │ (SwiftUI)    │          │
│  │          │    │ (Kotlin) │    │              │          │
│  └──────────┘    └──────────┘    └──────────────┘          │
└──────────────────────────────────────────────────────────────┘
```

详细架构：[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

## 📦 项目结构

```
mediahub/
├── services/
│   ├── api/              # Go + Gin 后端
│   │   ├── cmd/server/   # 入口
│   │   ├── internal/     # domain → repository → service → handler
│   │   ├── migrations/   # SQL migration
│   │   └── docs/         # OpenAPI 规范
│   ├── admin/            # CMS Admin (Vue 3 + Element Plus)
│   ├── web-player/       # Web Player (Vue 3 + hls.js)
│   ├── android-tv/       # Android TV (Kotlin + Leanback)  [计划]
│   └── tvos/             # tvOS (SwiftUI + AVPlayer)       [计划]
├── docs/                 # 方案 + 架构文档
├── .github/              # GitHub Actions + Issue 模板
├── docker-compose.yml
├── Makefile              # 一键开发命令
└── README.md
```

## 🛣️ 路线图

查看 [CHANGELOG.md](CHANGELOG.md) 了解历史变更。

- [x] **Phase 1** - 服务端 + CMS（W1-W2）：基础 CRUD + 布局编辑器 + 多端差异化
- [ ] **Phase 2** - 推荐引擎（W3）：Content-based + 协同过滤 + Hybrid
- [ ] **Phase 3** - Web Player 完整版（W4-W5）
- [ ] **Phase 4** - 下载管理器 + 字幕自动下载（W6）
- [ ] **Phase 5** - Android TV 客户端（W7-W9）
- [ ] **Phase 6** - tvOS 客户端（W10-W12）

## 🤝 贡献

我们欢迎所有形式的贡献！详见 [CONTRIBUTING.md](CONTRIBUTING.md)。

- 🐛 报告 Bug：提交 [Issue](.github/ISSUE_TEMPLATE/bug_report.md)
- 💡 提议功能：提交 [Issue](.github/ISSUE_TEMPLATE/feature_request.md)
- 🔧 提 PR：Fork 仓库 → 创建分支 → 提 PR
- 📖 改进文档：直接编辑 `.md` 文件
- 🌍 翻译：目前只有英文/简体中文，欢迎其他语言

详见 [CONTRIBUTING.md](CONTRIBUTING.md) 了解完整的开发流程。

## 📜 许可证

本项目基于 [Apache License 2.0](LICENSE) 开源。

## 🙏 致谢

- [TMDB](https://www.themoviedb.org/) - 影视元数据 API
- [Gin](https://github.com/gin-gonic/gin) - Go Web 框架
- [Vue.js](https://vuejs.org/) - 前端框架
- [Element Plus](https://element-plus.org/) - Vue 3 UI 库
- [FFmpeg](https://ffmpeg.org/) - 视频处理
- [Asynq](https://github.com/hibiken/asynq) - Go 任务队列

## 💬 社区

- GitHub Discussions: [讨论区](https://github.com/mediahub/mediahub/discussions)
- Issues: [问题反馈](https://github.com/mediahub/mediahub/issues)

---

⭐ 如果这个项目对你有帮助，请在 GitHub 上点个 Star！
