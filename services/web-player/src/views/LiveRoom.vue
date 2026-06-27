<template>
  <div class="live-room-page">
    <div class="player-header">
      <button class="back-btn" @click="$router.push('/live')">← 返回列表</button>
      <h1 v-if="room" class="title">
        {{ room.title }}
        <span v-if="room.status === 'live'" class="live-tag">直播中</span>
      </h1>
    </div>

    <div class="player-container" ref="containerRef">
      <video ref="videoRef" class="video" controls autoplay muted playsinline></video>

      <div v-if="waiting" class="overlay">
        <LoadingState :message="waitMessage" background />
      </div>

      <div v-if="error" class="overlay error-overlay">
        <p>{{ error }}</p>
        <button class="retry-btn" @click="startPlay">重试</button>
      </div>
    </div>

    <div v-if="room" class="room-info">
      <p v-if="room.description" class="description">{{ room.description }}</p>
      <div class="meta">
        <span v-if="room.status === 'live' && room.started_at">
          开播时间：{{ formatTime(room.started_at) }}
        </span>
        <span v-else-if="room.status === 'idle'">等待主播推流中…</span>
        <span v-else-if="room.status === 'ended'">直播已结束</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import Hls from 'hls.js'
import LoadingState from '@/components/LoadingState.vue'
import { liveApi, type LiveRoom } from '@/api'

const route = useRoute()
const videoRef = ref<HTMLVideoElement>()
const containerRef = ref<HTMLDivElement>()
const room = ref<LiveRoom | null>(null)
const hls = ref<Hls | null>(null)
const waiting = ref(true)
const waitMessage = ref('正在连接直播流…')
const error = ref('')

let pollTimer: ReturnType<typeof setInterval> | null = null
let statusTimer: ReturnType<typeof setInterval> | null = null

function formatTime(iso: string) {
  return new Date(iso).toLocaleString('zh-CN')
}

async function loadRoom() {
  const id = route.params.id as string
  room.value = await liveApi.get(id)
}

function destroyHls() {
  if (hls.value) {
    hls.value.destroy()
    hls.value = null
  }
}

function attachHls(playlistUrl: string) {
  if (!videoRef.value) return

  destroyHls()

  if (Hls.isSupported()) {
    const instance = new Hls({
      enableWorker: true,
      lowLatencyMode: false,
      liveSyncDurationCount: 3,
      liveMaxLatencyDurationCount: 10,
    })
    hls.value = instance
    instance.loadSource(playlistUrl)
    instance.attachMedia(videoRef.value)
    instance.on(Hls.Events.MANIFEST_PARSED, () => {
      waiting.value = false
      error.value = ''
      videoRef.value?.play().catch(() => {})
    })
    instance.on(Hls.Events.ERROR, (_e, data) => {
      if (data.fatal) {
        if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
          waitMessage.value = '等待推流中…'
          waiting.value = true
          instance.startLoad()
        } else {
          error.value = '播放出错，请稍后重试'
          waiting.value = false
        }
      }
    })
  } else if (videoRef.value.canPlayType('application/vnd.apple.mpegurl')) {
    videoRef.value.src = playlistUrl
    videoRef.value.addEventListener('loadedmetadata', () => {
      waiting.value = false
      videoRef.value?.play().catch(() => {})
    })
  } else {
    error.value = '当前浏览器不支持 HLS 播放'
    waiting.value = false
  }
}

async function startPlay() {
  error.value = ''
  waiting.value = true
  waitMessage.value = '正在连接直播流…'

  const id = route.params.id as string
  const playlistUrl = liveApi.playlistUrl(id)

  // 轮询直到 playlist 可用（OBS 推流后自动开始播放）
  for (let i = 0; i < 600; i++) {
    try {
      await refreshStatus()
      const resp = await fetch(playlistUrl)
      if (resp.ok) {
        attachHls(playlistUrl)
        return
      }
      const data = await resp.json().catch(() => null)
      if (data?.error === 'not_streaming' || data?.message?.includes('推流')) {
        waitMessage.value = '等待主播推流中…（请确认 OBS 已点击「开始直播」）'
      } else {
        waitMessage.value = room.value?.status === 'live'
          ? '正在缓冲直播流…'
          : '等待主播开始推流…'
      }
    } catch {
      waitMessage.value = '正在连接直播流…'
    }
    await new Promise((r) => setTimeout(r, 2000))
  }

  waiting.value = false
  error.value = '暂无直播信号。请确认 OBS 正在推流，且 Stream Key 与 CMS 一致'
}

async function refreshStatus() {
  try {
    const id = route.params.id as string
    const updated = await liveApi.get(id)
    if (room.value && updated.status !== room.value.status) {
      room.value = updated
      if (updated.status === 'live' && !hls.value) {
        startPlay()
      }
    } else if (room.value) {
      room.value = updated
    }
  } catch {
    // ignore
  }
}

onMounted(async () => {
  try {
    await loadRoom()
    await startPlay()
    statusTimer = setInterval(refreshStatus, 10000)
  } catch (e: any) {
    error.value = e?.message || '加载直播间失败'
    waiting.value = false
  }
})

onBeforeUnmount(() => {
  destroyHls()
  if (pollTimer) clearInterval(pollTimer)
  if (statusTimer) clearInterval(statusTimer)
})
</script>

<style lang="scss" scoped>
.live-room-page {
  min-height: 100vh;
  background: #000;
  color: #fff;
}

.player-header {
  display: flex;
  align-items: center;
  gap: var(--mh-space-4);
  padding: var(--mh-space-3) var(--mh-space-5);
  background: rgba(0, 0, 0, 0.8);
}

.back-btn {
  background: none;
  border: none;
  color: #aaa;
  cursor: pointer;
  font-size: 14px;

  &:hover { color: #fff; }
}

.title {
  font-size: 18px;
  font-weight: 600;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.live-tag {
  background: #e50914;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 700;
}

.player-container {
  position: relative;
  width: 100%;
  max-width: 1280px;
  margin: 0 auto;
  aspect-ratio: 16 / 9;
  background: #111;
}

.video {
  width: 100%;
  height: 100%;
  display: block;
  background: #000;
}

.overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.error-overlay {
  background: rgba(0, 0, 0, 0.85);
  gap: 16px;

  p { color: #ccc; margin: 0; }
}

.retry-btn {
  background: var(--mh-primary, #6366f1);
  color: #fff;
  border: none;
  padding: 8px 20px;
  border-radius: 6px;
  cursor: pointer;
}

.room-info {
  max-width: 1280px;
  margin: 0 auto;
  padding: var(--mh-space-5);

  .description {
    color: #ccc;
    line-height: 1.6;
    margin: 0 0 12px;
  }

  .meta {
    font-size: 13px;
    color: #888;
  }
}
</style>
