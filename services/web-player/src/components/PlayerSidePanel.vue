<template>
  <aside
    class="player-side-panel"
    :class="{ 'player-side-panel--open': open }"
    aria-label="播放信息面板"
  >
    <div class="panel-header">
      <div class="panel-tabs" role="tablist">
        <button
          v-for="tab in visibleTabs"
          :key="tab.id"
          type="button"
          role="tab"
          class="panel-tab"
          :class="{ 'panel-tab--active': modelTab === tab.id }"
          :aria-selected="modelTab === tab.id"
          @click="modelTab = tab.id"
        >
          {{ tab.label }}
        </button>
      </div>
      <button type="button" class="panel-close mh-icon-btn" title="关闭面板" @click="$emit('close')">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="18" height="18">
          <path d="M18 6L6 18M6 6l12 12" stroke-linecap="round" />
        </svg>
      </button>
    </div>

    <div class="panel-body">
      <!-- 选集 -->
      <section v-show="modelTab === 'episodes'" class="panel-section">
        <div v-if="!seasonsWithEpisodes.length" class="panel-empty">
          暂无选集，请确认文件命名含集数（如 EP01、S01E01）
        </div>
        <div v-for="season in seasonsWithEpisodes" :key="season.id" class="season-block">
          <h3 class="season-title">
            {{ season.title || `第 ${season.season_number} 季` }}
            <span class="season-count">{{ season.episodes.length }} 集</span>
          </h3>
          <div class="episode-grid">
            <button
              v-for="ep in season.episodes"
              :key="ep.id"
              type="button"
              class="episode-chip"
              :class="{ 'episode-chip--active': ep.id === currentEpisodeId }"
              @click="$emit('switch-episode', ep.id)"
            >
              <span class="episode-chip__num">{{ ep.episode_number }}</span>
              <span class="episode-chip__title">{{ ep.title || `第 ${ep.episode_number} 集` }}</span>
            </button>
          </div>
        </div>
      </section>

      <!-- 简介 -->
      <section v-show="modelTab === 'info'" class="panel-section">
        <div v-if="currentEpisode" class="episode-head">
          <span class="episode-badge">第 {{ currentEpisode.episode_number }} 集</span>
          <h3 class="episode-name">{{ currentEpisode.title || `第 ${currentEpisode.episode_number} 集` }}</h3>
        </div>
        <h3 v-else class="media-title">{{ media.title }}</h3>

        <div class="meta-row">
          <span v-if="media.year">{{ media.year }}</span>
          <span v-if="media.rating">⭐ {{ media.rating.toFixed(1) }}</span>
          <span v-if="media.runtime">{{ media.runtime }} 分钟</span>
        </div>

        <div v-if="media.genres?.length" class="genres">
          <span v-for="g in media.genres" :key="g" class="genre-tag">{{ g }}</span>
        </div>

        <p v-if="displayOverview" class="overview">{{ displayOverview }}</p>
        <p v-else class="panel-empty">暂无剧情简介</p>
      </section>

      <!-- 演职员 -->
      <section v-show="modelTab === 'cast'" class="panel-section">
        <p v-if="currentEpisodeId" class="section-hint">本集出场演员（含客串）</p>
        <div v-if="castCredits.length" class="credits-grid">
          <button
            v-for="c in castCredits"
            :key="c.id"
            type="button"
            class="credit-card"
            @click="$emit('open-person', c)"
          >
            <div class="credit-avatar">
              <img
                v-if="creditAvatar(c)"
                :src="creditAvatar(c)"
                :alt="c.person?.name"
                loading="lazy"
              />
              <span v-else>{{ c.person?.name?.slice(0, 1) || '?' }}</span>
            </div>
            <div class="credit-name">{{ c.person?.name }}</div>
            <div v-if="c.character_name" class="credit-role">饰 {{ c.character_name }}</div>
          </button>
        </div>
        <p v-else class="panel-empty">暂无演职员信息</p>
      </section>

      <!-- 音轨 / 字幕 -->
      <section v-show="modelTab === 'tracks'" class="panel-section">
        <div class="track-group">
          <h4 class="track-group-title">音轨</h4>
          <div v-if="audioTracks.length" class="track-list">
            <button
              v-for="track in audioTracks"
              :key="track.index"
              type="button"
              class="track-item"
              :class="{ 'track-item--active': selectedAudioIndex === track.index }"
              @click="$emit('select-audio', track.index)"
            >
              <span class="track-label">{{ track.label || track.language || `音轨 ${track.index}` }}</span>
              <span class="track-meta">
                {{ track.codec?.toUpperCase() }}
                <template v-if="track.channels"> · {{ track.channels }}ch</template>
                <template v-if="track.is_default"> · 默认</template>
              </span>
            </button>
          </div>
          <p v-else class="panel-empty">仅检测到默认音轨</p>
          <p v-if="audioTracks.length > 1 && !directPlayable" class="track-hint">
            HLS 转码模式下暂使用默认音轨；如需切换请尝试直连播放。
          </p>
        </div>

        <div class="track-group">
          <h4 class="track-group-title">字幕</h4>
          <div class="track-list">
            <button
              type="button"
              class="track-item"
              :class="{ 'track-item--active': !selectedSubtitleId }"
              @click="$emit('disable-subtitle')"
            >
              <span class="track-label">关闭字幕</span>
            </button>
            <button
              v-for="st in subtitleTracks"
              :key="st.id"
              type="button"
              class="track-item"
              :class="{ 'track-item--active': selectedSubtitleId === st.id }"
              @click="$emit('select-subtitle', st)"
            >
              <span class="track-label">{{ st.label || st.language }}</span>
              <span class="track-meta">{{ st.format?.toUpperCase() }}<template v-if="st.is_default"> · 默认</template></span>
            </button>
          </div>
          <p v-if="!subtitleTracks.length" class="panel-empty">暂无外挂字幕</p>
        </div>

        <div v-if="embeddedSubtitleTracks.length" class="track-group">
          <h4 class="track-group-title">内嵌字幕</h4>
          <div class="track-list track-list--readonly">
            <div v-for="st in embeddedSubtitleTracks" :key="st.index" class="track-item track-item--static">
              <span class="track-label">{{ st.label || st.language || `字幕 ${st.index}` }}</span>
              <span class="track-meta">{{ st.codec?.toUpperCase() }}</span>
            </div>
          </div>
          <p class="track-hint">内嵌字幕需在转码时烧录，当前仅展示信息。</p>
        </div>
      </section>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type {
  MediaDetail,
  MediaCredit,
  EpisodeDetail,
  SeasonDetail,
  SubtitleTrackInfo,
  StreamTrackInfo,
} from '@/api'

