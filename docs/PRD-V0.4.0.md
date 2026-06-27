# MediaHub v0.4.0 — 体验闭环 PRD

| 项     | 值                                          |
| ------ | ------------------------------------------- |
| 文档版本 | v1.0                                        |
| 目标版本 | v0.4.0                                      |
| 主题     | 体验闭环(Scraping Center · 4K 优先 · 一致性) |
| 维护     | 产品 / 后端 / 前端联合                       |
| 基线     | v0.3.0(猜你喜欢、Feed 缓存、HLS 480p、CI)  |
| 下游     | v0.5.0(多端产品化)· v0.6.0(智能与运营)   |
| 关联文档 | PRD-ROADMAP.md · MEDIA-DOMAIN.md · OTT-GAPS.md · GETTING-STARTED.md |

---

## 1. 背景与问题陈述

v0.3.0 把"猜你喜欢""Feed 缓存""HLS 480p""CI 绿"做了基线,但**端到端体验还有三处断点**:

1. **入库不闭环**:qBittorrent 下载完成后,大量任务停留在「99% / 残留 `.part`」状态,scanner 看不懂进度;`qB progress=100%` 后才触发入库,导致一个剧集前几集已入库、后几集始终 pending。
2. **播放不闭环**:
   - 4K HEVC 直连不通过时,客户端只能拿到 480p 转码,内网带宽浪费且启动慢;
   - 刮削/转码/重命名过程中没有「进度 + 准备中」反馈,前端只看到 spinner;
   - 剧集详情只能跳到首集,无法直接选季/集;
   - ffprobe 字段缺失,转码参数只能猜。
3. **CMS 与播放不一致**:发布布局后,Web/TV 端最长 5 分钟才刷新,体感像"没生效";Profile 切换后端缺少 cache invalidation;Admin 端的 type-check 没有进 CI 拦截。

v0.4.0 目标就是把这三处断点**全部闭合**,交付一个从"下载 → 刮削 → 布局 → 播放 → 续播"端到端流畅的体验。

---

## 2. 目标 / 非目标

### 2.1 目标(必达成)

| # | 目标                                                                       | 度量                                                                  |
| - | -------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| G1 | **下载到入库零等待**:qBit progress=100% 后 ≤60s 自动入库                  | `downloader.done → media.exist` P95 ≤ 60s                            |
| G2 | **刮削可观测可重试**:所有刮削任务入队,失败原因结构化,支持单条/批量重试    | 刮削看板可见 pending/failed ≥ 100%,可一键重试                       |
| G3 | **4K 优先播放**:客户端优先尝试直连/流复制,失败 fallback 到转码            | 内网 4K HEVC 命中率 ≥ 80%,首帧 ≤ 10s                                |
| G4 | **HLS 任务持久化**:转码任务存 Redis,API 重启不丢                          | `docker restart api` 后任务不丢,客户端可继续轮询                     |
| G5 | **剧集选集**:详情页支持按季/集跳转,带 `episode_id` 直链                   | 单集深链可直跳并自动播放                                              |
| G6 | **Feed 实时一致**:CMS 发布/失效后,Web/TV ≤ 5s 内刷新                     | 布局保存到 Feed 呈现 P95 ≤ 5s                                       |
| G7 | **CI 拦截回归**:Web Player / Admin `type-check` 进 CI,失败阻塞合并        | main 分支 type-check 必绿                                            |

### 2.2 非目标(明确不做)

| # | 不做                            | 原因                                  | 替代/延后            |
| - | ------------------------------- | ------------------------------------- | -------------------- |
| N1 | 公网多租户 SaaS                | 定位家庭自建                          | 见 PRD-ROADMAP       |
| N2 | 替换 Jellyfin 全协议(DLNA 等) | 范围控制                              | 不替代,仅做核心域     |
| N3 | 多用户/OAuth/2FA               | 当前 v0.4 范围                       | 推 v0.5 账号安全整章  |
| N4 | 海报 CDN/镜像分发             | 家庭场景下命中率低,运维成本高       | 推 v0.5+ P2           |
| N5 | qBit 完成后 copy/hardlink     | 跨卷语义复杂,需 NAS 文件系统支持     | 推 v0.5+ P2           |
| N6 | 广告/商业化/DRM               | 与家庭定位冲突                        | 长期不做              |

---

