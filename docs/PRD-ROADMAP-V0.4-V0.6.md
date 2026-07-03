# MediaHub 产品规划 v0.4 → v0.6

**文档版本**：v1.0
**编写日期**：2026-06-29
**基准版本**：v0.4.0（2026-06-27 已发布）
**目标范围**：v0.4.0 · v0.5.0 · v0.6.0
**关联文档**：[PRD-ROADMAP.md](PRD-ROADMAP.md) · [PRD-V0.4.0.md](PRD-V0.4.0.md) · [OTT-GAPS.md](OTT-GAPS.md) · [MEDIA-DOMAIN.md](MEDIA-DOMAIN.md)

---

## 1. 全局产品定位

**一句话**：家庭 NAS 上的自建媒资中心——从下载、刮削、布局运营到 Web / TV 播放，数据与体验完全自主可控。

**对标体验**：家庭内网的「可配置 Netflix」——CMS 拖拽首页、猜你喜欢、续播、儿童过滤，4K 内网优先原画质。

**核心约束**：
- 硬件：群晖 DS920+（Intel J4125，4 核 2.0GHz）
- 网络：家庭内网为主，不面向公网多租户
- 团队：单人维护，小步迭代

---

## 2. v0.4.0 现状评估（已发布 2026-06-27）

### 2.1 已交付能力

| 领域 | 能力 | 状态 | 关键提交 |
|------|------|------|----------|
| **刮削** | 刮削中心（队列看板、批量重试、失败原因） | ✅ | `8035b86` |
| **刮削** | OpenSubtitles OSHash 识片兜底 | ✅ | `23eed02` |
| **刮削** | TMDB 同系列（Collection）关联 | ✅ | `cc5188a` |
| **刮削** | recommendations 落库（推荐持久化） | ✅ | `8035b86` |
| **播放** | ffprobe 探测 API（media probe） | ✅ | `8035b86` |
| **播放** | Feed 版本轮询（CMS ↔ 播放端一致性） | ✅ | `8035b86` |
| **播放** | HLS 转码进度追踪 | ✅ | `8035b86` |
| **播放** | 库外推荐（TMDB → 入库向导） | ✅ | `8035b86` |
| **扫描** | EP 命名识别 + 剧集选集结构重建 | ✅ | `f70492b` `cbf3b50` |
| **扫描** | .rmvb/.rm 格式支持 | ✅ | `e898789` |
| **扫描** | CMS 可配置自动扫描频率 | ✅ | `cd955fe` |
| **布局** | 榜单、专题、沉浸式模版 | ✅ | `41db5b7` `f2c0542` |
| **布局** | CMS 跳转协议 + 播放端布局驱动首页 | ✅ | `07e5fad` |
| **直播** | IPTV 拉流 + M3U 批量导入 + 分组筛选 | ✅ | `b7f1b2a` ~ `d2654d2` |
| **UI** | Web Player 精选推荐 Hero 轮播 | ✅ | `8f956ee` |
| **UI** | 沉浸式首页榜单视觉升级 | ✅ | `e30dda0` |

### 2.2 v0.4.0 原始目标 vs 实际

| 原始目标 | 实际状态 | 说明 |
|----------|----------|------|
| G1: 下载到入库零等待 | 🚧 部分 | 扫描频率可配，但 qBit watcher 自动触发未完全闭环 |
| G2: 刮削可观测可重试 | ✅ 达成 | 刮削中心 + 批量重试 + 失败原因 |
| G3: 4K 优先播放 | ✅ 达成 | probe API + HLS 流复制 + 决策链路 |
| G4: HLS 任务持久化 | ✅ 达成 | Redis 持久化 + 进度追踪 |
| G5: 剧集选集 | ✅ 达成 | EP 命名识别 + 季/集结构 |
| G6: Feed 实时一致 | ✅ 达成 | 版本轮询机制 |
| G7: CI 拦截回归 | 🚧 部分 | Web/TV type-check 已有，Admin 待确认 |