const props = defineProps<{
  open: boolean
  media: MediaDetail
  currentEpisodeId?: string
  castCredits: MediaCredit[]
  audioTracks: StreamTrackInfo[]
  embeddedSubtitleTracks: StreamTrackInfo[]
  subtitleTracks: SubtitleTrackInfo[]
  selectedAudioIndex: number | null
  selectedSubtitleId: string | null
  directPlayable: boolean
}>()

const modelTab = defineModel<string>('tab', { default: 'info' })

defineEmits<{
  close: []
  'switch-episode': [episodeId: string]
  'select-audio': [streamIndex: number]
  'select-subtitle': [track: SubtitleTrackInfo]
  'disable-subtitle': []
  'open-person': [credit: MediaCredit]
}>()

const isSeries = computed(() => props.media.type === 'tvshow' || props.media.type === 'anime')

const seasonsWithEpisodes = computed((): Array<SeasonDetail & { episodes: EpisodeDetail[] }> => {
  if (!props.media.seasons) return []
  return props.media.seasons
    .map((s) => ({
      ...s,
      episodes: (s.episodes || [])
        .filter((ep) => ep.file_path)
        .sort((a, b) => a.episode_number - b.episode_number),
    }))
    .filter((s) => s.episodes.length > 0)
    .sort((a, b) => a.season_number - b.season_number)
})

