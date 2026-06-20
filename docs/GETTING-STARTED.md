# MediaHub 启动验证指南

> **目标**：在群晖 DS920+ 上从零跑通 MediaHub 全栈，验证 W1-W9 所有核心功能。
>
> **预计时间**：1.5 ~ 2 小时（含首次构建 + 下载时间）
>
> **验证完会得到**：
> - ✅ 6 个 Docker 容器稳定运行
> - ✅ 1 个示例媒资（带完整元数据 + 缩略图）
> - ✅ 1 个已发布布局（Web Player / Android TV 都能拉到）
> - ✅ 续播 / 搜索 / Profile / 下载 / 字幕 / 儿童模式全部可用
> - ✅ 8+ 张截图（README 用）

---

## 📋 0. 前置检查清单

### 0.1 硬件 & 网络

| 项 | 要求 | 检查命令 |
|---|---|---|
| **NAS** | 群晖 DS920+（或同等 x86 NAS） | `ssh admin@nas` 能登录 |
| **CPU** | Intel J4125 或更新（带 Quick Sync） | DSM → 控制面板 → 系统 |
| **内存** | ≥ 4GB（推荐 8GB） | DSM → 控制面板 → 系统 |
| **硬盘** | ≥ 20GB 可用（数据卷） | `df -h /volume1` |
| **网络** | NAS 和电脑在同一局域网 | `ping nas.local` |
| **公网出口** | 可选（仅远程访问需要） | `curl ifconfig.me` |

### 0.2 Docker & SSH

SSH 进 NAS 后确认：

```bash
# 1. Docker 版本 ≥ 20.10
docker --version
# 期望：Docker version 20.10.x 或更高

# 2. Docker Compose ≥ v2
docker compose version
# 期望：Docker Compose version v2.x.x

# 3. /dev/dri 设备存在（Intel GPU 直通）
ls -la /dev/dri
# 期望：看到 card0 和 renderD128

# 4. 磁盘空间
df -h /volume1
# 期望：可用 ≥ 20GB

# 5. 已创建以下目录（首次手动创建）
mkdir -p /volume1/docker/mediahub/{postgres,redis,qbittorrent/config}
mkdir -p /volume1/media
mkdir -p /volume1/downloads
ls -la /volume1/docker/mediahub/
```

### 0.3 必备账号

| 账号 | 用途 | 获取地址 |
|---|---|---|
| **TMDB API Key** | 自动刮削海报/简介 | https://www.themoviedb.org/settings/api （免费注册申请） |
| **NAS SSH 账号** | 部署和调试 | DSM → 控制面板 → 终端机和 SNMP → 启用 SSH |

---

## 🚀 1. 部署项目到 NAS

### 1.1 方式 A：scp 上传（推荐，先在本地确认 build 通过）

**Windows PowerShell：**

```powershell
# 1. 在本地先做完整构建（避免在 NAS 上等 30 分钟）
cd D:\project\mediahub
docker compose -f docker-compose.yml build

# 2. 打包项目（排除 node_modules / .git / dist）
Compress-Archive -Path "D:\project\mediahub\*" `
  -DestinationPath "D:\project\mediahub.zip" `
  -CompressionLevel Optimal `
  -Force

# 3. scp 上传（需要 NAS 启用 SSH）
scp D:\project\mediahub.zip admin@nas.local:/volume1/docker/
```

**NAS 上：**

```bash
# 1. 解压
cd /volume1/docker
unzip mediahub.zip -d mediahub
cd mediahub

# 2. 复制环境变量模板
cp .env.example .env
```

### 1.2 方式 B：git clone（如果你推到 GitHub 后）

```bash
ssh admin@nas.local
cd /volume1/docker
git clone https://github.com/<your-user>/mediahub.git
cd mediahub
cp .env.example .env
```

---

## 🔐 2. 配置 `.env`

**在 NAS 上编辑：**

```bash
nano /volume1/docker/mediahub/.env
# 或用 vim / DSM File Station
```

### 2.1 必须改的项

```bash
# ---------- 数据库（生成强密码）----------
POSTGRES_PASSWORD=<随机 32 位字符串>
REDIS_PASSWORD=<随机 32 位字符串>