### 2.3 v0.4.0 遗留项（滚入 v0.5）

| 项 | 优先级 | 说明 |
|----|--------|------|
| qBit watcher 自动触发入库 | P0 | 当前仍依赖手动扫描或定时扫描 |
| Admin type-check 进 CI | P1 | `.github/workflows/ci.yml` 需补充 |
| 跨设备续播同步 | P1 | 当前仅本机 localStorage |
| 海报 CDN / 本地缓存 | P2 | 全量走 TMDB CDN |

---

## 3. v0.5.0 — 多端产品化

### 3.1 版本主题

**从"能用"到"好用"**：补齐 OTT 核心体验（字幕、下一集、分龄过滤），Android TV 达到发布质量，Web Player 2.0 完善播放体验。

### 3.2 发布标准

```
新用户首次打开 → Feed 有内容 → 播放 4K 无卡顿 → 字幕可切 → 看完自动下一集
→ 儿童 Profile 自动过滤 → TV 遥控器全程可用 → APK 可下载安装
```

### 3.3 P0 — 必做

#### A1 · Web 字幕 / 音轨切换

**现状**：SubHD 后端已有，但播放端无 UI 切换入口。

**需求**：
- 播放器控制栏新增「字幕」和「音轨」按钮
- 支持外挂字幕（SRT/ASS/VTT）和内嵌字幕轨
- 字幕列表从 `subtitle_tracks` 表读取（需新建）
- 音轨列表从 ffprobe `audio_streams` 读取
- 选择后实时切换，不中断播放
- Profile 记住上次选择的字幕/音轨偏好

**数据模型**：
```sql
CREATE TABLE subtitle_tracks (
    id          BIGSERIAL PRIMARY KEY,
    file_id     BIGINT REFERENCES media_files(id),
    episode_id  BIGINT REFERENCES episodes(id),
    language    VARCHAR(8) NOT NULL,    -- 'zh', 'en', 'ja'
    format      VARCHAR(8) NOT NULL,    -- 'srt', 'ass', 'vtt', 'embedded'
    path        TEXT,                    -- 外挂字幕文件路径
    source      VARCHAR(16) NOT NULL,   -- 'embedded', 'subhd', 'manual'
    is_default  BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_subtitle_tracks_episode ON subtitle_tracks(episode_id);
CREATE INDEX idx_subtitle_tracks_file ON subtitle_tracks(file_id);
```

**验收**：
- 播放 4K 视频时可切换中英字幕，无卡顿
- 外挂字幕加载 < 2s
- 内嵌字幕轨正确识别（MKV 多轨）

---

#### A2 · 自动下一集

**现状**：OTT-GAPS.md 已规划，episode 数据已有。

**需求**：
- 播放进度 > 90% 或手动结束时，展示「即将播放下一集」倒计时 UI（10s）
- 倒计时内可取消
- 自动跳转 `episode_number + 1`
- 最后一集播放完，展示「返回详情」
- API：`GET /api/v1/media/:id/next-episode?after_episode_id=<id>`

**验收**：
- 连续播放 5 集，每次自动跳转无白屏
- 最后一集不报错，正确展示结束页
- 手动取消倒计时后不自动跳转

---

#### A3 · Android TV 1.0

**现状**：14 个 Kotlin 文件 / ~2300 行，骨架已有（Leanback Feed + ExoPlayer），但功能不完整。

**需求清单**：

| 功能 | 说明 | 优先级 |
|------|------|--------|
| Feed 对齐 Web | `guess-you-like`、榜单、专题行与 Web 一致 | P0 |
| Profile 携带 | 所有请求 `X-Profile-ID` header | P0 |
| 详情页完善 | 季选择器 + 集列表 + 海报 Hero | P0 |
| 播放器增强 | ExoPlayer 硬解优先、4K MKV 直连、字幕切换 | P0 |
| 搜索 | Leanback SearchFragment + 语音输入 | P1 |
| 续播 | 首页「继续观看」行，进度恢复 | P1 |
| 设置页 | API URL、画质偏好、字幕语言 | P1 |
| 儿童模式 | Profile 切换后自动过滤 is_adult | P1 |
| 错误处理 | 网络断开、API 不可达的友好提示 | P0 |
| Release APK | Codemagic CI 出 release 签名 APK | P0 |