## 3. 范围与优先级

> 沿用 `PRD-ROADMAP.md` §5.1 的 A1–A6 / B1–B5 编号,本 PRD 展开详细需求。

### 3.1 P0 — 必做

| # | 需求                              | 所属模块              | 简述                                                                |
| - | --------------------------------- | --------------------- | ------------------------------------------------------------------- |
| A1 | 刮削中心(Scraping Center)         | `services/scraper`    | CMS 队列看板(pending/failed)、单条/批量重试、失败原因结构化、回到入队 |
| A2 | 下载完成再入库                    | `services/downloader` | 监听 qBit `progress=100%`,跳过 `.part`/`.!qB`,触发 scanner          |
| A3 | 4K/直连优先播放                  | `services/api` + player | probe API(ffprobe)、HLS 流复制(`-c:v copy`)、客户端直连 fallback   |
| A4 | HLS 任务持久化                    | `services/transcoder` | Redis 存任务状态,API 重启不丢任务,客户端可续轮询                    |
| A5 | Web 剧集选集                      | Web Player + API      | 详情页季/集列表,带 `episode_id` 深链直跳并自动播放                  |
| A6 | Feed/布局一致                   | CMS Admin + Cache     | 布局保存即失效缓存,`X-Profile-ID` 强制失效,5s 内全端刷新           |

### 3.2 P1 — 应做

| # | 需求                              | 所属模块              | 简述                                                                |
| - | --------------------------------- | --------------------- | ------------------------------------------------------------------- |
| B1 | 猜你喜欢入库闭环                 | `services/recommend`  | 库外 TMDB → 「加入媒体库」向导,扫码/一键入队                       |
| B2 | recommendations 落库            | `services/recommend`  | 推荐结果缓存到 `recommendations` 表,减少 TMDB 调用                  |
| B3 | 转码进度 & Loading               | Transcoder + Player   | 播放器展示「正在准备 4K 流…」,对齐 HLS 切片就绪状态                  |
| B4 | ffprobe 补全字段                | `services/scanner`    | 入库写入 `file_size`/`duration`/`bitrate`/`video_codec` 等         |
| B5 | Admin type-check 进 CI          | CI                    | Web Player/Admin/Android TV 三端 `tsc --noEmit` 纳入 GitHub Actions |

### 3.3 P2 — 可选(v0.5+ 评估)

- 海报 CDN/镜像(本地缓存 + TMDB 兜底)
- qBit 完成后 copy/hardlink 到 `/media`
- QSV 不可用时的 CMS 提示(降级到 libx264)
- 续播跨设备同步(Redis 存 `profile_id + media_id` → `progress`)

---

## 4. 用户场景

### 4.1 家庭管理员(运维)

| 场景                                           | 期望                                                                            |
| ---------------------------------------------- | ------------------------------------------------------------------------------- |
| S1: 看到某剧第 3 集入库失败                    | 打开 CMS 刮削中心,看到第 3 集在 failed 列表,原因 "TMDB no match",点击重试      |
| S2: qBit 下载一个 50GB 4K 蓝光原盘            | 进度 100% 后,60s 内自动出现在媒资库,海报/简介/季/集完整                          |
| S3: TV 端播放 4K HEVC 一直转圈                 | 客户端优先尝试直连;若失败,展示「正在准备 4K 流…」Loading,不再看到白屏           |
| S4: 在 CMS 改了一行布局顺序                    | 5s 内 Web Player 和 Android TV 都按新顺序展示                                    |
| S5: PR 合入 type-check 失败                    | CI 红灯,无法合入 main                                                            |

### 4.2 家庭成员(消费)

| 场景                                       | 期望                                                                          |
| ------------------------------------------ | ----------------------------------------------------------------------------- |
| T1: 看一个剧,想直接跳到第 5 季第 8 集      | 详情页有季/集列表,点击后直接播放该集                                          |
| T2: 在手机上看到一半,换 TV 继续            | 进度同步(本期 v0.4 仅本机续播,跨设备放到 v0.5)                              |
| T3: 看 4K 演示片,卡在转码                  | 客户端自动选择流复制/直连,不再硬转码                                          |
| T4: 切换儿童 Profile                        | 5s 内 feed 失效,新 Profile 列表生效,is_adult 媒资消失                         |

### 4.3 开发者(扩展)

