<template>
  <div class="player-page">
    <div class="player-header">
      <button class="back-btn" @click="$router.back()">← 返回</button>
      <h1 v-if="media" class="title">{{ media.title }}</h1>
      <button
        v-if="subtitleTracks.length"
        class="header-btn"
        @click="showSubtitleMenu = !showSubtitleMenu"
      >
        字幕
      </button>
      <button
        v-if="supportsPiP"
        class="header-btn"
        @click="togglePiP"
        title="画中画"
      >
        画中画
      </button>
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

      <div v-if="transcodeLoading" class="transcode-overlay">
        <LoadingState :message="transcodeMessage" :progress="transcodeProgress" background />
      </div>

      <!-- 顶部快捷键提示 -->
      <div v-if="showShortcutTip" class="shortcut-tip">
        <div>空格：播放/暂停</div>
        <div>←→：快退/快进 10 秒</div>
        <div>↑↓：音量</div>
        <div>F：全屏</div>
        <div>M：静音</div>
      </div>
    </div>

    <!-- 字幕/音轨选择器 -->
    <div v-if="showSubtitleMenu" class="subtitle-menu">
      <div class="menu-title">字幕</div>
      <div
        v-for="st in subtitleTracks"
        :key="st.id"
        class="menu-item"
        :class="{ active: selectedSubtitleId === st.id }"
        @click="selectSubtitle(st)"
      >
        {{ st.label || st.language }} {{ st.is_default ? '(默认)' : '' }}
      </div>
      <div
        class="menu-item"
        :class="{ active: !selectedSubtitleId }"
        @click="disableSubtitle"
      >
        关闭字幕
      </div>
    </div>

    <!-- 下一集倒计时 -->
    <div v-if="nextEpisodePrompt" class="next-episode-banner">
      <span>
        {{ nextCountdown > 0 ? `${nextCountdown}s 后自动播放` : '即将播放' }}：
        {{ nextEpisodePrompt.title || `第 ${nextEpisodePrompt.episode_number} 集` }}
      </span>
      <button class="next-btn" @click="playNextEpisode">立即播放</button>
      <button class="dismiss-btn" @click="cancelNextEpisode">取消</button>
    </div>

    <div v-else-if="seriesEnded" class="next-episode-banner series-ended-banner">
      <span>本剧已播放完毕</span>
      <button class="next-btn" @click="goToDetail">返回详情</button>
      <button class="dismiss-btn" @click="seriesEnded = false">关闭</button>
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
import { useRoute, useRouter } from 'vue-router'
import Hls from 'hls.js'
import LoadingState from '@/components/LoadingState.vue'
import { mediaApi, historyApi, catalogApi, type MediaDetail, type ResumeInfo, type EpisodeDetail, type EpisodeNext, type SubtitleTrackInfo } from '@/api'

const route = useRoute()
const router = useRouter()
const videoRef = ref<HTMLVideoElement>()
const containerRef = ref<HTMLDivElement>()
const media = ref<MediaDetail | null>(null)
const resumeInfo = ref<ResumeInfo | null>(null)
const hls = ref<Hls | null>(null)
const showShortcutTip = ref(true)
const currentEpisodeId = ref<string | undefined>()
const playablePath = ref('')
const nextEpisodePrompt = ref<EpisodeNext | null>(null)
const nextCountdown = ref(0)
const seriesEnded = ref(false)
const nextEpisodeTriggered = ref(false)
let nextCountdownTimer: ReturnType<typeof setInterval> | null = null
const transcodeLoading = ref(false)
const transcodeMessage = ref('正在准备播放流…')
const transcodeProgress = ref<number | undefined>()

// 字幕状态
const subtitleTracks = ref<SubtitleTrackInfo[]>([])
const selectedSubtitleId = ref<string | null>(null)
const showSubtitleMenu = ref(false)

// 画中画
const supportsPiP = ref(typeof document !== 'undefined' && 'pictureInPictureEnabled' in document)