# ---------- API JWT 密钥（必须 ≥ 32 字符）----------
API_JWT_SECRET=<随机 64 位字符串>

# ---------- TMDB API Key ----------
TMDB_API_KEY=<你的 TMDB v3 API key>

# ---------- qBittorrent WebUI 密码 ----------
QBIT_PASSWORD=<强密码>

# ---------- NAS 实际路径（按需调整）----------
NAS_MEDIA_HOST=/volume1/media
NAS_DOWNLOADS_HOST=/volume1/downloads
```

### 2.2 生成强密码

```bash
# 一次性生成 4 个强密码
echo "POSTGRES_PASSWORD=$(openssl rand -hex 16)"
echo "REDIS_PASSWORD=$(openssl rand -hex 16)"
echo "API_JWT_SECRET=$(openssl rand -hex 32)"
echo "QBIT_PASSWORD=$(openssl rand -base64 18 | tr -d '=+/')"
```

### 2.3 完整 .env 示例

```bash
# ====== 数据库 ======
POSTGRES_USER=mediahub
POSTGRES_PASSWORD=a1b2c3d4e5f6...  # ← 实际填入生成的密码
POSTGRES_DB=mediahub
POSTGRES_PORT=5432

# ====== Redis ======
REDIS_PASSWORD=f6e5d4c3b2a1...  # ← 实际填入
REDIS_PORT=6379

# ====== API ======
API_PORT=3000
API_MODE=debug
API_JWT_SECRET=...  # ← 64 字符
API_LOG_LEVEL=info

# ====== 前端端口 ======
ADMIN_PORT=8080
WEB_PORT=8081

# ====== TMDB ======
TMDB_API_KEY=eyJhbGciOiJIUzI1NiJ9...  # ← 你的 key
TMDB_LANGUAGE=zh-CN
TMDB_BASE_URL=https://api.themoviedb.org/3

# ====== 媒资路径 ======
MEDIA_ROOT=/media
DOWNLOAD_ROOT=/downloads
NAS_MEDIA_HOST=/volume1/media
NAS_DOWNLOADS_HOST=/volume1/downloads

# ====== qBittorrent ======
QBIT_HOST=qbittorrent
QBIT_PORT=8080
QBIT_USER=admin
QBIT_PASSWORD=...  # ← 填入

# ====== 转码 ======
TRANSCODE_ENABLED=true
TRANSCODE_HW_ACCEL=qsv
TRANSCODE_MAX_BITRATE=8000k

# ====== 刮削 ======
SCRAPER_WORKER_COUNT=3
SCRAPER_RETRY_TIMES=3
```

---

## 🐳 3. 启动服务

```bash
cd /volume1/docker/mediahub

# 首次启动（自动构建镜像，5-10 分钟）
docker compose up -d

# 观察启动过程
docker compose logs -f
```

**期望日志**：

```
✔ Network mediahub_mediahub-net  Created
✔ Container mediahub-postgres     Healthy
✔ Container mediahub-redis        Healthy
✔ Container mediahub-qbittorrent  Started
✔ Container mediahub-api          Started
✔ Container mediahub-admin        Started
✔ Container mediahub-web-player   Started
```

按 `Ctrl+C` 退出日志跟踪（容器仍在后台运行）。

---

## ✅ 4. 健康检查

### 4.1 基础健康

```bash
# API 存活检查
curl -s http://localhost:3000/health | jq
# 期望：{"status":"ok","service":"mediahub-api",...}

# K8s 风格别名
curl -s http://localhost:3000/healthz | jq

# 就绪检查（含 DB + Redis）
curl -s http://localhost:3000/health/ready | jq
# 期望：{"status":"ready","checks":{"database":"ok","redis":"ok"},...}