| 场景                                       | 期望                                                                          |
| ------------------------------------------ | ----------------------------------------------------------------------------- |
| D1: 加一个刮削源(Jellyfin metadata agent) | 走 `ScrapingService` 接口,自动入队,无需改 CMS                                |
| D2: 加一个新转码预设(720p H.264)           | 在 `transcoder/profiles.yaml` 追加,API 自动暴露                              |
| D3: CI 改 type-check                       | 在 `.github/workflows/typecheck.yml` 调整,所有 PR 自动跑                      |

---

## 5. 功能需求详述

### A1 · 刮削中心(Scraping Center)

#### A1.1 看板

- 路径:CMS `/admin/scrape`
- 视图:三栏 `pending` / `in_progress` / `failed`,可切换。
- 字段:`media_id` / `title` / `source`(TMDB/Manual/Bangumi) / `attempt` / `last_error` / `updated_at`
- 操作:单条重试、批量重试(多选)、全量重试、强制刷新元数据。

#### A1.2 队列

- 实现:在 `services/scraper` 内维护 `scrape_jobs` 表(已存在,扩展字段)。
- 字段新增:`last_error_code` / `last_error_message` / `attempt_count` / `next_retry_at` / `worker_id`
- 状态机:`queued → running → done | failed | retry`
- 失败分类:`tmdb_no_match` / `tmdb_rate_limit` / `network_error` / `parse_error` / `internal_error`

#### A1.3 API

```
GET    /api/v1/scrape/jobs?status=pending&page=1&page_size=20
POST   /api/v1/scrape/jobs/:id/retry
POST   /api/v1/scrape/jobs/batch-retry     body: { ids: [1,2,3] }
POST   /api/v1/scrape/jobs/retry-all-failed
GET    /api/v1/scrape/stats                 # { pending, running, failed, done_24h }
```

#### A1.4 验收

- 1000 个待刮削任务,CMS 看板分页流畅(≤200ms P95)
- 失败原因结构化,前端可读
- 一键重试不会重复入队(幂等)

### A2 · 下载完成再入库

#### A2.1 监听

- 改造 `downloader-watcher`:
  - 监听 qBittorrent `torrentfinished` 与 `progress=100%` 事件
  - 跳过文件扩展名:`.part` / `.!qB` / `.partial` / `.tmp`
  - 等所有文件都 `100%` 才触发 scanner

#### A2.2 入库路径

```
qB progress=100%
  → watcher enqueueScan(torrent_hash)
    → scanner.scanPath(/downloads/<title>/)
      → ffprobe 抽取元数据(见 B4)
        → scraper.enqueue(media_id, source=auto)
          → scrape_jobs 表
```

#### A2.3 验收

- 一个 8 集剧集,8 集全部下载完成后,≤60s 全部入库
- 残留 `.part` 文件不触发入库
- Watcher 异常后,可手动 `POST /api/v1/scanner/scan?path=/downloads/<x>` 补救

### A3 · 4K/直连优先播放

#### A3.1 probe API

```
GET /api/v1/media/:id/probe
→ {
     file_size, duration, bitrate,
     video_codec, video_profile, video_level,
     audio_codec, audio_channels, audio_language,
     width, height, hdr: "hdr10" | "hlg" | "sdr"
   }
```

#### A3.2 播放策略决策

```
if client supports codec && (codec in {h264, hevc, av1}):
    return direct_url  # 客户端直连
elif request has Range && duration < 30min:
    return hls_stream_copy  # -c:v copy, 无转码
else:
    return hls_transcode    # QSV H.264 / 720p / 2500k
```

#### A3.3 客户端改动

- Web Player:按决策依次试 direct → stream_copy → transcode
- Android TV:同上,ExoPlayer 优先 HLS direct
- Loading 提示文案:「正在准备 4K 流…」(对齐 B3)

#### A3.4 验收

- 内网 4K HEVC 命中率 ≥ 80%
- 首帧 ≤ 10s(流复制路径 ≤ 3s)
- 决策逻辑单测覆盖率 ≥ 90%

### A4 · HLS 任务持久化

#### A4.1 存储

- Redis Key:`transcode:task:<task_id>`
- Value(JSON):`{ media_id, profile, status, progress, segment_index, error }`
- TTL:7 天(任务完成后过期)

#### A4.2 状态机

