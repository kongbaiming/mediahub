<template>
  <div class="player-page">
    <div class="player-header">
      <button class="back-btn" @click="$router.back()">← 返回</button>
      <h1 v-if="media" class="title">{{ media.title }}</h1>
      <span v-if="resumeInfo && !resumeInfo.completed" class="resume-badge">
        <el-icon><VideoPlay /></el-icon>
        续播 {{ formatTime(resumeInfo.progress) }} / {{ formatTime(resumeInfo.duration) }}
      </span>
    </div>

    <div class="player-container" tabindex="0" @keydown="onKeyDown" ref="containerRef">
      <video
        ref="videoRef"
        class="video"
        controls
        autoplay
        :poster="media?.backdrop_url || media?.poster_url"
        @timeupdate="onTimeUpdate"
        @loadedmetadata="onLoadedMetadata"
        @ended="onEnded"
      ></video>

      <!-- 顶部快捷键提示 -->
      <div v-if="showShortcutTip" class="shortcut-tip">
        <div>空格：播放/暂停</div>
        <div>←→：快退/快进 10 秒</div>
        <div>↑↓：音量</div>
        <div>F：全屏</div>
        <div>M：静音</div>
      </div>
    </div>

    <div v-if="media" class="player-info">
      <h2>{{ media.title }} <span v-if="media.year">({{ media.year }})</span></h2>
      <div class="meta">
        <span v-if="media.rating">⭐ {{ media.rating.toFixed(1) }}</span>
        <span v-if="media.runtime">{{ media.runtime }} 分钟</span>
        <span v-for="g in media.genres" :key="g">{{ g }}</span>
      </div>
      <p v-if="media.overview" class="overview">{{ media.overview }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import Hls from 'hls.js'
import { mediaApi, historyApi, type MediaDetail, type ResumeInfo } from '@/api'

const route = useRoute()
const videoRef = ref<HTMLVideoElement>()
const containerRef = ref<HTMLDivElement>()
const media = ref<MediaDetail | null>(null)
const resumeInfo = ref<ResumeInfo | null>(null)
const hls = ref<Hls | null>(null)
const showShortcutTip = ref(true)

const lastReportAt = ref(0)
const REPORT_INTERVAL = 10

async function loadMedia() {
  try {
    const data = await mediaApi.get(route.params.id as string)
    media.value = data

    try {
      const resume = await historyApi.getResume(data.id)
      if (resume) resumeInfo.value = resume
    } catch {
      // no history
    }

    setupVideo()
  } catch (e: any) {
    console.error('加载媒资失败', e)
    window.toast?.(`加载失败：${e?.message || '未知错误'}`, 'error', 5000)
  }
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function setupVideo() {
  if (!videoRef.value || !media.value) return

  const storagePath = media.value.storage_path
  const ext = storagePath.split('.').pop()?.toLowerCase() || ''
  if (ext === 'mp4' || ext === 'm4v' || ext === 'webm') {
    switchToDirect(storagePath)
    return
  }

  const mediaID = media.value.id
  try {
    await startHLSPlayback(mediaID, storagePath)
  } catch (e: any) {
    console.error('HLS 启动失败', e)
    window.toast?.(`HLS 转码失败：${e?.message || '未知错误'}`, 'error', 5000)
    switchToDirect(storagePath)
  }
}

async function startHLSPlayback(mediaID: string, storagePath: string) {
  const startUrl = `/api/v1/stream/hls?path=${encodeURIComponent(storagePath)}&media_id=${mediaID}`
  const resp = await fetch(startUrl)
  const data = await resp.json()

  const playlistUrl = `/api/v1/stream/hls/${mediaID}/playlist.m3u8`

  if (data.status === 'ready' || data.cached) {
    attachHlsPlaylist(playlistUrl)
    return
  }
  if (data.status === 'failed') {
    throw new Error(data.error || '转码失败')
  }
  if (data.status === 'done') {
    attachHlsPlaylist(data.playlist || playlistUrl)
    return
  }

  await pollHLSReady(mediaID)
  attachHlsPlaylist(playlistUrl)
}

async function pollHLSReady(mediaID: string, maxAttempts = 180) {
  for (let i = 0; i < maxAttempts; i++) {
    await sleep(2000)
    const resp = await fetch(`/api/v1/stream/hls/${mediaID}/status`)
    const st = await resp.json()
    if (st.status === 'done') return
    if (st.status === 'failed') {
      throw new Error(st.error || 'HLS 转码失败')
    }
  }
  throw new Error('转码超时，请稍后重试')
}

function attachHlsPlaylist(playlistUrl: string) {
  if (!videoRef.value) return
  const video = videoRef.value

  if (Hls.isSupported()) {
    if (hls.value) hls.value.destroy()
    hls.value = new Hls({
      enableWorker: true,
      lowLatencyMode: false,
    })
    hls.value.loadSource(playlistUrl)
    hls.value.attachMedia(video)
    hls.value.on(Hls.Events.ERROR, (_event, data) => {
      console.error('HLS 错误', data)
      if (data.fatal) {
        window.toast?.('HLS 播放失败', 'error', 3000)
      }
    })
  } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
    video.src = playlistUrl
  } else {
    throw new Error('浏览器不支持 HLS')
  }
}

function switchToDirect(storagePath: string) {
  if (!videoRef.value) return
  videoRef.value.src = `/api/v1/stream/direct?path=${encodeURIComponent(storagePath)}`
}

function onLoadedMetadata() {
  if (!videoRef.value || !media.value) return

  if (resumeInfo.value && !resumeInfo.value.completed && resumeInfo.value.progress > 0) {
    videoRef.value.currentTime = resumeInfo.value.progress
  }

  videoRef.value.play().catch(() => {})

  // 5 秒后隐藏快捷键提示
  setTimeout(() => {
    showShortcutTip.value = false
  }, 5000)
}

function onTimeUpdate() {
  if (!videoRef.value || !media.value) return
  const now = Date.now() / 1000
  if (now - lastReportAt.value < REPORT_INTERVAL) return
  lastReportAt.value = now
  reportProgress().catch(console.error)
}

async function reportProgress() {
  if (!videoRef.value || !media.value) return
  await historyApi.record({
    media_id: media.value.id,
    progress: Math.floor(videoRef.value.currentTime),
    duration: Math.floor(videoRef.value.duration || 0),
  })
}

function onEnded() {
  reportProgress().catch(console.error)
}

// ---- 键盘快捷键 ----

function onKeyDown(e: KeyboardEvent) {
  if (!videoRef.value) return

  // 防止默认行为
  const tag = (e.target as HTMLElement).tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA') return

  switch (e.key) {
    case ' ': // 空格 - 播放/暂停
    case 'k':
      e.preventDefault()
      if (videoRef.value.paused) videoRef.value.play()
      else videoRef.value.pause()
      showShortcutTip.value = true
      break
    case 'ArrowLeft': // ← - 后退 10s
      e.preventDefault()
      videoRef.value.currentTime = Math.max(0, videoRef.value.currentTime - 10)
      break
    case 'ArrowRight': // → - 前进 10s
      e.preventDefault()
      videoRef.value.currentTime = Math.min(
        videoRef.value.duration || videoRef.value.currentTime + 10,
        videoRef.value.currentTime + 10
      )
      break
    case 'ArrowUp': // ↑ - 音量 +0.1
      e.preventDefault()
      videoRef.value.volume = Math.min(1, videoRef.value.volume + 0.1)
      window.toast?.(`音量 ${Math.round(videoRef.value.volume * 100)}%`, 'info', 800)
      break
    case 'ArrowDown': // ↓ - 音量 -0.1
      e.preventDefault()
      videoRef.value.volume = Math.max(0, videoRef.value.volume - 0.1)
      window.toast?.(`音量 ${Math.round(videoRef.value.volume * 100)}%`, 'info', 800)
      break
    case 'f':
    case 'F': // F - 全屏切换
      e.preventDefault()
      toggleFullscreen()
      break
    case 'm':
    case 'M': // M - 静音切换
      e.preventDefault()
      videoRef.value.muted = !videoRef.value.muted
      break
    case 'j':
    case 'J': // J - 后退 30s（YouTube 风格）
      e.preventDefault()
      videoRef.value.currentTime = Math.max(0, videoRef.value.currentTime - 30)
      break
    case 'l':
    case 'L': // L - 前进 30s
      e.preventDefault()
      videoRef.value.currentTime = Math.min(
        videoRef.value.duration || videoRef.value.currentTime + 30,
        videoRef.value.currentTime + 30
      )
      break
  }
}

function toggleFullscreen() {
  if (!document.fullscreenElement) {
    containerRef.value?.requestFullscreen?.()
  } else {
    document.exitFullscreen?.()
  }
}

function formatTime(seconds: number) {
  if (!seconds || seconds < 0) return '0:00'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = Math.floor(seconds % 60)
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}

onMounted(async () => {
  let profileId = localStorage.getItem('mediahub_profile_id')
  if (!profileId) {
    profileId = '00000000-0000-0000-0000-000000000001'
    localStorage.setItem('mediahub_profile_id', profileId)
  }
  await loadMedia()

  // 自动聚焦接收键盘事件
  setTimeout(() => containerRef.value?.focus(), 100)
})

onBeforeUnmount(() => {
  reportProgress().catch(() => {})
  if (hls.value) {
    hls.value.destroy()
  }
})
</script>

<style lang="scss" scoped>
.player-page {
  min-height: 100vh;
  background: #000;
  color: #fff;
}

.player-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 60px;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.9), transparent);
  z-index: 100;
  display: flex;
  align-items: center;
  padding: 0 40px;
  gap: 16px;
}