const currentEpisode = computed(() => {
  if (!props.currentEpisodeId) return null
  for (const season of props.media.seasons || []) {
    const ep = season.episodes?.find((e) => e.id === props.currentEpisodeId)
    if (ep) return ep
  }
  return null
})

const displayOverview = computed(() => {
  if (currentEpisode.value?.overview) return currentEpisode.value.overview
  return props.media.overview || ''
})

const visibleTabs = computed(() => {
  const tabs = [{ id: 'info', label: '简介' }]
  if (isSeries.value) tabs.unshift({ id: 'episodes', label: '选集' })
  tabs.push({ id: 'cast', label: '演职员' })
  tabs.push({ id: 'tracks', label: '音轨/字幕' })
  return tabs
})

function creditAvatar(c: MediaCredit) {
  return c.person?.profile_url || c.person?.profile_path || ''
}
</script>

<style lang="scss" scoped>
.player-side-panel {
  position: fixed;
  top: var(--mh-topbar-height);
  right: 0;
  bottom: 0;
  width: min(400px, 100vw);
  z-index: 90;
  display: flex;
  flex-direction: column;
  background: var(--mh-glass-bg);
  backdrop-filter: var(--mh-glass-blur);
  border-left: 1px solid var(--mh-outline);
  transform: translateX(100%);
  transition: transform var(--mh-duration-slow) var(--mh-ease-out);
  box-shadow: var(--mh-shadow-xl);

  &--open {
    transform: translateX(0);
  }
}

.panel-header {
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
  padding: var(--mh-space-3) var(--mh-space-3) var(--mh-space-2);
  border-bottom: 1px solid var(--mh-outline);
  flex-shrink: 0;
}

.panel-tabs {
  display: flex;
  gap: 4px;
  flex: 1;
  overflow-x: auto;
  scrollbar-width: none;

  &::-webkit-scrollbar {
    display: none;
  }
}

.panel-tab {
  flex-shrink: 0;
  height: 32px;
  padding: 0 12px;
  border: none;
  border-radius: var(--mh-radius-full);
  background: transparent;
  color: var(--mh-text-tertiary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background var(--mh-duration-fast), color var(--mh-duration-fast);

  &:hover {
    color: var(--mh-text-secondary);
    background: rgba(255, 255, 255, 0.06);
  }

  &--active {
    color: var(--mh-text);
    background: var(--mh-primary-muted);
  }
}

.panel-close {
  flex-shrink: 0;
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--mh-space-4);
}

.panel-section {
  animation: mh-fade-up var(--mh-duration-fast) var(--mh-ease-out) both;
}

.section-hint {
  margin: 0 0 var(--mh-space-3);
  font-size: 12px;
  color: var(--mh-text-muted);
}

.panel-empty {
  margin: 0;
  color: var(--mh-text-muted);
  font-size: 13px;
  line-height: 1.6;
}

.season-block + .season-block {
  margin-top: var(--mh-space-5);
}

.season-title {
  margin: 0 0 var(--mh-space-3);
  font-size: 14px;
  font-weight: 600;
  color: var(--mh-text-secondary);
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
}

.season-count {
  font-size: 12px;
  font-weight: 400;
  color: var(--mh-text-muted);
}

.episode-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(88px, 1fr));
  gap: var(--mh-space-2);
}

.episode-chip {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  padding: 10px 12px;
  border: 1px solid var(--mh-outline);
  border-radius: var(--mh-radius-sm);
  background: rgba(255, 255, 255, 0.04);
  color: var(--mh-text-secondary);
  cursor: pointer;
  text-align: left;
  transition: border-color var(--mh-duration-fast), background var(--mh-duration-fast);

  &:hover {
    border-color: var(--mh-outline-strong);
    background: rgba(255, 255, 255, 0.08);
  }

  &--active {
    border-color: rgba(10, 132, 255, 0.55);
    background: var(--mh-primary-muted);
    color: var(--mh-text);
  }
}