```
queued → preparing → transcoding → packaging → done
                                      ↘ failed
                                      ↘ canceled
```

#### A4.3 API 改动

```
POST /api/v1/transcode/start     → { task_id, status: "queued" }
GET  /api/v1/transcode/:task_id  → { task_id, status, progress, manifest_url }
POST /api/v1/transcode/:task_id/cancel
```

- API 重启时:`On boot → Redis SCAN transcode:task:* → rehydrate in-memory map`
- 已 `done` 的任务保留 7 天供回看

#### A4.4 验收

- `docker restart api` 后,客户端轮询 task_id 仍能拿到正确状态
- 客户端可断网后重连续播(manifest_url 仍有效)

### A5 · Web 剧集选集

#### A5.1 详情页

- Web Player 详情页:
  - 顶部 Hero(海报/简介/元数据,沿用 v0.3.0)
  - 中部「季选择器」:`Season 1 / Season 2 / ...`(横滑卡片)
  - 下部「集列表」:`Episode 1 / Episode 2 / ...`(行项 + 时长 + 缩略图)
  - 点击集 → `/player/<media_id>?episode=<episode_id>`

#### A5.2 API

```
GET /api/v1/media/:id/seasons                  → [{ season_number, episode_count, poster }]
GET /api/v1/media/:id/seasons/:n/episodes      → [{ episode_id, number, title, duration, thumbnail }]
GET /api/v1/episode/:id                        → { media_id, season, episode, title, source_url }
```

#### A5.3 验收

- 单集深链 `/player/<m>?episode=<e>` 直跳并自动播放
- 季切换无白屏,集列表懒加载

### A6 · Feed/布局一致

#### A6.1 失效策略

- CMS 发布布局:`POST /api/v1/layouts/:id/publish` → `Redis DEL feed:layout:<layout_id> feed:profile:*`
- Profile 切换:`POST /api/v1/profiles/switch` → `Redis DEL feed:profile:<new_profile_id>`
- 主动失效:`POST /api/v1/cache/invalidate` body: `{ keys: [...] }`

#### A6.2 客户端

- Web Player:布局保存后,Admin 通过 `EventSource` 推送 → 客户端 ≤5s 拉新
- Android TV:轮询 `GET /api/v1/feed/version`,不一致即拉新

#### A6.3 验收

- CMS 改一行 → Web Player ≤5s 看到新顺序
- Profile 切换 ≤5s 生效
- 失效不会误删其他 Profile 的缓存

### B1 · 猜你喜欢入库闭环

#### B1.1 向导

- 路径:CMS `/admin/recommendations/library-missing`
- 视图:TMDB 推荐但库里没有的媒资列表
- 操作:点击「加入媒体库」→ 弹出「磁力链/BT 种子/TMDB ID」三种入队方式

#### B1.2 实现

- 新增 `services/recommend/library_inbox` 服务
- 入库即创建 `scrape_jobs` 记录,复用 A1 看板

#### B1.3 验收

- 看到推荐 50 部未入库,10 部点击入库,5 分钟内出现在待刮削列表

### B2 · recommendations 落库

#### B2.1 表设计

```sql
CREATE TABLE recommendations (
  id              BIGSERIAL PRIMARY KEY,
  profile_id      UUID NOT NULL,
  media_id        BIGINT NOT NULL,
  score           REAL NOT NULL,
  reason          VARCHAR(64) NOT NULL,  -- ''tmdb_similar'' | ''history'' | ''hybrid''
  payload         JSONB,
  generated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at      TIMESTAMPTZ NOT NULL,
  UNIQUE(profile_id, media_id)
);
CREATE INDEX idx_recommendations_profile ON recommendations(profile_id, expires_at);
```

#### B2.2 落库策略

- 每天 03:00 cron job 刷新
- 每次刷新覆盖 80% 内容(滚动窗口),减少冷启动
- Feed 拉推荐先查表,miss 才回源 TMDB

#### B2.3 验收

- Feed 拉推荐 P95 < 50ms(命中表)
- TMDB 调用量下降 ≥ 60%

### B3 · 转码进度 & Loading

#### B3.1 接口

- 复用 A4 的 `GET /api/v1/transcode/:task_id` 轮询
- 客户端按 `progress` 展示:
  - `0~30%` → "正在分析视频流…"
  - `30~80%` → "正在转码 4K 流…"
  - `80~100%` → "即将就绪…"