function togglePiP() {
  if (!videoRef.value) return
  if (document.pictureInPictureElement) {
    document.exitPictureInPicture().catch(() => {})
  } else {
    videoRef.value.requestPictureInPicture().catch(() => {
      window.toast?.('画中画不可用', 'error', 2000)
    })
  }
}

const lastReportAt = ref(0)
const REPORT_INTERVAL = 10

type PlayableEpisode = EpisodeDetail & { season_number: number }

function isSeriesType(type: string) {
  return type === 'tvshow' || type === 'anime'
}

function listEpisodes(detail: MediaDetail): PlayableEpisode[] {
  const out: PlayableEpisode[] = []
  for (const season of detail.seasons || []) {
    for (const ep of season.episodes || []) {
      if (ep.file_path) out.push({ ...ep, season_number: season.season_number })
    }
  }
  out.sort((a, b) => {
    const seasonDiff = a.season_number - b.season_number
    if (seasonDiff !== 0) return seasonDiff
    return a.episode_number - b.episode_number
  })
  return out
}

function resolvePlayback(detail: MediaDetail, preferredEpisodeId?: string) {
  if (!isSeriesType(detail.type)) {
    return { filePath: detail.storage_path, episodeId: undefined as string | undefined }
  }

  const episodes = listEpisodes(detail)
  if (preferredEpisodeId) {
    const picked = episodes.find((e) => e.id === preferredEpisodeId)
    if (picked?.file_path) {
      return { filePath: picked.file_path, episodeId: picked.id }
    }
  }

  const first = episodes[0]
  if (first?.file_path) {
    return { filePath: first.file_path, episodeId: first.id }
  }

  throw new Error('该剧集暂无可播放的单集文件')
}

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

    const queryEpisode = route.query.episode_id as string | undefined
    const resumeEpisode = resumeInfo.value?.episode_id
    const preferredEpisode = queryEpisode || resumeEpisode

    // 路由明确指定集数时，优先按路由播放，不应用其他集的服务端续播进度。
    if (queryEpisode && resumeInfo.value?.episode_id && queryEpisode !== resumeInfo.value.episode_id) {
      resumeInfo.value = null
    }

    const playback = resolvePlayback(data, preferredEpisode)
    currentEpisodeId.value = playback.episodeId
    playablePath.value = playback.filePath || ''

    await setupVideo()

    // 加载字幕轨
    if (data.id) {
      try {
        const tracks = await catalogApi.subtitles(data.id, currentEpisodeId.value)
        subtitleTracks.value = tracks || []
      } catch {
        // 无字幕
      }
    }
  } catch (e: any) {
    console.error('加载媒资失败', e)
    window.toast?.(`加载失败：${e?.message || '未知错误'}`, 'error', 5000)
  }
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function setupVideo() {
  if (!videoRef.value || !media.value || !playablePath.value) return

  const storagePath = playablePath.value

  let recommended: string | undefined
  try {
    const probeResp = await fetch(
      `/api/v1/stream/probe?path=${encodeURIComponent(storagePath)}`
    )
    if (probeResp.ok) {
      const probe = await probeResp.json()
      recommended = probe.recommended
      if (probe.recommended === 'direct' || probe.direct_playable) {
        const ok = await tryDirectPlay(storagePath)
        if (ok) return
      }
    }
  } catch {
    // probe 失败时走下方 HLS / 直连兜底
  }

  const ext = storagePath.split('.').pop()?.toLowerCase() || ''
  if (ext === 'mp4' || ext === 'm4v' || ext === 'webm') {
    switchToDirect(storagePath)
    return
  }

  const streamId = currentEpisodeId.value || media.value.id
  try {
    await startHLSPlayback(streamId, storagePath)
  } catch (e: any) {
    console.error('HLS 启动失败', e)
    if (recommended !== 'hls_copy' && recommended !== 'hls_transcode') {
      window.toast?.(`HLS 转码失败：${e?.message || '未知错误'}`, 'error', 5000)
    }
    const ok = await tryDirectPlay(storagePath)
    if (!ok) {
      window.toast?.(`播放失败：${e?.message || '未知错误'}`, 'error', 5000)
    }
  }
}