**技术要点**：
- ExoPlayer 优先 `MediaItem.DASH` / `MediaItem.HLS`，codec 能力匹配后直连
- Coil 图片加载：内存 LRU + 磁盘缓存 256MB
- Leanback BrowseFragment → Compose TV 混合（逐步迁移）
- Min SDK 21（Android TV 5.0+），Target SDK 34

**验收**：
- 索尼 Android TV 实机测试通过
- Feed 加载 < 2s（内网）
- 4K HEVC 播放流畅（直连或流复制）
- 遥控器全程无死角（无触屏依赖）
- APK < 15MB

---

#### A4 · 媒资 OTT Schema 补全（A 域）

**现状**：OTT-GAPS.md 列出 A 域缺口（影人/演职员/分类/标签/专辑）。

**数据模型**：
```sql
-- 影人
CREATE TABLE persons (
    id          BIGSERIAL PRIMARY KEY,
    tmdb_id     INT UNIQUE,
    name        VARCHAR(255) NOT NULL,
    profile_url TEXT,
    biography   TEXT,
    birthday    DATE,
    known_for   VARCHAR(32),  -- 'acting', 'directing', 'writing'
    created_at  TIMESTAMPTZ DEFAULT now()
);

-- 演职员关联
CREATE TABLE credits (
    id          BIGSERIAL PRIMARY KEY,
    media_id    BIGINT REFERENCES media(id),
    person_id   BIGINT REFERENCES persons(id),
    role_type   VARCHAR(16) NOT NULL,  -- 'cast', 'crew'
    character   VARCHAR(255),
    job         VARCHAR(64),
    "order"     SMALLINT DEFAULT 0,
    UNIQUE(media_id, person_id, role_type, job)
);

-- 专题专辑
CREATE TABLE albums (
    id          BIGSERIAL PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    cover_url   TEXT,
    created_at  TIMESTAMPTZ DEFAULT now()
);

-- 分类（替代现有 genres[] 字符串数组）
CREATE TABLE categories (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(64) NOT NULL UNIQUE,
    tmdb_id     INT
);

-- 标签
CREATE TABLE tags (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(64) NOT NULL UNIQUE
);

-- 关联表
CREATE TABLE media_categories (media_id BIGINT, category_id INT, PRIMARY KEY(media_id, category_id));
CREATE TABLE media_tags (media_id BIGINT, tag_id INT, PRIMARY KEY(media_id, tag_id));
CREATE TABLE album_media (album_id BIGINT, media_id BIGINT, "order" SMALLINT, PRIMARY KEY(album_id, media_id));
```

**刮削联动**：
- TMDB `movie/{id}/credits` → `persons` + `credits`
- TMDB `genre` → `categories`
- TMDB `keywords` → `tags`
- 刮削时自动写入，已有媒资走 backfill 任务

**验收**：
- 新媒资刮削后自动有影人/分类/标签
- CMS 媒资详情可查看演职员表
- 详情页展示导演/主演

---

#### A5 · 内容分级 + 儿童 Profile 策略（E 域）

**现状**：仅 `is_adult` 布尔值 + Profile `is_kid`，粒度不够。