#### B3.2 客户端

- Web Player:进度条 + 文案
- Android TV:圆环 + 文案
- 进度刷新频率 1Hz

#### B3.3 验收

- 转码 4K 视频时,用户看到分段提示,不再白屏
- progress 不会回退(单调递增)

### B4 · ffprobe 补全字段

#### B4.1 表改动

- `media_files` 表新增列:`file_size BIGINT` / `duration_sec INT` / `bitrate INT` / `video_codec VARCHAR(16)` / `video_profile VARCHAR(16)` / `audio_codec VARCHAR(16)` / `audio_channels smallint` / `width INT` / `height INT` / `hdr VARCHAR(8)`
- 迁移:在 v0.4.0 启动时执行 `ALTER TABLE` + backfill(已有文件走 ffprobe)

#### B4.2 触发

- scanner.scanPath 时同步触发 ffprobe
- 失败时:`video_codec = ''unknown''`,不阻塞入库

#### B4.3 验收

- 90% 媒资有完整 probe 数据
- A3 probe API 直接走表,无 ffprobe 调用

### B5 · Admin type-check 进 CI

#### B5.1 CI 改造

- `.github/workflows/ci.yml` 新增 job:`typecheck`
- 步骤:
  - Web Player:`pnpm tsc --noEmit`
  - CMS Admin:`pnpm tsc --noEmit`
  - Android TV:`./gradlew :app:compileDebugKotlin`

#### B5.2 失败处理

- typecheck job 失败 → 阻塞 merge
- cache:`node_modules` / `~/.gradle/caches` 复用

#### B5.3 验收

- main 分支 type-check 必绿
- 5 分钟内跑完三端 typecheck

---

## 6. 技术方案

### 6.1 架构概览

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│  qBittorrent│ ──────► │  downloader  │ ──────► │   scanner   │
│  (WebSocket)│         │  -watcher    │         │  -scanPath  │
└─────────────┘         └──────┬───────┘         └──────┬──────┘
                               │ enqueue                │ enqueue
                               ▼                        ▼
                        ┌──────────────┐         ┌─────────────┐
                        │  scrape_jobs │ ◄────── │  scraper    │
                        │  (Redis)     │         │  (TMDB)     │
                        └──────┬───────┘         └─────────────┘
                               │ done
                               ▼
                        ┌──────────────┐         ┌─────────────┐
                        │  media_files │ ◄────── │  ffprobe    │
                        │  (probe cols)│         │  (subproc)  │
                        └──────┬───────┘         └─────────────┘
                               │ HLS stream (copy/transcode)
                               ▼
                        ┌──────────────┐
                        │  Web Player  │
                        │  Android TV  │
                        └──────────────┘