.episode-chip__num {
  font-size: 15px;
  font-weight: 700;
  color: var(--mh-primary-hover);
}

.episode-chip__title {
  font-size: 11px;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.episode-head {
  margin-bottom: var(--mh-space-3);
}

.episode-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: var(--mh-radius-full);
  background: var(--mh-primary-muted);
  color: var(--mh-primary-hover);
  font-size: 11px;
  font-weight: 600;
  margin-bottom: var(--mh-space-2);
}

.episode-name,
.media-title {
  margin: 0 0 var(--mh-space-3);
  font-size: 18px;
  font-weight: 700;
  line-height: 1.3;
}

.meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--mh-space-3);
  font-size: 13px;
  color: var(--mh-text-tertiary);
  margin-bottom: var(--mh-space-3);
}

.genres {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: var(--mh-space-4);
}

.genre-tag {
  padding: 4px 10px;
  border-radius: var(--mh-radius-full);
  background: rgba(255, 255, 255, 0.06);
  font-size: 12px;
  color: var(--mh-text-secondary);
}

.overview {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: var(--mh-text-secondary);
  white-space: pre-wrap;
}

.credits-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(96px, 1fr));
  gap: var(--mh-space-3);
}

.credit-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 0;
  border: none;
  background: transparent;
  color: inherit;
  cursor: pointer;
  text-align: center;

  &:hover .credit-avatar {
    transform: scale(1.04);
    box-shadow: var(--mh-shadow-md);
  }
}

.credit-avatar {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  overflow: hidden;
  background: var(--mh-surface-variant);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  font-weight: 600;
  color: var(--mh-text-muted);
  transition: transform var(--mh-duration-fast), box-shadow var(--mh-duration-fast);

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.credit-name {
  font-size: 12px;
  font-weight: 600;
  line-height: 1.3;
}

.credit-role {
  font-size: 11px;
  color: var(--mh-text-muted);
  line-height: 1.3;
}

.track-group + .track-group {
  margin-top: var(--mh-space-5);
}

.track-group-title {
  margin: 0 0 var(--mh-space-3);
  font-size: 13px;
  font-weight: 600;
  color: var(--mh-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.track-list {
  display: flex;
  flex-direction: column;
  gap: 6px;

  &--readonly .track-item {
    cursor: default;
  }
}

.track-item {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--mh-outline);
  border-radius: var(--mh-radius-sm);
  background: rgba(255, 255, 255, 0.03);
  color: var(--mh-text-secondary);
  cursor: pointer;
  text-align: left;
  transition: border-color var(--mh-duration-fast), background var(--mh-duration-fast);

  &:hover:not(.track-item--static) {
    background: rgba(255, 255, 255, 0.07);
  }

  &--active {
    border-color: rgba(10, 132, 255, 0.5);
    background: var(--mh-primary-muted);
    color: var(--mh-text);
  }

  &--static {
    opacity: 0.85;
  }
}

.track-label {
  font-size: 14px;
  font-weight: 500;
}

.track-meta {
  font-size: 11px;
  color: var(--mh-text-muted);
}

.track-hint {
  margin: var(--mh-space-2) 0 0;
  font-size: 12px;
  color: var(--mh-text-muted);
  line-height: 1.5;
}

@media (max-width: 900px) {
  .player-side-panel {
    top: auto;
    left: 0;
    right: 0;
    width: 100%;
    height: min(62vh, 520px);
    border-left: none;
    border-top: 1px solid var(--mh-outline);
    border-radius: var(--mh-radius-xl) var(--mh-radius-xl) 0 0;
    transform: translateY(100%);

    &--open {
      transform: translateY(0);
    }
  }
}
</style>