# 性能指标
curl -s http://localhost:3000/metrics | jq
# 期望：包含 goroutines / heap_alloc_mb / database 字段
```

### 4.2 容器状态

```bash
docker compose ps
```

**期望输出**：6 个容器状态都是 `running` 或 `healthy`：

```
NAME                    STATUS              PORTS
mediahub-postgres       Up (healthy)        0.0.0.0:5432->5432/tcp
mediahub-redis          Up (healthy)        0.0.0.0:6379->6379/tcp
mediahub-qbittorrent    Up                  0.0.0.0:8082->8080/tcp
mediahub-api            Up (healthy)        0.0.0.0:3000→3000/tcp
mediahub-admin          Up                  0.0.0.0:8080->80/tcp
mediahub-web-player     Up                  0.0.0.0:8081->80/tcp
```

### 4.3 端口验证

在浏览器依次访问：

| URL | 期望 |
|---|---|
| http://nas.local:3000/health | JSON: `status: ok` |
| http://nas.local:8080 | CMS Admin 登录页 |
| http://nas.local:8081 | Web Player 首页 |
| http://nas.local:8082 | qBittorrent WebUI 登录页 |

> 💡 如果浏览器访问不到：用电脑的浏览器访问 `http://<NAS 内网 IP>:端口`，NAS IP 在 DSM → 控制面板 → 网络 → 网络接口 查看

---

## 🔑 5. 初始化默认管理员

### 5.1 登录 CMS Admin

1. 浏览器打开 **http://nas.local:8080**
2. 默认账号：`admin` / `admin123`（首次登录会提示修改密码）
3. **立即改密码**（右上角 → 用户设置）

> ⚠️ **生产前必须改 admin 密码！**

### 5.2 修改默认密码

```bash
# API 方式（如果忘记管理员密码想重置）
docker exec -it mediahub-api sh
# 容器里：
# 暂时没做 CLI 重置命令，可通过 SQL：
# psql ... -c "UPDATE users SET password_hash = crypt('new_pass', gen_salt('bf')) WHERE username = 'admin';"
```

### 5.3 检查 TMDB 配置

```bash
# API 健康检查里验证 TMDB key 有效
docker exec mediahub-api sh -c 'curl -s "https://api.themoviedb.org/3/configuration?api_key=$TMDB_API_KEY" | head -50'
```

**期望**：返回 JSON，包含 `images.base_url` 字段

---

## 🎬 6. 添加第一个媒资

### 6.1 准备一个测试视频文件

把任意一个 mp4 / mkv 文件上传到 NAS：

```bash
# 通过 scp 上传（PowerShell）
scp "D:\Downloads\test-movie.mp4" admin@nas.local:/volume1/media/movies/
```

或者直接在 DSM File Station 上传。

> 💡 **没有测试视频？** 用 ffmpeg 生成一个 30 秒的彩条视频：
>
> ```bash
> docker exec mediahub-qbittorrent bash -c \
>   "ffmpeg -f lavfi -i testsrc=duration=30:size=640x360:rate=24 \
>    -f lavfi -i sine=frequency=1000:duration=30 \
>    -c:v libx264 -c:a aac /downloads/test-signal.mp4"
> cp /volume1/downloads/test-signal.mp4 /volume1/media/movies/
> ```

### 6.2 触发库扫描

**方式 A：自动扫描**（30 分钟 watcher，会自动发现）

**方式 B：手动触发**：

```bash
# 通过 API 触发全库扫描
curl -X POST http://localhost:3000/api/v1/scanner/scan

# 期望响应：
# {"data":{"total":1,"added":1,"skipped":0,"failed":0}}
```

### 6.3 在 CMS Admin 查看

1. 浏览器打开 **http://nas.local:8080**
2. 左侧菜单 → **媒资库**
3. 应该能看到刚上传的视频（暂未刮削，海报是占位）

### 6.4 触发 TMDB 刮削

在媒资库列表，找到刚入库的视频 → 点击进入详情 → 顶部 **「重新刮削」** 按钮。

或用 API：

```bash
# 列出所有媒资，找到 ID
curl -s http://localhost:3000/api/v1/media?page_size=10 | jq '.data.items[] | {id, title, scrape_status}'

# 触发重新刮削（替换 MEDIA_ID）
curl -X POST http://localhost:3000/api/v1/media/MEDIA_ID/rescan
```

**日志观察**：

```bash
docker compose logs -f api | grep -E "scrape|TMDB"
```

**期望日志**：