function tryDirectPlay(storagePath: string): Promise<boolean> {
  return new Promise((resolve) => {
    if (!videoRef.value) {
      resolve(false)
      return
    }
    const video = videoRef.value
    if (hls.value) {
      hls.value.destroy()
      hls.value = null
    }

    let settled = false
    const finish = (ok: boolean) => {
      if (settled) return
      settled = true
      video.removeEventListener('loadeddata', onOk)
      video.removeEventListener('error', onFail)
      clearTimeout(timer)
      resolve(ok)
    }
    const onOk = () => finish(true)
    const onFail = () => finish(false)
    const timer = setTimeout(() => finish(false), 8000)

    video.addEventListener('loadeddata', onOk)
    video.addEventListener('error', onFail)
    video.src = `/api/v1/stream/direct?path=${encodeURIComponent(storagePath)}`
    video.load()
  })
}

async function startHLSPlayback(streamId: string, storagePath: string) {
  transcodeLoading.value = true
  try {
    const startUrl = `/api/v1/stream/hls?path=${encodeURIComponent(storagePath)}&media_id=${streamId}`
    const resp = await fetch(startUrl)
    const data = await resp.json()
    if (!resp.ok) {
      throw new Error(data.message || data.error || `转码启动失败 (${resp.status})`)
    }

    transcodeMessage.value = data.copy_video
      ? '正在准备 4K 流…'
      : `正在转码${data.height ? `（${data.height}p）` : ''}，请稍候…`

    const playlistUrl = `/api/v1/stream/hls/${streamId}/playlist.m3u8`

    if (data.status === 'ready' || data.cached) {
      attachHlsPlaylist(playlistUrl, false)
      return
    }
    if (data.status === 'failed') {
      throw new Error(data.error || '转码失败')
    }
    if (data.status === 'done') {
      attachHlsPlaylist(data.playlist || playlistUrl, false)
      return
    }
    if (data.playable) {
      await pollAndPlayHLS(streamId, playlistUrl, true)
      return
    }

    await pollAndPlayHLS(streamId, playlistUrl, false)
  } finally {
    transcodeLoading.value = false
  }
}

async function pollAndPlayHLS(streamId: string, playlistUrl: string, alreadyAttached: boolean) {
  let attached = alreadyAttached
  if (attached) {
    attachHlsPlaylist(playlistUrl, true)
  }

  for (let i = 0; i < 3600; i++) {
    await sleep(1500)
    const resp = await fetch(`/api/v1/stream/hls/${streamId}/status`)
    const st = await resp.json()
    if (!resp.ok) {
      throw new Error(st.message || st.error || 'HLS 状态查询失败')
    }
    if (st.copy_video !== undefined) {
      transcodeMessage.value = st.copy_video
        ? '正在准备 4K 流…'
        : `正在转码${st.height ? `（${st.height}p）` : ''}，请稍候…`
    }
    if (typeof st.progress === 'number' && st.progress > 0) {
      transcodeProgress.value = st.progress
    }
    if (st.status === 'failed') {
      throw new Error(st.error || 'HLS 转码失败')
    }
    const progressive = st.status !== 'done' && st.status !== 'ready'
    if (!attached && (st.playable || st.status === 'done')) {
      attachHlsPlaylist(playlistUrl, progressive)
      attached = true
      transcodeLoading.value = false
    }
    if (st.status === 'done' || st.status === 'ready') {
      return
    }
  }
  throw new Error('转码超时，请稍后重试')
}

