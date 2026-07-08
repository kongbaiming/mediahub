<template>
  <div class="player-page" :class="{ 'player-page--panel-open': sidePanelOpen }">
    <div class="player-header mh-topbar mh-sub-topbar">
      <button type="button" class="mh-back-btn" @click="$router.back()">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18" aria-hidden="true">
          <path d="M15 18l-6-6 6-6" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        <span>返回</span>
      </button>
      <div v-if="media" class="title-block">
        <h1 class="title">{{ displayTitle }}</h1>
        <p v-if="displaySubtitle" class="subtitle">{{ displaySubtitle }}</p>
      </div>
      <div class="header-actions">
        <button
          type="button"
          class="mh-icon-btn"
          :class="{ 'mh-icon-btn--active': sidePanelOpen && panelTab === 'episodes' }"
          title="选集"
          @click="togglePanel('episodes')"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
            <rect x="3" y="4" width="18" height="4" rx="1" />
            <rect x="3" y="10" width="18" height="4" rx="1" />
            <rect x="3" y="16" width="18" height="4" rx="1" />
          </svg>
        </button>
        <button
          type="button"
          class="mh-icon-btn"
          :class="{ 'mh-icon-btn--active': sidePanelOpen && panelTab === 'tracks' }"
          title="音轨 / 字幕"
          @click="togglePanel('tracks')"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
            <path d="M12 3v18M8 7h8M7 12h10M8 17h8" stroke-linecap="round" />
          </svg>
        </button>
        <button
          type="button"
          class="mh-icon-btn"
          :class="{ 'mh-icon-btn--active': sidePanelOpen && panelTab === 'info' }"
          title="简介"
          @click="togglePanel('info')"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
            <circle cx="12" cy="12" r="9" />
            <path d="M12 10v6M12 7h.01" stroke-linecap="round" />
          </svg>
        </button>
        <button
          type="button"
          class="mh-icon-btn"
          :class="{ 'mh-icon-btn--active': sidePanelOpen && panelTab === 'cast' }"
          title="演职员"
          @click="togglePanel('cast')"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
            <circle cx="9" cy="8" r="3" />
            <circle cx="17" cy="10" r="2.5" />
            <path d="M3 20c0-3 2.5-5 6-5s6 2 6 5M14 20c0-2 1.5-3.5 3.5-3.5" stroke-linecap="round" />
          </svg>
        </button>
        <button
          v-if="supportsPiP"
          type="button"
          class="mh-icon-btn"
          title="画中画"
          @click="togglePiP"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
            <rect x="3" y="5" width="18" height="14" rx="2" />
            <rect x="11" y="11" width="8" height="6" rx="1" fill="currentColor" opacity="0.35" />
          </svg>
        </button>
      </div>
      <span v-if="resumeInfo && !resumeInfo.completed" class="resume-badge">
        续播 {{ formatTime(resumeInfo.progress) }} / {{ formatTime(resumeInfo.duration) }}
      </span>
    </div>

    <div
      class="player-main"
      :class="{ 'player-main--panel-open': sidePanelOpen }"
    >
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

        <div v-if="showShortcutTip" class="shortcut-tip">
          <div>空格：播放/暂停</div>
          <div>←→：快退/快进 10 秒</div>
          <div>↑↓：音量</div>
          <div>F：全屏</div>
          <div>M：静音</div>
        </div>
      </div>

      <Teleport to="body">
        <PlayerSidePanel
          v-if="media"
          v-model:tab="panelTab"
          :open="sidePanelOpen"
          :media="media"
          :current-episode-id="currentEpisodeId"
          :cast-credits="castCredits"
          :audio-tracks="audioTracks"
          :embedded-subtitle-tracks="embeddedSubtitleTracks"
          :subtitle-tracks="subtitleTracks"
          :selected-audio-index="selectedAudioIndex"
          :selected-subtitle-id="selectedSubtitleId"
          :direct-playable="directPlayable"
          @close="sidePanelOpen = false"
          @switch-episode="switchToEpisode"
          @select-audio="selectAudioTrack"
          @select-subtitle="selectSubtitle"
          @disable-subtitle="disableSubtitle"
          @open-person="openPerson"
        />
      </Teleport>
    </div>

    <Teleport to="body">
      <div
        v-if="sidePanelOpen"
        class="panel-backdrop"
        @click="sidePanelOpen = false"
      ></div>
    </Teleport>

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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Hls from 'hls.js'
import LoadingState from '@/components/LoadingState.vue'
import PlayerSidePanel from '@/components/PlayerSidePanel.vue'
import {
  mediaApi,
  historyApi,
  catalogApi,
  streamApi,
  type MediaDetail,
  type ResumeInfo,
  type EpisodeDetail,
  type EpisodeNext,
  type SubtitleTrackInfo,
  type StreamTrackInfo,
  type MediaCredit,
} from '@/api'

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
let playbackToken = 0
const transcodeLoading = ref(false)
const transcodeMessage = ref('正在准备播放流…')
const transcodeProgress = ref<number | undefined>()