**数据模型**：
```sql
CREATE TABLE content_ratings (
    id          BIGSERIAL PRIMARY KEY,
    media_id    BIGINT REFERENCES media(id),
    country     VARCHAR(4),         -- 'CN', 'US'
    system      VARCHAR(16),        -- 'tmdb', 'mpaa', 'bbfc'
    rating      VARCHAR(16),        -- 'PG-13', 'TV-MA', '15+'
    advisories  TEXT[],             -- ['violence', 'language']
    UNIQUE(media_id, country, system)
);

CREATE TABLE profile_content_policy (
    profile_id          UUID PRIMARY KEY,
    max_rating_level    SMALLINT DEFAULT 18,  -- 数值越大越宽松
    block_adult         BOOLEAN DEFAULT TRUE,
    allowed_category_ids INT[],               -- 可选白名单
    created_at          TIMESTAMPTZ DEFAULT now()
);
```

**Feed / 搜索 / 推荐统一过滤**：
```
filterMedia(media, profile) → bool:
    if profile.block_adult && media.is_adult → false
    if content_ratings.rating > profile.max_rating_level → false
    if profile.allowed_category_ids 非空 && media.category 无交集 → false
    → true
```

**验收**：
- 儿童 Profile 首页无成人内容
- 内容分级从 TMDB `release_dates` 自动刮削
- CMS 可手动覆盖分级

---

### 3.4 P1 — 应做

#### B1 · 跨设备续播同步

**现状**：播放进度存在 localStorage，换设备丢失。

**方案**：
```
表：playback_progress (profile_id, media_id, episode_id?, position_sec, updated_at)
API：
  PUT  /api/v1/progress          body: { media_id, episode_id?, position_sec }
  GET  /api/v1/progress/:media_id  ?episode_id=
```

- 播放器每 10s 上报一次
- 进入播放页先拉服务端进度，> 本地进度则跳转
- Redis 缓存热数据，定期落库

---

#### B2 · CMS 仪表盘

**需求**：
- 刮削队列统计（pending / failed / done 24h）
- HLS 缓存占用（磁盘空间）
- 系统健康（DB / Redis / qBit / TMDB 连通性）
- 媒资库概览（总数、类型分布、最近入库）

---

#### B3 · 收藏 / 观看列表 UI

**现状**：API 已有 `favorites` 表（want / watching / watched / liked），Web 端无 UI。

**需求**：
- 详情页「想看」「在看」「已看」按钮
- 个人中心「我的片单」页面
- TV 端同步

---

#### B4 · 搜索增强（作品 + 影人）

**现状**：`title ILIKE` 模糊搜索。

**需求**：
- 搜索框支持作品名 + 影人名联合搜索
- 搜索结果分组：作品 / 影人
- 影人详情页（作品列表 + 简介）

---

#### B5 · 播放结束推荐

**需求**：
- 播放完最后一集 / 电影结束，展示「猜你喜欢」推荐卡片
- 复用现有推荐 API
- 3 个推荐 + 「返回」按钮

---

### 3.5 P2 — 可选

| 项 | 说明 |
|----|------|
| 海报本地缓存 | TMDB 图片代理 → NAS 本地缓存，减少外网依赖 |
| QSV 降级提示 | CMS 仪表盘提示 QSV 不可用，降级到 libx264 |
| TV 语音搜索 | Android TV SearchFragment + 语音输入 |
| Admin type-check CI | `.github/workflows/ci.yml` 补充 Admin tsc |

---

### 3.6 Sprint 拆分（建议 4 周 × 2 迭代）

#### S1（第 1-2 周）— 播放体验 + OTT Schema

| 任务 | 产出 |
|------|------|
| A1 字幕/音轨切换 | subtitle_tracks 表 + Player UI + API |
| A2 自动下一集 | next-episode API + 播放结束 UI |
| A4 媒资 Schema 补全 | persons/credits/categories/tags 表 + 刮削联动 |
| B4 搜索增强 | 联合搜索 API + 前端分组展示 |

#### S2（第 3-4 周）— TV 产品化 + 分龄