```

### 6.2 关键决策

| 主题              | 决策                                                                       | 原因                                          |
| ----------------- | -------------------------------------------------------------------------- | --------------------------------------------- |
| 任务队列          | 复用现有 Asynq(Redis backed),加 `scrape_jobs` 专属队列                     | 已有依赖,无需新组件                            |
| 进度推送          | 客户端轮询(1Hz),不引入 WebSocket                                          | 简单可控,断线恢复容易                          |
| Cache 失效        | 显式 `DEL` + 短 TTL(5s)双保险                                             | 避免假失效,5s 内自动兜底                      |
| FFmpeg 调用       | 后端 `ffprobe` 子进程,结果入库                                             | 不在请求路径上跑,性能稳定                      |
| 4K 决策           | 客户端按 codec 能力匹配,后端只暴露 endpoint                                | 决策可演进,后端无状态                          |
| 跨 Profile 隔离   | `feed:profile:<profile_id>` 前缀,Profile 切换只 DEL 自己 + 新 Profile       | 隔离清晰,不会全量失效                          |

### 6.3 依赖

- 新增:`ffprobe`(随 FFmpeg 自带,镜像内置)
- 新增:`@vueuse/core` 用于 Web Player 轮询
- 不引入:WebSocket 服务、消息队列、CDN

---

## 7. 数据模型 / API 变更

### 7.1 表改动(累计)

| 表                  | 改动                                                                                         |
| ------------------- | -------------------------------------------------------------------------------------------- |
| `scrape_jobs`       | + `last_error_code` / `last_error_message` / `attempt_count` / `next_retry_at` / `worker_id` |
| `media_files`       | + 9 列(file_size / duration_sec / bitrate / video_codec / video_profile / audio_codec / audio_channels / width / height / hdr) |
| `recommendations`   | 新表                                                                                         |
| `layouts`           | + `version` 列(乐观锁)                                                                      |

### 7.2 API 增量(12 个)

| 类别       | 端点                                                  | 方法 | 说明                          |
| ---------- | ----------------------------------------------------- | ---- | ----------------------------- |
| 刮削       | `/api/v1/scrape/jobs`                                 | GET  | 分页查询                      |
| 刮削       | `/api/v1/scrape/jobs/:id/retry`                       | POST | 单条重试                      |
| 刮削       | `/api/v1/scrape/jobs/batch-retry`                     | POST | 批量重试                      |
| 刮削       | `/api/v1/scrape/jobs/retry-all-failed`                | POST | 全量重试                      |
| 刮削       | `/api/v1/scrape/stats`                                | GET  | 看板统计                      |
| 转码       | `/api/v1/transcode/:task_id`                          | GET  | 含 progress                   |
| 转码       | `/api/v1/transcode/:task_id/cancel`                   | POST | 取消任务                      |
| 媒资       | `/api/v1/media/:id/probe`                             | GET  | 读取 probe 元数据             |
| 媒资       | `/api/v1/media/:id/seasons`                           | GET  | 季列表                        |
| 媒资       | `/api/v1/media/:id/seasons/:n/episodes`               | GET  | 集列表                        |
| 媒资       | `/api/v1/episode/:id`                                 | GET  | 单集详情                      |
| 缓存       | `/api/v1/cache/invalidate`                            | POST | 主动失效                      |

### 7.3 OpenAPI 同步

- `services/api/docs/openapi.yaml` 必须同步更新
- CI 跑 `redocly lint` 校验规范

---

## 8. UI / 交互

### 8.1 刮削中心(CMS Admin)

```
┌──────────────────────────────────────────────────────────────┐
│  刮削中心                                       [刷新] [批量重试]│
├──────────────────────────────────────────────────────────────┤
│  ●Pending(128)  ○Running(3)  ●Failed(12)  ●Done 24h: 1567    │
├──────────────────────────────────────────────────────────────┤
│  筛选: [全部▼]  来源: [TMDB▼]  关键字: [_____]               │
├──────────────────────────────────────────────────────────────┤
│ ☐ │ ID    │ 标题                │ 来源   │ 尝试 │ 错误   │ 时间│
│ ☐ │ 1001 │ 蝙蝠侠:黑暗骑士     │ TMDB   │ 1    │ -      │ ... │
│ ☐ │ 1002 │ 低智商犯罪 S03E08   │ TMDB   │ 3    │ 429    │ ... │
│ ...                                                          │
└──────────────────────────────────────────────────────────────┘
```

### 8.2 详情页(Web Player 剧集)

```
┌──────────────────────────────────────────────────────────────┐
│ [海报]  蝙蝠侠:黑暗骑士                                       │
│         2008·152 分钟·动作/犯罪                              │
│         简介:xxxxxxxxxxxx...                                  │
│         [▶ 立即播放]  [+ 我的片单]                            │
├──────────────────────────────────────────────────────────────┤
│ 季: [S01] [S02] [S03]                                         │
├──────────────────────────────────────────────────────────────┤
│ 集:                                                          │
│   01  Batman Begins                    140 分钟   [缩略图]    │
│   02  The Dark Knight                  152 分钟   [缩略图]    │
│   ...                                                         │
└──────────────────────────────────────────────────────────────┘
```

### 8.3 转码进度(Web Player/Android TV)

```
┌────────────────────────────────────┐
│         正在准备 4K 流…             │
│                                    │
│  ●●●●●●●●●●○○○○○○○○  60%           │
│  正在转码 4K 流(已用 02:13)        │
└────────────────────────────────────┘
```

### 8.4 布局保存 Toast(CMS Admin)

```
✓ 布局已发布
  Web Player / Android TV 将在 5s 内刷新