function attachHlsPlaylist(playlistUrl: string, progressive = false) {
  if (!videoRef.value) return
  const video = videoRef.value

  if (Hls.isSupported()) {
    if (hls.value) hls.value.destroy()
    hls.value = new Hls({
      enableWorker: true,
      lowLatencyMode: false,
      maxBufferLength: progressive ? 45 : 300,
      maxMaxBufferLength: progressive ? 90 : 600,
      manifestLoadingTimeOut: 20000,
      manifestLoadingMaxRetry: 8,
      manifestLoadingRetryDelay: 1500,
      ...(progressive
        ? {
            liveSyncDurationCount: 2,
            liveMaxLatencyDurationCount: 8,
          }
        : {}),
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

  nextEpisodeTriggered.value = false
  seriesEnded.value = false

  if (resumeInfo.value && !resumeInfo.value.completed && resumeInfo.value.progress > 0) {
    const duration = videoRef.value.duration || 0
    const maxSeek = duration > 0 ? Math.max(0, duration - 3) : resumeInfo.value.progress
    videoRef.value.currentTime = Math.min(resumeInfo.value.progress, maxSeek)
  }

  videoRef.value.play().catch(() => {})

  // 5 秒后隐藏快捷键提示
  setTimeout(() => {
    showShortcutTip.value = false
  }, 5000)
}

function onTimeUpdate() {
  if (!videoRef.value || !media.value) return

  if (
    !nextEpisodeTriggered.value &&
    !nextEpisodePrompt.value &&
    !seriesEnded.value &&
    currentEpisodeId.value &&
    isSeriesType(media.value.type)
  ) {
    const duration = videoRef.value.duration || 0
    if (duration > 0 && videoRef.value.currentTime >= duration * 0.9) {
      triggerNextEpisodePrompt()
    }
  }

  const now = Date.now() / 1000
  if (now - lastReportAt.value < REPORT_INTERVAL) return
  lastReportAt.value = now
  reportProgress().catch(console.error)
}

async function reportProgress() {
  if (!videoRef.value || !media.value) return
  await historyApi.record({
    media_id: media.value.id,
    episode_id: currentEpisodeId.value,
    progress: Math.floor(videoRef.value.currentTime),
    duration: Math.floor(videoRef.value.duration || 0),
  })
}

function onEnded() {
  reportProgress().catch(console.error)
  if (!media.value || !currentEpisodeId.value || !isSeriesType(media.value.type)) return
  if (nextEpisodeTriggered.value) return
  triggerNextEpisodePrompt()
}

function triggerNextEpisodePrompt() {
  if (!media.value || !currentEpisodeId.value || !isSeriesType(media.value.type)) return
  nextEpisodeTriggered.value = true
  seriesEnded.value = false
  catalogApi
    .nextEpisode(media.value.id, currentEpisodeId.value)
    .then((next) => {
      if (next?.id && next.file_path) {
        nextEpisodePrompt.value = next
        startNextCountdown()
        return
      }
      seriesEnded.value = true
    })
    .catch(() => {})
}

function startNextCountdown() {
  if (nextCountdownTimer) {
    clearInterval(nextCountdownTimer)
    nextCountdownTimer = null
  }
  nextCountdown.value = 10
  nextCountdownTimer = setInterval(() => {
    nextCountdown.value--
    if (nextCountdown.value <= 0) {
      if (nextCountdownTimer) {
        clearInterval(nextCountdownTimer)
        nextCountdownTimer = null
      }
      playNextEpisode()
    }
  }, 1000)
}

function cancelNextEpisode() {
  if (nextCountdownTimer) {
    clearInterval(nextCountdownTimer)
    nextCountdownTimer = null
  }
  nextCountdown.value = 0
  nextEpisodePrompt.value = null
}

async function playNextEpisode() {
  const next = nextEpisodePrompt.value
  if (!next?.file_path || !media.value) return
  seriesEnded.value = false
  nextEpisodeTriggered.value = false
  cancelNextEpisode()
  currentEpisodeId.value = next.id
  await router.replace({
    path: `/play/${media.value.id}`,
    query: { episode_id: next.id },
  })
  playablePath.value = next.file_path
  resumeInfo.value = null
  subtitleTracks.value = []
  selectedSubtitleId.value = null
  if (hls.value) {
    hls.value.destroy()
    hls.value = null
  }
  if (videoRef.value) {
    videoRef.value.src = ''
    videoRef.value.load()
  }
  await setupVideo()
}

function goToDetail() {
  if (!media.value) return
  router.push(`/media/${media.value.id}`)
}

// ---- 字幕 ----

function selectSubtitle(track: SubtitleTrackInfo) {
  if (!videoRef.value) return
  selectedSubtitleId.value = track.id
  showSubtitleMenu.value = false

  const video = videoRef.value

  // 禁用所有现有轨道
  for (let i = 0; i < video.textTracks.length; i++) {
    video.textTracks[i].mode = 'disabled'
  }

  // 移除之前动态添加的 track 元素
  video.querySelectorAll('track[data-dynamic]').forEach((el) => el.remove())

  // 如果是外挂字幕，加载 VTT/SRT
  if (track.path) {
    const trackEl = document.createElement('track')
    trackEl.setAttribute('data-dynamic', 'true')
    trackEl.kind = 'subtitles'
    trackEl.src = `/api/v1/stream/direct?path=${encodeURIComponent(track.path)}`
    trackEl.srclang = track.language
    trackEl.label = track.label || track.language
    trackEl.default = true
    video.appendChild(trackEl)

    // 启用新轨道
    setTimeout(() => {
      for (let i = 0; i < video.textTracks.length; i++) {
        video.textTracks[i].mode = i === video.textTracks.length - 1 ? 'showing' : 'disabled'
      }
    }, 100)
  }
}

function disableSubtitle() {
  selectedSubtitleId.value = null
  showSubtitleMenu.value = false
  if (!videoRef.value) return
  for (let i = 0; i < videoRef.value.textTracks.length; i++) {
    videoRef.value.textTracks[i].mode = 'disabled'
  }
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
  await historyApi.ensureProfileId()
  await loadMedia()

  // 自动聚焦接收键盘事件
  setTimeout(() => containerRef.value?.focus(), 100)
})

onBeforeUnmount(() => {
  reportProgress().catch(() => {})
  cancelNextEpisode()
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

.transcode-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.72);
  z-index: 40;
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
  z-index: 20;

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

.next-episode-banner {
  position: fixed;
  bottom: 32px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 200;
  display: flex;
  align-items: center;
  gap: var(--mh-space-3, 12px);
  padding: var(--mh-space-3, 12px) var(--mh-space-5, 20px);
  background: rgba(10, 10, 18, 0.92);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(108, 99, 255, 0.35);
  border-radius: var(--mh-radius-md, 12px);
  box-shadow: var(--mh-shadow-lg, 0 12px 32px rgba(0, 0, 0, 0.45));
  color: var(--mh-text, #f0f0f5);
  font-size: 14px;
}

.series-ended-banner {
  border-color: rgba(16, 185, 129, 0.5);
  background: linear-gradient(90deg, rgba(16, 185, 129, 0.26), rgba(0, 0, 0, 0.78));
}

.next-btn {
  height: 36px;
  padding: 0 16px;
  border: none;
  border-radius: 8px;
  background: var(--mh-primary, #6c63ff);
  color: #fff;
  font-weight: 600;
  cursor: pointer;
}

.dismiss-btn {
  height: 36px;
  padding: 0 12px;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 8px;
  background: transparent;
  color: var(--mh-text-secondary, #a8a8bc);
  cursor: pointer;
}

.header-btn {
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: #fff;
  padding: 6px 14px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;

  &:hover { background: rgba(255, 255, 255, 0.2); }
}

.subtitle-menu {
  position: fixed;
  top: 70px;
  right: 40px;
  z-index: 150;
  min-width: 180px;
  background: rgba(10, 10, 18, 0.95);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  padding: 8px 0;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
}

.menu-title {
  padding: 8px 16px 4px;
  font-size: 12px;
  color: #94a3b8;
  font-weight: 600;
}

.menu-item {
  padding: 10px 16px;
  font-size: 14px;
  color: #cbd5e1;
  cursor: pointer;
  transition: background 0.15s;

  &:hover { background: rgba(255, 255, 255, 0.08); }

  &.active {
    color: #6c63ff;
    font-weight: 500;
  }
}
</style>