// 字幕 / 音轨 / 侧栏
const subtitleTracks = ref<SubtitleTrackInfo[]>([])
const selectedSubtitleId = ref<string | null>(null)
const audioTracks = ref<StreamTrackInfo[]>([])
const embeddedSubtitleTracks = ref<StreamTrackInfo[]>([])
const selectedAudioIndex = ref<number | null>(null)
const defaultAudioIndex = ref<number | null>(null)
const directPlayable = ref(false)
const useDirectPlay = ref(false)
const castCredits = ref<MediaCredit[]>([])
const sidePanelOpen = ref(false)
const panelTab = ref('info')

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

const currentEpisode = computed((): EpisodeDetail | null => {
  if (!media.value || !currentEpisodeId.value) return null
  for (const season of media.value.seasons || []) {
    const ep = season.episodes?.find((e) => e.id === currentEpisodeId.value)
    if (ep) return ep
  }
  return null
})

const displayTitle = computed(() => {
  if (!media.value) return ''
  if (currentEpisode.value) {
    return `${media.value.title} · 第 ${currentEpisode.value.episode_number} 集`
  }
  return media.value.title
})

const displaySubtitle = computed(() => {
  if (currentEpisode.value?.title) return currentEpisode.value.title
  if (media.value?.year) return String(media.value.year)
  return ''
})

function togglePanel(tab: string) {
  if (sidePanelOpen.value && panelTab.value === tab) {
    sidePanelOpen.value = false
    return
  }
  panelTab.value = tab
  sidePanelOpen.value = true
}