```

---

## 9. 验收标准

### 9.1 功能验收(GA Gate)

| # | 项                                                                 | 通过条件                                              |
| - | ------------------------------------------------------------------ | ----------------------------------------------------- |
| F1 | 刮削中心可看、可重试                                               | 1000 个任务下分页 < 200ms,重试幂等                     |
| F2 | 下载 8 集全部 100% 后 ≤60s 全部入库                                | 手动测试连续 3 次全通过                                |
| F3 | 4K HEVC 优先直连,失败流复制,再失败转码                            | 内网 4K 命中率 ≥ 80%,首帧 ≤ 10s                       |
| F4 | API 重启后转码任务不丢                                             | `docker restart api` 后客户端可继续轮询                |
| F5 | 剧集详情可点选单集                                                 | 深链 `/player/<m>?episode=<e>` 直跳并自动播放          |
| F6 | CMS 发布后 5s 内全端刷新                                           | 手动测试 3 次,3 次通过                                |
| F7 | type-check 进 CI                                                   | main 必绿,3 端 typecheck < 5min                       |
| F8 | 猜你喜欢「加入媒体库」可走完                                       | 入队后能在刮削中心看到                                  |
| F9 | recommendations 落库,TMDB 调用下降                                | 灰度对比 ≥ 60%                                         |
| F10 | 转码进度可见                                                       | 4K 视频转码时,前端分段提示                              |
| F11 | ffprobe 字段入库                                                   | 90% 媒资有完整数据                                     |

### 9.2 非功能验收

| 维度      | 指标                                                                              |
| --------- | --------------------------------------------------------------------------------- |
| 性能      | API P95 < 200ms;Feed 拉取 P95 < 50ms;转码首帧 ≤ 10s                              |
| 可靠性    | 任务持久化通过 `docker restart` 测试;FFmpeg 失败不阻塞入库                       |
| 可观测    | scrape_jobs / transcode_tasks / media_files 三表有 Prometheus exporter            |
| 安全      | CMS 看板/转码接口走 JWT;`/api/v1/scrape/*` 需要 admin 角色                      |
| 兼容      | 老版本 `media_files` 表可平滑迁移(ALTER TABLE + backfill)                         |
| i18n      | CMS 错误码用 `code` 字段,文案 i18n 走 `zh-CN` / `en-US`                            |

### 9.3 回归验收

- v0.3.0 全部功能(猜你喜欢、Feed 缓存、HLS 480p)不退化
- CI 全部绿
- `make api-deps && make api-dev` / `make admin-deps && make admin-dev` / `make web-deps && make web-dev` 一键起

---

## 10. 度量指标

| 指标                                | 目标       | 采集方式                              |
| ----------------------------------- | ---------- | ------------------------------------- |
| `scrape_jobs.done / total`          | > 95%      | `scrape_stats` Prometheus exporter     |
| `downloader.done → media.exist` P95 | ≤ 60s      | 时间戳差值,落 `downloader_latency` 表  |
| 内网 4K HEVC 直连/流复制命中率      | ≥ 80%      | `player_decision` 埋点                |
| 转码任务持久化恢复成功率             | 100%       | `docker restart` 自动化测试           |
| Feed 失效到呈现 P95                  | ≤ 5s       | `feed_freshness` 时间戳                |
| typecheck CI 成功率                 | 100%       | GitHub Actions 统计                   |
| recommendations 落库命中率          | ≥ 90%      | `recommendations.hit / total`         |

---

## 11. 风险与缓解

| 风险                                         | 影响                          | 缓解                                                                   |
| -------------------------------------------- | ----------------------------- | ---------------------------------------------------------------------- |
| qBit WebSocket 不稳定,事件丢失                | 下载完成不入库                 | 5 分钟兜底轮询 `torrents?filter=completed`                              |
| TMDB 限流 429                                 | 刮削 pending 堆积              | 指数退避 + 看板告警;允许手动降级到 Bangumi/手动刮削                   |
| 4K 客户端 codec 不识别                        | 命中率低                      | 决策逻辑做 codec 白名单,自动降级;埋点上报命中率                          |
| `docker restart` 时转码任务中途,文件被锁       | 任务状态错乱                  | 启动时 rehydrate 后,所有 `transcoding` 任务置 `failed` 并 alert         |
| ffprobe 慢(>2s)                              | 阻塞入库                      | 入库异步化,probe 失败置 `unknown`,后续重试                              |
| `media_files` backfill 时长                   | 升级窗口长                    | 分批处理(每批 100),利用夜间低峰                                        |
| CMS 改布局后误清全 Profile                    | 全部冷启动                    | 失效 key 必须含 `profile_id` 前缀校验,集成测试覆盖                       |
| 跨设备续播被误纳入 v0.4                       | 范围蔓延                      | 显式 N5;放到 v0.5 P1                                                    |

---

## 12. Sprint 拆分(2 周 × 3 迭代)

> 与 `PRD-ROADMAP.md` §6 的 S1/S2/S3 节奏对齐。

### S1 — 刮削中心 + 下载再入库(第 1-2 周)

| 任务                              | 产出                                             |
| --------------------------------- | ------------------------------------------------ |
| A1 刮削中心(看板 + 队列)          | CMS 路由 `/admin/scrape` + 5 个 API              |
| A2 下载完成再入库                 | downloader-watcher 改造 + scanner 联动           |
| B4 ffprobe 补全字段              | `media_files` 9 列 + backfill                    |
| 验收:F1 / F2 / F11               |                                                  |

### S2 — 4K 优先 + 任务持久化(第 3-4 周)

| 任务                              | 产出                                             |
| --------------------------------- | ------------------------------------------------ |
| A3 4K/直连优先播放              | probe API + 决策逻辑 + 三端 fallback             |
| A4 HLS 任务持久化                 | Redis rehydrate + 状态机 + 客户端轮询            |
| B3 转码进度 & Loading            | 前端进度条 + 客户端轮询                           |
| 验收:F3 / F4 / F10               |                                                  |

### S3 — 一致性 + 选集 + 落库(第 5-6 周)

| 任务                              | 产出                                             |
| --------------------------------- | ------------------------------------------------ |
| A5 Web 剧集选集                    | 季/集 API + 详情页                                |
| A6 Feed/布局一致                | 显式失效 + EventSource/轮询版本号               |
| B1 猜你喜欢入库闭环              | `library_inbox` 服务 + 向导                       |
| B2 recommendations 落库           | cron job + 表 + 命中统计                          |
| B5 Admin type-check 进 CI         | `.github/workflows/typecheck.yml`                |
| 验收:F5 / F6 / F7 / F8 / F9      |                                                  |

### Release(第 7 周)

- 全量回归 + CHANGELOG + tag `v0.4.0`
- 升级指南见 §13
- 度量看板上线(见 §10)

---

## 13. 兼容性 / 升级

### 13.1 数据库迁移

- `scrape_jobs` 加列:无锁 ALTER,即时生效
- `media_files` 加列:同上加 backfill
- `recommendations` 新建:无侵入
- `layouts` 加 `version` 列:默认 1,optimistic lock

### 13.2 API 兼容

- 老客户端:`/api/v1/media/:id` 响应不变(probe 数据由前端按需拉)
- 升级窗口:旧 CMS 与新 API 共存,旧 CMS 看不到新看板但不阻塞使用

### 13.3 镜像升级

- `docker compose pull && docker compose up -d`
- 启动顺序:api → admin → web-player → android-tv
- 回滚:保留 v0.3.0 镜像 tag

### 13.4 客户端降级

- Web Player/Android TV 未升级时,使用旧版 `media/:id` 仍可工作
- 布局一致性的 EventSource 是可选信令,无则走轮询

---

## 14. 文档交付

| 文档                                       | 类型       | 负责人 |
| ------------------------------------------ | ---------- | ------ |
| `docs/PRD-V0.4.0.md`(本文件)              | PRD        | 产品   |
| `docs/openapi.yaml` 增量                    | API 契约   | 后端   |
| `docs/RUNBOOK-scraping-center.md`         | Runbook    | 后端   |
| `docs/RUNBOOK-4k-playback.md`             | Runbook    | 后端   |
| `docs/CHANGELOG.md` 更新                    | Changelog  | 全员   |
| `docs/UPGRADE-v0.3-to-v0.4.md`             | 升级指南   | 运维   |
| `CHANGELOG.md` 链接本文件                   | 索引       | 产品   |

---

## 15. 变更记录

| 版本 | 日期       | 说明                                                  |
| ---- | ---------- | ----------------------------------------------------- |
| v1.0 | 2026-06-27 | 初版,基于 `PRD-ROADMAP.md` v1.2 展开 A1-A6 / B1-B5   |

---

*下一版本:v0.5.0(多端产品化)· PRD 见 PRD-ROADMAP.md §5.2。*