```
INFO  scrape worker started   media_id=...
INFO  TMDB search hit  title="..." tmdb_id=12345
INFO  TMDB metadata fetched  poster_url=... backdrop_url=...
INFO  media updated   scrape_status=done
```

### 6.5 验证刮削结果

刷新媒资详情页，海报、背景、简介、年份、类型应该都自动填充。

📸 **截图保存**：`screenshots/01-media-detail-scraped.png`

---

## 📐 7. 创建 + 发布布局

### 7.1 创建布局

1. 左侧菜单 → **布局管理** → **新建布局**
2. 填写：
   - 名称：`测试布局 - 主推荐`
   - 描述：`首屏 Hero + 4 行横滑`
   - 目标平台：**Web + Android TV + iOS**（多选）
3. 进入编辑器

### 7.2 配置行（拖拽 / 卡片操作）

按下面顺序添加 5 行：

| 行序 | 类型 | 数据源 | 标题 | 卡片样式 |
|---|---|---|---|---|
| 1 | Hero Banner | **Hero 推荐** | — | Backdrop |
| 2 | Shelf | **最近添加** | 最近添加 | Poster |
| 3 | Shelf | **类型 - 动作** | 动作片 | Poster |
| 4 | Shelf | **类型 - 科幻** | 科幻片 | Poster |
| 5 | Shelf | **推荐 - 热门** | 热门推荐 | Poster |

> 💡 **没数据？** 至少添加 `Hero 推荐` + `最近添加` 两行就够了，单个媒资也会显示。

### 7.3 启用 + 发布

1. 右上角 → **启用** 按钮（开关变绿）
2. 顶部 → **发布** 按钮
3. 弹窗：选择目标平台（Web / Android TV / iOS）+ 立即生效

📸 **截图保存**：`screenshots/02-layout-editor.png`

---

## 🎥 8. Web Player 验证

### 8.1 打开 Web Player

浏览器 → **http://nas.local:8081**

**期望看到**：

1. 顶部 Hero 区显示刚刮削的电影（带背景大图 + 标题 + 元数据 + 播放/详情按钮）
2. 下方 4-5 行横滑卡片
3. 右上角有 🔍 搜索 + 👤 Profile 切换按钮

📸 **截图保存**：`screenshots/03-webplayer-home.png`

### 8.2 验证 Hero + 横滑

- ✅ Hero 切换动画（Vue Transition）
- ✅ 卡片 hover 缩放
- ✅ 横滑顺畅（拖动 / 滚轮）
- ✅ 评分 ⭐ 显示

### 8.3 验证详情页

- 点击任意卡片 → 跳到详情页
- 海报 + 标题 + 简介 + 元数据齐全
- 顶部背景大图（backdrop）

📸 **截图保存**：`screenshots/04-webplayer-detail.png`

### 8.4 验证搜索

- 点击右上角 🔍 → 输入「电影名」或「星际」
- 300ms debounce 后实时显示结果
- 4 列网格 + 缩略图

📸 **截图保存**：`screenshots/05-webplayer-search.png`

### 8.5 验证播放

- 任意详情页 → 点击 ▶ 播放
- **期望**：
  - 视频开始播放（HLS / 直连）
  - 5 秒后快捷键提示自动消失
  - 键盘按 → / ← 测试 10 秒跳跃
  - 键盘按 F 全屏

📸 **截图保存**：`screenshots/06-webplayer-playing.png`

### 8.6 验证续播

1. 播放 30 秒后按暂停
2. 关闭浏览器
3. 重新打开 → 播放器自动从 30 秒处继续

### 8.7 验证 Toast 错误处理

- 故意拔掉网线 / 关掉 API 容器
- 刷新页面 → 应该看到 `加载失败：xxx` 红色 Toast 提示

---

## 👨‍👩‍👧 9. Profile + 儿童模式验证

### 9.1 添加 Profile

API 方式添加（Admin 后台也有界面，但 API 更快）：

```bash
# 获取 token
TOKEN=$(curl -s -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | jq -r '.data.token')

# 添加「爸爸」
curl -X POST http://localhost:3000/api/v1/profiles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"爸爸","avatar_emoji":"👨","is_kid":false,"pin":""}'

# 添加「小宝」+ PIN 锁
curl -X POST http://localhost:3000/api/v1/profiles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"小宝","avatar_emoji":"🧒","is_kid":true,"pin":"1234"}'
```