| 任务 | 产出 |
|------|------|
| A3 Android TV 1.0 | Feed 对齐 + 详情页 + 播放器 + Release APK |
| A5 内容分级 | content_ratings 表 + profile_content_policy + Feed 过滤 |
| B1 跨设备续播 | progress API + Web/TV 上报 + 恢复 |
| B2 CMS 仪表盘 | 统计页面 |
| B3 收藏 UI | 我的片单页面 |

#### Release（第 5 周）

- 全量回归 + CHANGELOG + tag `v0.5.0`
- Android TV APK 发布到 GitHub Release

---

## 4. v0.6.0 — 智能与运营

### 4.1 版本主题

**从"好用"到"聪明"**：推荐算法升级、布局运营工具化、数据驱动决策。

### 4.2 发布标准

```
管理员打开 CMS → 仪表盘一目了然 → 拖拽调整布局 → AB 测试自动分流
→ 推荐越来越准 → 用户停留时间增长 → 全链路可观测
```

### 4.3 P0 — 必做

#### C1 · 推荐 2.0（协同过滤）

**现状**：v0.4 的推荐是内容-based（TMDB 相似度 + Jaccard 标签），缺少用户行为驱动。

**方案**：

```
推荐引擎 = 三层融合：

1. 内容相似（已有）
   TMDB similar + genre/tag Jaccard → 基线推荐

2. 协同过滤（新增）
   基于 history 表的用户行为矩阵：
   - item-item CF：看了 A 的人也看了 B
   - SQL 聚合：history JOIN history ON profile_id → 共现矩阵
   - 余弦相似度计算（定期 cron）

3. 热度加权（新增）
   recent_views / total_views → 热度分
   新入库媒资 boost（7 天内 +0.3）

融合公式：
final_score = α × content_score + β × cf_score + γ × popularity_score
α/β/γ 可配（默认 0.4 / 0.4 / 0.2）
```

**数据模型**：
```sql
-- 推荐权重配置
CREATE TABLE recommend_config (
    id              SERIAL PRIMARY KEY,
    content_weight  REAL DEFAULT 0.4,
    cf_weight       REAL DEFAULT 0.4,
    popularity_weight REAL DEFAULT 0.2,
    cf_min_cooccurrence INT DEFAULT 3,   -- 最小共现次数
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- 协同过滤共现矩阵（定期刷新）
CREATE TABLE cf_similarity (
    media_a_id  BIGINT,
    media_b_id  BIGINT,
    score       REAL NOT NULL,
    updated_at  TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY(media_a_id, media_b_id)
);
CREATE INDEX idx_cf_sim_a ON cf_similarity(media_a_id, score DESC);
```

**定时任务**：
```
每天 03:00：
  1. 刷新 cf_similarity（history 共现 → 余弦相似度）
  2. 刷新 recommendations 表（三层融合 → top-N）
  3. 清理过期推荐（> 7 天）
```

**验收**：
- 有 50+ 观看历史的 Profile，推荐准确率（点击率）≥ 30%
- 推荐多样性：top-20 中至少覆盖 5 个不同分类
- 推荐刷新 P95 < 30s（1000 部媒资规模）

---

#### C2 · 布局 AB 测试 UI

**现状**：后端 `layout_publications` 已有 AB 字段，但 CMS 无 UI 配置。

**需求**：
- CMS 布局编辑器支持「创建 AB 变体」
- 每个变体可独立编辑行内容
- 流量分配比例滑块（如 A:70% / B:30%）
- 发布后自动分流（基于 `X-Profile-ID` hash）
- 效果对比：点击率、播放率

**数据模型**：
```sql
-- AB 测试结果统计（匿名）
CREATE TABLE ab_test_events (
    id              BIGSERIAL PRIMARY KEY,
    publication_id  UUID REFERENCES layout_publications(id),
    variant         VARCHAR(1) NOT NULL,    -- 'A', 'B'
    profile_id      UUID,
    event_type      VARCHAR(16) NOT NULL,   -- 'impression', 'click', 'play'
    row_key         VARCHAR(64),            -- 哪一行
    media_id        BIGINT,
    created_at      TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_ab_events_pub ON ab_test_events(publication_id, variant, event_type);
```

