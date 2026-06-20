# Changelog

所有对本项目的显著变更都记录在此文件。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
本项目遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)，
使用 [Apache License 2.0](LICENSE)。

## [Unreleased]

### Added
- 推荐引擎：Content-based（标签相似度）+ Hybrid 框架
- 模板继承 + AB 测试 + 动态规则（时段/星期）
- HLS 异步转码（Quick Sync 硬转 + 切片）
- 多 Profile 管理（家庭成员 + 家长 PIN 锁）
- 续播支持（跨设备同步进度）
- 可视化布局编辑器（拖拽 + 多平台预览 + AB 配置）
- 流代理（直连 + Range + Content-Type 嗅探）
- 开源基础设施：Apache 2.0 LICENSE、README、CONTRIBUTING、CHANGELOG、CI/CD、Makefile
- 单元测试覆盖（scanner / apperr / layout model / recommend）
- 儿童模式（is_adult 字段 + FeedService 过滤）
- Web Player Profile 切换 UI
- 键盘快捷键（空格/←→/↑↓/F/M/J/L）
- 搜索页增强（类型筛选 + 评分筛选 + 排序）
- **下载管理：qBittorrent WebUI 客户端 + Downloader Service + 自动入库 watcher**
- **库自动扫描：ScannerService + 30 分钟 watcher + 手动触发 API**
- **字幕下载：SubHD 客户端 + Service + 媒资匹配（TMDB ID + 文件名）**
- **CMS Admin 下载管理页面**

### Changed
- 重构仓储层，支持事务和软删
- 优化 GORM 配置（连接池 + 预编译 SQL）
- 升级 LICENSE 从 MIT 到 Apache 2.0

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

### Changed
- 详情页 / 播放页通过 Intent 接收续播秒数（resumeSec）跳转
- PlaybackActivity 增加 `intent()` 工厂方法（替代裸 Intent）
- BrowseScreen 增加顶部工具栏（搜索 + 设置图标按钮）+ 加载/错误/空三态
- DetailActivity 增加续播提示（如果进度 > 0）
- Player.vue 集成 Toast（错误/HLS fallback/音量调节）

### Test Coverage
- 5 个测试包通过（apperr / cache / domain:layout / recommend / scanner）
- cache 包新增：disabled cache 调用次数测试 + 长 key hash 测试

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

[Unreleased]: https://github.com/mediahub/mediahub/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/mediahub/mediahub/releases/tag/v0.1.0