### 9.2 Web Player 切换

- 点击右上角 👤 头像
- 弹窗显示两个 Profile
- 切换到「小宝」→ 头像变成 🧒
- **如果设置有 is_adult 媒资，应该被过滤掉**

📸 **截图保存**：`screenshots/07-profile-switch.png`

### 9.3 验证儿童过滤

```sql
-- 手动把刚入库的媒资标记为成人（演示用）
docker exec -it mediahub-postgres psql -U mediahub -d mediahub
> UPDATE media SET is_adult = true WHERE title LIKE '%测试%';
> \q
```

刷新 Web Player → 切换到「小宝」→ 这部媒资应该从 Feed 里消失。

---

## ⬇️ 10. 下载 + 字幕验证

### 10.1 qBittorrent 登录

- 浏览器 → **http://nas.local:8082**
- 用户名：`admin`，密码：`.env` 里的 `QBIT_PASSWORD`

### 10.2 添加下载任务（API 方式）

```bash
# 找一个合法 BT 种子 URL（用 Ubuntu 镜像测试）
curl -X POST http://localhost:3000/api/v1/downloader/add \
  -H "Content-Type: application/json" \
  -d '{
    "url": "magnet:?xt=urn:btih:DD8255ECDC7CA55FB0BBF81323D87062DB1F6D1C",
    "category": "movies",
    "save_path": "/downloads"
  }'
```

> ⚠️ **重要**：用合法种子（Ubuntu 镜像 / Debian 镜像）测试，不要下载盗版内容。

### 10.3 在 CMS Admin 查看下载

- 左侧菜单 → **下载管理**
- 应该看到任务列表，5 秒自动刷新
- 进度条 / 速度 / 状态实时更新

📸 **截图保存**：`screenshots/08-downloads-page.png`

### 10.4 验证自动入库

下载完成后，下载器 watcher 会自动：

1. 移动文件到 `/volume1/media/movies/`
2. 调用 Scanner 入库
3. 入库后调用 Scraper 抓 TMDB

几分钟后在 **媒资库** 应该能看到这个新文件（带元数据）。

### 10.5 字幕下载

> 当前字幕模块是 SubHD 占位实现（HTML 解析未实装），这里演示 API：

```bash
# 给指定媒资搜索字幕
MEDIA_ID=<你的媒资 ID>
curl -X POST http://localhost:3000/api/v1/subtitle/search \
  -H "Content-Type: application/json" \
  -d "{\"media_id\":\"$MEDIA_ID\",\"lang\":\"zh-CN\"}"
```

---

## 📺 11. Android TV 客户端验证

> ⚠️ **需要 Android Studio + JDK 17 + Android SDK 34**

### 11.1 准备开发环境