**验收**：
- CMS 创建 AB 变体 → 发布 → 两组用户看到不同首页
- 7 天后可查看对比数据
- 不影响 Feed 加载性能

---

#### C3 · 动态规则 UI

**现状**：后端支持时段/星期规则，CMS 无 UI。

**需求**：
- 布局行支持「显示条件」：
  - 时段：如「晚间 20:00-23:00 显示恐怖片行」
  - 星期：如「周末显示合家欢行」
  - Profile：如「儿童 Profile 显示动画行」
- 规则可视化编辑器（时间轴 + 条件卡片）

---

### 4.4 P1 — 应做

#### D1 · Feed 行曝光统计

**需求**：
- 匿名统计每行 Feed 的曝光和点击
- 数据写入 `ab_test_events` 或独立 `feed_analytics` 表
- CMS 仪表盘展示：曝光量、点击率、CTR 趋势图
- 不关联个人身份，仅按行 + 日期聚合

---

#### D2 · 布局模板库

**需求**：
- CMS 内置 5-10 个布局模板（电影之夜、追剧模式、儿童专区、全类型等）
- 一键应用模板
- 模板可导出 / 导入（JSON）

---

#### D3 · 影人详情页

**需求**：
- Web Player 影人详情页：简介 + 参演作品列表
- 从详情页「主演」可跳转
- TV 端影人搜索结果

---

#### D4 · 用户自建片单

**数据模型**：
```sql
CREATE TABLE user_lists (
    id          BIGSERIAL PRIMARY KEY,
    profile_id  UUID NOT NULL,
    name        VARCHAR(128) NOT NULL,
    description TEXT,
    cover_url   TEXT,
    is_public   BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE user_list_items (
    list_id     BIGINT REFERENCES user_lists(id) ON DELETE CASCADE,
    media_id    BIGINT REFERENCES media(id),
    "order"     SMALLINT DEFAULT 0,
    added_at    TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY(list_id, media_id)
);
```

---

### 4.5 P2 — 可选

| 项 | 说明 |
|----|------|
| 全文搜索升级 | Postgres FTS 或 Meilisearch：作品 + 影人 + 标签 |
| 本地化元数据 | TMDB translations → media_localizations 多语言标题/简介 |
| 海报变体 | media_artworks 表（poster / backdrop / logo / still） |
| 作品关系图 | media_relations（续集 / 前传 / 同系列） |

---

### 4.6 Sprint 拆分（建议 4 周 × 2 迭代）

#### S1（第 1-2 周）— 推荐升级 + 运营基础

| 任务 | 产出 |
|------|------|
| C1 推荐 2.0 | cf_similarity 表 + 三层融合 + 定时刷新 |
| D1 Feed 曝光统计 | 匿名事件采集 + CMS 仪表盘图表 |
| C2 AB 测试 UI | CMS 变体编辑器 + 流量分配 |

#### S2（第 3-4 周）— 布局运营 + 影人

| 任务 | 产出 |
|------|------|
| C3 动态规则 UI | 时段/星期/Profile 条件编辑器 |
| D2 布局模板库 | 内置模板 + 导入导出 |
| D3 影人详情页 | Web + TV 影人页面 |
| D4 用户自建片单 | user_lists API + Web UI |

#### Release（第 5 周）

- 全量回归 + CHANGELOG + tag `v0.6.0`

---

## 5. 跨版本技术规划

### 5.1 数据库演进

| 版本 | 迁移 | 说明 |
|------|------|------|
| v0.5 | `000011a` | subtitle_tracks, persons, credits, categories, tags |
| v0.5 | `000011b` | content_ratings, profile_content_policy |
| v0.5 | `000011c` | playback_progress |
| v0.6 | `000012a` | recommend_config, cf_similarity |
| v0.6 | `000012b` | ab_test_events / feed_analytics |
| v0.6 | `000012c` | user_lists, user_list_items |