.back-btn {
  background: rgba(255, 255, 255, 0.1);
  border: none;
  color: #fff;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;

  &:hover { background: rgba(255, 255, 255, 0.2); }
}

.title {
  margin: 0;
  font-size: 16px;
  color: #fff;
  font-weight: 500;
  flex: 1;
}

.resume-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  background: rgba(99, 102, 241, 0.2);
  border: 1px solid rgba(99, 102, 241, 0.4);
  border-radius: 20px;
  color: #c7d2fe;
  font-size: 13px;
}

.player-container {
  width: 100%;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #000;
  outline: none;
  position: relative;
}

.video {
  width: 100%;
  height: 100%;
  object-fit: contain;
  max-height: 100vh;
}

.shortcut-tip {
  position: absolute;
  bottom: 100px;
  left: 40px;
  padding: 16px 20px;
  background: rgba(0, 0, 0, 0.85);
  backdrop-filter: blur(10px);
  border-radius: 8px;
  color: #cbd5e1;
  font-size: 13px;
  line-height: 1.8;
  z-index: 50;
  border: 1px solid rgba(255, 255, 255, 0.1);
  animation: fadeIn 0.3s ease;

  div {
    margin: 2px 0;
  }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.player-info {
  position: fixed;
  bottom: 80px;
  left: 40px;
  right: 40px;
  max-width: 600px;
  color: #fff;

  h2 {
    margin: 0 0 8px;
    font-size: 24px;
  }

  .meta {
    display: flex;
    gap: 12px;
    font-size: 13px;
    color: #cbd5e1;
    margin-bottom: 8px;
  }

  .overview {
    margin: 0;
    color: #94a3b8;
    font-size: 14px;
    line-height: 1.5;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
}
</style>