function openPerson(c: MediaCredit) {
  const personId = c.person?.id
  const tmdbPersonId = c.person?.tmdb_person_id
  const nilUUID = '00000000-0000-0000-0000-000000000000'
  if (personId && personId !== nilUUID) {
    router.push(`/person/${personId}`)
  } else if (tmdbPersonId) {
    router.push(`/person/tmdb/${tmdbPersonId}`)
  }
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

function bumpPlaybackToken() {
  playbackToken += 1
  return playbackToken
}

function isActivePlayback(token: number) {
  return token === playbackToken
}

async function loadMedia() {
  try {
    const data = await mediaApi.get(route.params.id as string)
    media.value = data

    const queryEpisode = route.query.episode_id as string | undefined

    let latestResume: ResumeInfo | null = null
    if (!queryEpisode && isSeriesType(data.type)) {
      try {
        latestResume = await historyApi.getResume(data.id)
      } catch {
        // no history
      }
    }

    const preferredEpisode = queryEpisode || latestResume?.episode_id
    const playback = resolvePlayback(data, preferredEpisode)
    currentEpisodeId.value = playback.episodeId
    playablePath.value = playback.filePath || ''

    await applyResumeForEpisode(data.id, currentEpisodeId.value)

    await setupVideo(bumpPlaybackToken())

    await loadAuxiliaryData()

    if (isSeriesType(data.type)) {
      panelTab.value = 'episodes'
      sidePanelOpen.value = true
    }
  } catch (e: any) {
    console.error('加载媒资失败', e)
    window.toast?.(`加载失败：${e?.message || '未知错误'}`, 'error', 5000)
  }
}

async function loadAuxiliaryData() {
  if (!media.value) return

  const tasks: Promise<void>[] = []

  tasks.push(
    catalogApi
      .credits(media.value.id, 'actor', currentEpisodeId.value)
      .then((items) => {
        castCredits.value = items || []
      })
      .catch(() => {
        castCredits.value = []
      }),
  )

  tasks.push(
    catalogApi
      .subtitles(media.value.id, currentEpisodeId.value)
      .then((tracks) => {
        subtitleTracks.value = tracks || []
        const def = subtitleTracks.value.find((t) => t.is_default)
        if (def && !selectedSubtitleId.value) {
          selectedSubtitleId.value = def.id
        }
      })
      .catch(() => {
        subtitleTracks.value = []
      }),
  )

  if (playablePath.value) {
    tasks.push(
      streamApi
        .probe(playablePath.value)
        .then((probe) => {
          audioTracks.value = probe.audio_tracks || []
          embeddedSubtitleTracks.value = probe.embedded_subtitle_tracks || []
          directPlayable.value = !!probe.direct_playable
          defaultAudioIndex.value = probe.default_audio_index ?? audioTracks.value[0]?.index ?? null
          if (selectedAudioIndex.value == null) {
            selectedAudioIndex.value = defaultAudioIndex.value
          }
        })
        .catch(() => {
          audioTracks.value = []
          embeddedSubtitleTracks.value = []
        }),
    )
  }

  await Promise.all(tasks)
}

function shouldApplyResume(resume: ResumeInfo | null | undefined, episodeId?: string) {
  if (!resume || resume.completed || resume.progress <= 0) return false
  if (!episodeId) return true
  if (!resume.episode_id) return false
  return resume.episode_id === episodeId
}

async function applyResumeForEpisode(mediaId: string, episodeId?: string) {
  resumeInfo.value = null
  try {
    const resume = await historyApi.getResume(mediaId, episodeId)
    if (shouldApplyResume(resume, episodeId)) {
      resumeInfo.value = resume
    }
  } catch {
    // no history
  }
}

function directStreamUrl(storagePath: string, audioStream?: number | null) {
  const params = new URLSearchParams({ path: storagePath })
  if (
    audioStream != null &&
    defaultAudioIndex.value != null &&
    audioStream !== defaultAudioIndex.value
  ) {
    params.set('audio_stream', String(audioStream))
  }
  return `/api/v1/stream/direct?${params.toString()}`
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function setupVideo(token?: number) {
  if (!videoRef.value || !media.value || !playablePath.value) return
  const activeToken = token ?? bumpPlaybackToken()

  const storagePath = playablePath.value

  let recommended: string | undefined
  try {
    const probeResp = await fetch(
      `/api/v1/stream/probe?path=${encodeURIComponent(storagePath)}`
    )
    if (!isActivePlayback(activeToken)) return
    if (probeResp.ok) {
      const probe = await probeResp.json()
      recommended = probe.recommended
      if (probe.recommended === 'direct' || probe.direct_playable) {
        directPlayable.value = !!probe.direct_playable
        const ok = await tryDirectPlay(storagePath)
        if (!isActivePlayback(activeToken)) return
        if (ok) {
          useDirectPlay.value = true
          return
        }
      }
    }
  } catch {
    // probe 失败时走下方 HLS / 直连兜底
  }

  if (!isActivePlayback(activeToken)) return

  const ext = storagePath.split('.').pop()?.toLowerCase() || ''
  if (ext === 'mp4' || ext === 'm4v' || ext === 'webm') {
    switchToDirect(storagePath)
    useDirectPlay.value = true
    return
  }

  useDirectPlay.value = false

  const streamId = currentEpisodeId.value || media.value.id
  try {
    await startHLSPlayback(streamId, storagePath, activeToken)
  } catch (e: any) {
    if (!isActivePlayback(activeToken)) return
    console.error('HLS 启动失败', e)
    if (recommended !== 'hls_copy' && recommended !== 'hls_transcode') {
      window.toast?.(`HLS 转码失败：${e?.message || '未知错误'}`, 'error', 5000)
    }
    const ok = await tryDirectPlay(storagePath)
    if (!isActivePlayback(activeToken)) return
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
    video.src = directStreamUrl(storagePath, selectedAudioIndex.value)
    video.load()
  })
}

async function startHLSPlayback(streamId: string, storagePath: string, token: number) {
  transcodeLoading.value = true
  try {
    const startUrl = `/api/v1/stream/hls?path=${encodeURIComponent(storagePath)}&media_id=${streamId}`
    const resp = await fetch(startUrl)
    if (!isActivePlayback(token)) return
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
      await pollAndPlayHLS(streamId, playlistUrl, true, token)
      return
    }

    await pollAndPlayHLS(streamId, playlistUrl, false, token)
  } finally {
    if (isActivePlayback(token)) {
      transcodeLoading.value = false
    }
  }
}