### 5.2 API 增量汇总

| 版本 | 新增端点数 | 关键端点 |
|------|-----------|----------|
| v0.5 | ~15 | subtitles, audio-tracks, next-episode, progress, search, persons |
| v0.6 | ~12 | recommend/config, ab-tests, dynamic-rules, user-lists, feed-analytics |

### 5.3 性能目标

| 指标 | v0.5 目标 | v0.6 目标 |
|------|-----------|-----------|
| Feed 拉取 P95 | < 50ms | < 50ms |
| 推荐生成耗时 | < 10s (1000 部) | < 30s (含 CF) |
| 搜索响应 | < 100ms | < 100ms |
| 4K 首帧（流复制） | < 3s | < 3s |
| Android TV 启动 | < 2s | < 2s |

---

## 6. 成功指标

| 指标 | v0.5 目标 | v0.6 目标 | 度量方式 |
|------|-----------|-----------|----------|
| 刮削完成率 | > 95% | > 98% | scrape_status=done / total |
| 播放成功率 | > 97% | > 99% | 无 fatal 错误的播放请求 |
| 推荐点击率 | — | ≥ 30% | 推荐卡片点击 / 曝光 |
| 字幕使用率 | ≥ 20% | ≥ 30% | 字幕切换请求 / 播放请求 |
| TV 日活 | — | ≥ 1 次/天 | TV 端 Feed 请求 |
| Feed 一致性 | 100% | 100% | CMS 发布后 5s 内全端刷新 |

---

## 7. 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| TMDB 限流导致刮削堆积 | 影人/分级数据不全 | 指数退避 + 批量 backfill 夜间执行 |
| Android TV 碎片化 | 不同品牌遥控器兼容性 | 聚焦索尼/小米实机，Leanback 标准控件 |
| 协同过滤冷启动 | 新用户无 CF 推荐 | 内容推荐兜底，CF 需 ≥ 10 次观看 |
| AB 测试样本量不足 | 结论不可信 | 仅家庭场景，7 天数据作为参考而非决策依据 |
| subtitle_tracks 数据量 | 大库字幕管理复杂 | 按需下载，不预加载全量 |
| 单人维护 | 迭代速度受限 | 严格 P0 优先，版本小步发 |

---

## 8. 明确不做（v0.4 ~ v0.6）

| 项 | 原因 |
|----|------|
| 公网多租户 SaaS | 定位家庭自建 |
| 内置 tracker / 盗版索引 | 合规 |
| 替代 Jellyfin 全协议（DLNA 等） | 范围控制 |
| 手机原生 App | Web + TV 优先 |
| 完整账号 / OAuth / 2FA | 见 PRD-ROADMAP §10 |
| DRM / 广告 | 与家庭定位冲突 |
| tvOS 客户端 | 推 v1.0 |
| 跳过片头/章节标记 | 推 v1.0 |
| 多版本/导演剪辑 | 推 v1.0 |

---

## 9. 版本总览

```
v0.4.0 ✅  体验闭环    ──  刮削中心 · 4K 播放 · Feed 一致性 · 推荐落库
    │
    ▼
v0.5.0 📋  多端产品化  ──  字幕切换 · 自动下一集 · Android TV 1.0
    │                       内容分级 · 跨设备续播 · 搜索增强
    ▼
v0.6.0 📋  智能与运营  ──  推荐 2.0 · AB 测试 · 动态规则
                            曝光统计 · 模板库 · 影人详情 · 用户片单
    │
    ▼
v1.0.0 📋  生产可用    ──  tvOS · 跳过片头 · 性能专项 · 字幕生态
```

---

## 10. 变更记录

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0 | 2026-06-29 | 初版：v0.4 现状评估 + v0.5/v0.6 完整 PRD |

---

*本文档随版本迭代持续更新。v0.5.0 发布后更新 §2 基线和 §3 状态列。*
