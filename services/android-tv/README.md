# MediaHub Android TV 客户端

> 基于 Android TV Leanback + Jetpack Compose for TV 的原生 TV 应用。

## ✨ 特色

- 🎯 **D-pad 焦点系统**（遥控器原生支持）
- 📺 **Compose for TV UI**（卡片 / 横滑 / Hero / 详情 / 搜索）
- 🎬 **Media3 ExoPlayer**（HLS + 直连 fallback）
- 💾 **续播支持**（跨设备同步进度，秒级精度）
- 👨‍👩‍👧 **多 Profile**（家庭成员切换，独立历史/收藏）
- 🚀 **Coil 图片缓存**（内存 25% + 磁盘 250MB，复用 OkHttp）
- 🛡️ **Release 优化**（R8 + 资源压缩 + ProGuard 规则）

## 🚀 快速开始

### 前置条件

- **Android Studio Hedgehog (2023.1.1)** 或更新
- **JDK 17**
- **Android SDK 34** + Android TV SDK
- 真机（Sony TV / Mi TV / Nvidia Shield）或 Android TV 模拟器

### 打开项目

1. 用 Android Studio 打开 `services/android-tv/` 目录
2. 等待 Gradle Sync 完成
3. 连接 TV 设备或启动 TV 模拟器
4. Run ▶ app

### 配置 API 地址

应用首次启动会要求配置 NAS API 地址：

```
http://10.0.0.1:3000      # 局域网 IP
http://mediahub.local:3000  # mDNS（如果路由器支持）
```

可在「设置」页随时修改。

## 📐 项目结构

```
app/src/main/
├── AndroidManifest.xml                   # Leanback + 4 个 Activity + MediaSession
├── java/com/mediahub/tv/
│   ├── MediaHubApp.kt                    # Application + Coil ImageLoader
│   ├── ui/
│   │   ├── MainActivity.kt                # 启动入口 + 路由
│   │   ├── theme/Theme.kt                 # 深色主题
│   │   ├── setup/SetupActivity.kt         # 首次配置（API URL）
│   │   ├── browse/
│   │   │   ├── BrowseScreen.kt           # 首页（拉 Feed + 加载/错误/空状态）
│   │   │   └── RowCarousel.kt             # 横滑 + 卡片（稳定 key）
│   │   ├── detail/DetailActivity.kt      # 详情页（背景/海报/元数据/续播/操作）
│   │   ├── search/SearchActivity.kt      # 搜索（debounce + grid）
│   │   ├── settings/SettingsActivity.kt  # 设置（API / Profile / 播放）
│   │   └── playback/PlaybackActivity.kt  # ExoPlayer + 进度上报
│   ├── data/
│   │   ├── api/MediaHubApi.kt            # OkHttp + 5 个接口
│   │   ├── model/Models.kt                # Feed / MediaDetail / Profile
│   │   └── PreferencesRepository.kt       # DataStore
│   └── playback/
│       └── MediaSessionService.kt         # 系统媒体控件
└── res/
    ├── values/themes.xml                  # Leanback 主题
    ├── values/colors.xml
    ├── values/strings.xml
    ├── drawable/tv_banner.xml             # 占位 logo（请替换）
    └── mipmap-anydpi-v26/ic_launcher.xml
```

## 🎮 遥控器操作

| 按键 | 操作 |
|---|---|
| 方向键 | 焦点移动 |
| OK / Enter | 选中当前焦点 |
| 返回 | 上级 / 退出 |
| 播放 / 暂停 | 控制播放 |
| 快进 / 快退 | 30 秒跳跃 |
| 长按 OK | 显示卡片操作菜单 |

## 🔧 调试

### 启用 ADB 调试

```bash
adb connect <tv-ip>:5555
adb shell pm grant com.mediahub.tv android.permission.POST_NOTIFICATIONS
```

### 查看日志

```bash
adb logcat -s MediaHubApp:V ExoPlayer:V Coil:V
```

## 🏗️ CI/CD (Codemagic)

`codemagic.yaml` 已配置：

- **android-tv-debug**：每次 push 自动构建 debug APK + AAB + lint 报告
- **android-tv-release**：tag 触发，输出签名 APK / AAB / mapping.txt

### 启用步骤

1. 把仓库推到 GitHub
2. 登录 https://codemagic.io 连接仓库
3. 选 "Import from codemagic.yaml"
4. 在 Settings 配置 release keystore
5. 之后每次 push / tag 自动触发

## 🛣️ 路线图

- [x] 项目骨架 + Gradle 配置（Compose for TV）
- [x] Browse 首页（横滑 + 卡片 + 加载/错误状态）
- [x] SetupActivity（首次配置 API URL）
- [x] SettingsActivity（API / Profile / 播放 / 关于）
- [x] DetailActivity（详情 + 续播 + 操作）
- [x] SearchActivity（搜索 + debounce + 网格）
- [x] PlaybackActivity + MediaSession + ExoPlayer
- [x] Coil 图片缓存（性能优化）
- [x] Codemagic CI（自动构建 APK）
- [ ] 视频详情推荐（等待后端 /similar 接口）
- [ ] 字幕选择 UI
- [ ] 收藏功能
- [ ] 儿童模式快捷切换

## 📝 已知限制

1. **Play Store 上架**：TV 应用需要单独适配和审核
2. **图标资源**：当前是占位 vector，正式发布前需要替换为 320x180 TV banner
3. **API 鉴权**：当前 TV 客户端走匿名（profile_id=tv-anonymous），未来加 TV 设备 token

## 📜 许可证

Apache License 2.0