async function pollAndPlayHLS(
  streamId: string,
  playlistUrl: string,
  alreadyAttached: boolean,
  token: number,
) {
  let attached = alreadyAttached
  if (attached) {
    attachHlsPlaylist(playlistUrl, true)
  }

  for (let i = 0; i < 3600; i++) {
    if (!isActivePlayback(token)) return
    await sleep(1500)
    if (!isActivePlayback(token)) return
    const resp = await fetch(`/api/v1/stream/hls/${streamId}/status`)
    const st = await resp.json()
    if (!isActivePlayback(token)) return
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
  videoRef.value.src = directStreamUrl(storagePath, selectedAudioIndex.value)
}

function onLoadedMetadata() {
  if (!videoRef.value || !media.value) return

  nextEpisodeTriggered.value = false
  seriesEnded.value = false

  if (shouldApplyResume(resumeInfo.value, currentEpisodeId.value) && resumeInfo.value) {
    const duration = videoRef.value.duration || 0
    const maxSeek = duration > 0 ? Math.max(0, duration - 3) : resumeInfo.value.progress
    videoRef.value.currentTime = Math.min(resumeInfo.value.progress, maxSeek)
  }

  videoRef.value.play().catch(() => {})

  if (selectedSubtitleId.value) {
    const track = subtitleTracks.value.find((t) => t.id === selectedSubtitleId.value)
    if (track) selectSubtitle(track)
  }

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
  await switchToEpisode(next.id, next.file_path)
}

async function switchToEpisode(episodeId: string, filePath?: string) {
  if (!media.value || !episodeId) return
  if (episodeId === currentEpisodeId.value) return

  const token = bumpPlaybackToken()
  reportProgress().catch(() => {})

  seriesEnded.value = false
  nextEpisodeTriggered.value = false
  cancelNextEpisode()

  const path = filePath || listEpisodes(media.value).find((e) => e.id === episodeId)?.file_path
  if (!path) {
    window.toast?.('该集暂无可播放文件', 'error', 3000)
    return
  }

  transcodeLoading.value = true
  transcodeMessage.value = '正在切换集数…'
  transcodeProgress.value = undefined

  currentEpisodeId.value = episodeId
  playablePath.value = path
  selectedSubtitleId.value = null
  selectedAudioIndex.value = null
  defaultAudioIndex.value = null
  subtitleTracks.value = []
  audioTracks.value = []
  embeddedSubtitleTracks.value = []
  castCredits.value = []

  if (hls.value) {
    hls.value.destroy()
    hls.value = null
  }
  if (videoRef.value) {
    videoRef.value.src = ''
    videoRef.value.load()
  }

  useDirectPlay.value = false

  try {
    await router.replace({
      path: `/play/${media.value.id}`,
      query: { episode_id: episodeId },
    })
    if (!isActivePlayback(token)) return

    await applyResumeForEpisode(media.value.id, episodeId)
    if (!isActivePlayback(token)) return

    await setupVideo(token)
    if (!isActivePlayback(token)) return

    await loadAuxiliaryData()
  } catch (e: any) {
    if (!isActivePlayback(token)) return
    console.error('切换集数失败', e)
    window.toast?.(`切换集数失败：${e?.message || '未知错误'}`, 'error', 4000)
  } finally {
    if (isActivePlayback(token)) {
      transcodeLoading.value = false
    }
  }
}

function goToDetail() {
  if (!media.value) return
  router.push(`/media/${media.value.id}`)
}

// ---- 字幕 ----

function selectSubtitle(track: SubtitleTrackInfo) {
  if (!videoRef.value) return
  selectedSubtitleId.value = track.id

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
  if (!videoRef.value) return
  for (let i = 0; i < videoRef.value.textTracks.length; i++) {
    videoRef.value.textTracks[i].mode = 'disabled'
  }
}

async function selectAudioTrack(streamIndex: number) {
  if (selectedAudioIndex.value === streamIndex) return
  if (!directPlayable.value && !useDirectPlay.value) {
    window.toast?.('当前为 HLS 转码模式，暂无法切换音轨', 'info', 3000)
    return
  }
  if (!playablePath.value || !videoRef.value) return

  const prevTime = videoRef.value.currentTime
  selectedAudioIndex.value = streamIndex
  panelTab.value = 'tracks'

  if (hls.value) {
    hls.value.destroy()
    hls.value = null
  }

  transcodeLoading.value = true
  transcodeMessage.value = '正在切换音轨…'
  try {
    const ok = await tryDirectPlay(playablePath.value)
    if (!ok) {
      window.toast?.('音轨切换失败', 'error', 3000)
      return
    }
    useDirectPlay.value = true
    videoRef.value.currentTime = prevTime
    videoRef.value.play().catch(() => {})
    window.toast?.('音轨已切换', 'success', 2000)
  } finally {
    transcodeLoading.value = false
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

watch(
  () => route.query.episode_id as string | undefined,
  async (episodeId, prevEpisodeId) => {
    if (!media.value || !episodeId || episodeId === prevEpisodeId) return
    if (episodeId === currentEpisodeId.value) return
    await switchToEpisode(episodeId)
  },
)

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

.player-main {
  display: flex;
  min-height: 100vh;
  padding-top: var(--mh-topbar-height);

  &--panel-open .player-container {
    @media (min-width: 901px) {
      width: calc(100% - min(400px, 100vw));
    }
  }
}

.panel-backdrop {
  display: none;

  @media (max-width: 900px) {
    display: block;
    position: fixed;
    inset: 0;
    z-index: 105;
    background: rgba(0, 0, 0, 0.45);
  }
}

.player-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 100;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.88), rgba(0, 0, 0, 0.35));
  border-bottom: none;
  padding: 0 var(--mh-page-gutter);
  flex-wrap: wrap;
  gap: var(--mh-space-2);
}

.title-block {
  flex: 1;
  min-width: 0;
}

.title {
  margin: 0;
  font-size: 16px;
  color: #fff;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.subtitle {
  margin: 2px 0 0;
  font-size: 12px;
  color: var(--mh-text-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
  margin-left: auto;
}

.resume-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 12px;
  background: var(--mh-primary-muted);
  border: 1px solid rgba(10, 132, 255, 0.35);
  border-radius: var(--mh-radius-full);
  color: var(--mh-primary-hover);
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}

.player-container {
  flex: 1;
  width: 100%;
  min-height: calc(100vh - var(--mh-topbar-height));
  display: flex;
  align-items: center;
  justify-content: center;
  background: #000;
  outline: none;
  position: relative;
  transition: width var(--mh-duration-slow) var(--mh-ease-out);
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
</style>