1. 安装 [Android Studio Hedgehog+](https://developer.android.com/studio)
2. SDK Manager 安装：
   - Android 14 (API 34)
   - Android TV SDK
   - JDK 17

### 11.2 打开项目

```bash
# 在 Windows 上：
# 用 Android Studio 打开 D:\project\mediahub\services\android-tv\
```

等待 Gradle Sync（约 5 分钟首次）。

### 11.3 配置 API URL

**方式 A：首次启动配置**

1. Run ▶ app 到 TV 模拟器
2. 启动后进入 Setup 页
3. 输入：`http://10.0.2.2:3000`（**模拟器访问宿主机**）或 NAS 内网 IP
4. 点击「测试并继续」
5. 验证通过后跳到 Browse

**方式 B：修改默认地址**

编辑 `app/build.gradle.kts`：

```kotlin
buildConfigField("String", "DEFAULT_API_BASE", "\"http://nas.local:3000\"")
```

### 11.4 TV 模拟器操作

- 用方向键（↑↓←→）移动焦点
- 按 OK 选中
- 焦点应在卡片之间平滑移动

📸 **截图保存**：`screenshots/09-androidtv-browse.png`

### 11.5 验证详情页 + 播放

- 焦点在某个电影卡片 → 按 OK
- 跳到 DetailActivity
- 显示背景大图 + 海报 + 元数据 + 操作按钮
- 按 ▶ 播放 → ExoPlayer 启动

📸 **截图保存**：`screenshots/10-androidtv-detail.png` + `11-androidtv-playing.png`

### 11.6 验证设置页

- 主页右上角 ⚙ 图标 → 设置
- 显示 API URL + Profile 列表 + 关于
- 切换 Profile → 主页重新加载

### 11.7 验证续播

1. 播放任意视频 30 秒后退出
2. 再进详情页 → 应该出现「续播」按钮（蓝色边框）

---

## 📊 12. 性能验证

### 12.1 Redis 缓存

```bash
# 查看 Feed 缓存
docker exec -it mediahub-redis redis-cli -a <REDIS_PASSWORD> KEYS "mediahub:feed:*"

# 期望：看到至少一个 key（feed:web:anonymous 等）
# TTL
docker exec -it mediahub-redis redis-cli -a <REDIS_PASSWORD> TTL "mediahub:feed:web:anonymous"
# 期望：300 左右（5 分钟）

# 命中次数统计
docker exec -it mediahub-redis redis-cli -a <REDIS_PASSWORD> INFO stats | grep keyspace
```

### 12.2 Feed 缓存验证

```bash
# 第一次拉取（无缓存，应该稍慢）
time curl -s http://localhost:3000/api/v1/feed/web -o /dev/null
# 期望：~100ms

# 第二次拉取（有缓存，应该很快）
time curl -s http://localhost:3000/api/v1/feed/web -o /dev/null
# 期望：~10ms
```

### 12.3 Metrics 端点

```bash
curl -s http://localhost:3000/metrics | jq
```

**期望输出**（关键字段）：

```json
{
  "runtime": {
    "goroutines": 25,
    "heap_alloc_mb": 12.5,
    "heap_sys_mb": 18.0,
    "gc_runs": 5,
    "go_version": "go1.22.x"
  },
  "database": {
    "OpenConnections": 3,
    "InUse": 1,
    "Idle": 2,
    "WaitCount": 0,
    "WaitDuration": "0s"
  },
  "time": "2026-06-20T..."
}
```

### 12.4 Intel Quick Sync 验证

```bash
# 进 API 容器
docker exec -it mediahub-api sh

# 检查 ffmpeg 能否识别 Intel GPU
ffmpeg -hide_banner -init_hw_device vaapi=intel:/dev/dri/renderD128 -f lavfi -i testsrc=duration=1:size=320x240:rate=1 -vf 'format=nv12,hwupload' -c:v h264_vaapi -f null -
# 期望：输出包含 "VAAPI" 字样，没有 "Cannot open" 错误
```

📸 **截图保存**：`screenshots/12-metrics-output.png`

---

## 🧹 13. 清理（验证完成后）

### 13.1 停止但不删除数据

```bash
docker compose down
```

### 13.2 完全清理（⚠️ 删除所有数据）

```bash
# 用 Makefile 一键清理
make clean

# 或手动
docker compose down -v
rm -rf /volume1/docker/mediahub/postgres
rm -rf /volume1/docker/mediahub/redis
rm -rf /volume1/docker/mediahub/qbittorrent/config
```

---

## 📸 14. 截图清单（README 素材）

| # | 截图 | 何时截 | 文件名 |
|---|---|---|---|
| 1 | 媒资详情（含 TMDB 元数据） | 6.5 节验证后 | `01-media-detail-scraped.png` |
| 2 | 布局编辑器（拖拽中） | 7.2 节配置时 | `02-layout-editor.png` |
| 3 | Web Player 首页 | 8.1 节 | `03-webplayer-home.png` |
| 4 | Web Player 详情页 | 8.3 节 | `04-webplayer-detail.png` |
| 5 | Web Player 搜索 | 8.4 节 | `05-webplayer-search.png` |
| 6 | Web Player 播放 | 8.5 节 | `06-webplayer-playing.png` |
| 7 | Profile 切换 | 9.2 节 | `07-profile-switch.png` |
| 8 | 下载管理 | 10.3 节 | `08-downloads-page.png` |
| 9 | Android TV 首页 | 11.4 节 | `09-androidtv-browse.png` |
| 10 | Android TV 详情 | 11.5 节 | `10-androidtv-detail.png` |
| 11 | Android TV 播放 | 11.5 节 | `11-androidtv-playing.png` |
| 12 | Metrics 输出 | 12.3 节 | `12-metrics-output.png` |

存到：`D:\project\mediahub\screenshots\` （README 引用相对路径）

---

## 🔧 15. 常见问题排查

### Q1: API 容器启动后立刻退出

```bash
docker compose logs api
```

**常见错误**：

- `API_JWT_SECRET too short` → 检查 .env 里 ≥ 32 字符
- `DATABASE_URL invalid` → 检查密码特殊字符（用引号或 URL encode）
- `Failed to connect to postgres` → 容器依赖顺序，等 postgres healthy

### Q2: TMDB 刮削一直 pending

```bash
docker compose logs api | grep -i scrap
# 看 worker 是否启动
```

确认 `.env` 里 `TMDB_API_KEY` 正确，curl 测试：

```bash
docker exec mediahub-api sh -c \
  'curl -s "https://api.themoviedb.org/3/configuration?api_key=$TMDB_API_KEY" | head'
```

返回 JSON 说明 key 正确。

### Q3: Web Player 显示空白

打开浏览器 DevTools（F12）→ Console 看错误：

- `Failed to fetch` → API URL 不对，CORS 错
- `404 on /api/v1/feed/web` → 布局未发布
- `401 Unauthorized` → Token 过期，重新登录

### Q4: 视频播放失败

```bash
docker compose logs api | grep -i hls
```

- `/dev/dri` 不存在 → 检查硬件加速配置
- 文件不存在 → 检查 `MEDIA_ROOT` 路径是否对得上 NAS 路径

### Q5: qBittorrent WebUI 进不去

默认端口 8082，第一次启动需要等 30 秒初始化。

```bash
docker compose logs qbittorrent | tail -20
```

看到 `WebUI listening on ...` 就 OK 了。

### Q6: Android Studio 同步失败

- 检查 `gradle/wrapper/gradle-wrapper.properties` 里的 distribution URL
- 国内环境换成阿里云镜像：
  ```
  distributionUrl=https\://mirrors.aliyun.com/macports/distfiles/gradle/gradle-8.5-bin.zip
  ```

### Q7: Docker build 太慢

国内加速：

```bash
# /etc/docker/daemon.json
{
  "registry-mirrors": ["https://mirror.ccs.tencentyun.com"]
}
sudo systemctl restart docker
```

---

## 🎉 16. 验证完成 checklist

跑完后逐项打勾：

- [ ] 所有 6 个 Docker 容器 healthy
- [ ] `/health` 返回 ok
- [ ] `/metrics` 返回完整指标
- [ ] CMS Admin 登录成功，密码已修改
- [ ] 媒资库至少有 1 个刮削完成的视频
- [ ] 布局编辑器创建并发布了 1 个布局
- [ ] Web Player 首页显示 Hero + 至少 1 行
- [ ] Web Player 搜索可用
- [ ] Web Player 播放视频（HLS 或直连）
- [ ] 续播生效（关闭再开能继续）
- [ ] 至少有 2 个 Profile（含儿童）
- [ ] 切换到儿童 Profile 后 is_adult 媒资消失
- [ ] qBittorrent 下载任务可见
- [ ] 下载完成后自动入库 + 刮削
- [ ] Android TV 模拟器跑通浏览 + 播放
- [ ] 12 张截图齐全

**全部打勾 = Phase 1+2 完整体验通过！** 🚀

---

## 📚 相关文档

- [README.md](../README.md) - 项目总览
- [CHANGELOG.md](../CHANGELOG.md) - 版本变更
- [家庭媒资中心-DS920+定制方案.md](家庭媒资中心-DS920+定制方案.md) - 硬件 + 部署方案
- [Makefile](../Makefile) - 一键命令

> 遇到问题先看 [15. 常见问题](#-15-常见问题排查)，还是没解决可以提 Issue。