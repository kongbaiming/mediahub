# Changelog

所有对本项目的显著变更都记录在此文件。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
本项目遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)，
使用 [Apache License 2.0](LICENSE)。

## [Unreleased]

_暂无_

## [0.3.0] - 2026-06-21

### Added
- **猜你喜欢**：基于观影历史 + TMDB 相似推荐 + 库内兜底，可作为布局数据源 `guess-you-like`
- 数据库迁移自动为 Web/TV 首页布局插入「猜你喜欢」行；API 启动时幂等补全
- **播放前校验**：检测文件大小与视频头（MKV/MP4 等），跳过 qBit 未完成下载的占位文件
- 扫描入库时跳过 `.part`、`.!qB` 等临时文件

### Fixed
- **CMS 布局编辑器空白**：`vue-draggable-plus` 改用 `v-for` 子元素渲染（不支持 `#item` 插槽）
- **布局预览**：继承父布局行合并；预览 API 免登录
- **Feed 缓存**：布局保存、启动补全、发布时强制失效 Redis 缓存，避免播放端长期显示旧版布局
- **Web Player 首页**：移除硬编码「热门推荐/继续观看」，严格展示 CMS 发布的 Feed 行
- **HLS 播放**：支持 `DOWNLOAD_ROOT`（`/downloads`）下媒资；修复 incomplete playlist 误判与路由重定向
- **HLS 转码性能**：默认 480p / 2500k / ultrafast，AAC copy；支持边转边播
- **CI**：修复 Go 测试、Admin lint、Web Player type-check；GHCR 镜像路径改为 `github.repository_owner`

### Changed
- Web Player `tsconfig` 与 Admin 对齐，修复 `vue-tsc` 类型检查
- CI 后端 Lint 改为 `go vet`（兼容 `go.mod` toolchain）

## [0.2.0] - 2026-06-20

### Added (W7-W9：原生客户端 + 体验优化)
- **Android TV 客户端骨架**：Kotlin + Compose for TV + Media3 ExoPlayer + Leanback
- **Android TV 首次配置**：SetupActivity（输入 NAS API URL + 健康检查）
- **Android TV 设置页**：SettingsActivity（API / Profile 切换 / 播放 / 关于）
- **Android TV 详情页**：DetailActivity（背景大图 + 海报 + 元数据 + 续播 + 操作按钮）
- **Android TV 搜索页**：SearchActivity（debounce + grid + 自动聚焦）
- **Android TV 性能优化**：Coil ImageLoader（25% 内存 + 250MB 磁盘）+ ProGuard + R8
- **Android TV Codemagic CI**：debug + release 双 workflow 自动构建
- **Web Player Toast 系统**：4 种类型（info/success/warning/error）+ 全局 Host
- **Web Player Skeleton 加载**：Shimmer 动画 + 卡片骨架
- **Web Player EmptyState**：自定义空状态（图标 + 描述 + 操作按钮）
- **CMS Admin Skeleton**：媒资列表 shimmer 加载 + 加载结果提示
- **后端缓存层**：Redis GetOrLoad + singleflight 防击穿 + TTL 抖动防雪崩
- **后端 Feed 缓存**：5 分钟 TTL + 布局发布自动失效
- **后端 Metrics 接口**：`/metrics` 返回 Go runtime + DB 连接池 + Redis 状态
- **后端搜索 API**：`GET /api/v1/search`（q + type + limit）
- 推荐引擎：Content-based（标签相似度）+ Hybrid 框架
- 模板继承 + AB 测试 + 动态规则（时段/星期）
- HLS 异步转码（Quick Sync 硬转 + 切片）
- 多 Profile 管理（家庭成员 + 家长 PIN 锁）
- 续播支持（跨设备同步进度）
- 可视化布局编辑器（拖拽 + 多平台预览 + AB 配置）
- 下载管理：qBittorrent WebUI 客户端 + 自动入库 watcher
- 库自动扫描：ScannerService + 30 分钟 watcher + 手动触发 API
- 字幕下载：SubHD 客户端 + Service + 媒资匹配
- CMS Admin 下载管理页面

### Changed
- 详情页 / 播放页通过 Intent 接收续播秒数（resumeSec）跳转
- BrowseScreen 增加顶部工具栏（搜索 + 设置图标按钮）+ 加载/错误/空三态
- Player.vue 集成 Toast（错误/HLS fallback/音量调节）
- 重构仓储层，支持事务和软删
- 升级 LICENSE 从 MIT 到 Apache 2.0

## [0.1.0] - 2026-06-20

### Added
- 项目骨架：Go 后端 + Vue 3 CMS Admin + Vue 3 Web Player
- Docker Compose 一键部署（postgres + redis + api + admin + web + qbittorrent）
- 媒资 CRUD + 过滤 + 分页
- TMDB 自动刮削（电影/剧集/季/集 + ffprobe 视频信息）
- Asynq 异步任务队列（刮削/缩略图/扫描）
- JWT 鉴权（HS256 + bcrypt 密码哈希）
- Swagger OpenAPI 文档
- 文件扫描（fsnotify 监听 + Sonarr 风格命名解析）
- 默认管理员账户（admin / admin123）

[Unreleased]: https://github.com/kongbaiming/mediahub/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/kongbaiming/mediahub/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/kongbaiming/mediahub/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/kongbaiming/mediahub/releases/tag/v0.1.0
