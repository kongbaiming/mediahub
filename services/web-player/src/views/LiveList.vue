<template>
  <div class="live-list-page">
    <header class="live-topbar mh-topbar mh-sub-topbar">
      <button class="back-btn" @click="$router.push('/')">← 首页</button>
      <h1 class="page-title">直播</h1>
      <div class="topbar-search">
        <input
          v-model="searchInput"
          type="search"
          placeholder="搜索频道…"
          @keyup.enter="applySearch"
        />
      </div>
    </header>

    <div class="live-list-body">
    <div class="filter-bar">
      <div class="filter-row">
        <span class="filter-label">类型</span>
        <div class="pill-tabs">
          <button
            v-for="t in typeTabs"
            :key="t.value"
            class="pill"
            :class="{ active: roomType === t.value }"
            @click="setRoomType(t.value)"
          >
            {{ t.label }}
          </button>
        </div>
      </div>

      <div class="filter-row">
        <span class="filter-label">状态</span>
        <div class="pill-tabs">
          <button
            v-for="s in statusTabs"
            :key="s.value"
            class="pill"
            :class="{ active: statusFilter === s.value }"
            @click="setStatus(s.value)"
          >
            {{ s.label }}
          </button>
        </div>
      </div>

      <div v-if="groupTabs.length > 0" class="filter-row filter-row--groups">
        <span class="filter-label">栏目</span>
        <div ref="groupScrollRef" class="group-scroll">
          <button
            v-for="g in groupTabs"
            :key="g.value"
            class="group-tab"
            :class="{ active: groupTitle === g.value }"
            @click="setGroup(g.value)"
          >
            <span class="group-tab__name">{{ g.label }}</span>
            <span class="group-tab__count">{{ g.count }}</span>
          </button>
        </div>
      </div>
    </div>

    <main class="content">
      <LoadingState v-if="loading && rooms.length === 0" message="加载频道…" />

      <div v-else-if="rooms.length === 0" class="empty">
        <p>{{ hasFilters ? '没有匹配的频道' : '暂无直播频道' }}</p>
        <p v-if="!hasFilters" class="hint">管理员可在 CMS 创建推流间或导入 M3U 列表</p>
      </div>

      <div v-else class="room-grid">
        <article
          v-for="room in rooms"
          :key="room.id"
          class="room-card"
          @click="$router.push(`/live/${room.id}`)"
        >
          <div class="room-cover" :style="coverStyle(room)">
            <span v-if="room.status === 'live'" class="live-badge">LIVE</span>
            <span v-else-if="room.room_type === 'iptv'" class="iptv-badge">IPTV</span>
            <span v-else class="idle-badge">推流</span>
          </div>
          <div class="room-info">
            <h3>{{ room.title }}</h3>
            <div class="meta">
              <span v-if="room.group_title" class="group-tag">{{ room.group_title }}</span>
              <span v-if="room.status === 'live' && room.started_at" class="time">
                {{ formatTime(room.started_at) }}
              </span>
            </div>
          </div>
        </article>
      </div>

      <div v-if="loading && rooms.length > 0" class="loading-more">刷新中…</div>
    </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import LoadingState from '@/components/LoadingState.vue'
import { liveApi, type LiveRoom, type LiveGroupStat } from '@/api'

const loading = ref(true)
const rooms = ref<LiveRoom[]>([])
const groups = ref<LiveGroupStat[]>([])
const roomType = ref('')
const statusFilter = ref('')
const groupTitle = ref('')
const searchInput = ref('')
const searchQuery = ref('')
const groupScrollRef = ref<HTMLElement | null>(null)

let pollTimer: ReturnType<typeof setInterval> | null = null
let searchDebounce: ReturnType<typeof setTimeout> | null = null

const typeTabs = [
  { value: '', label: '全部' },
  { value: 'iptv', label: 'IPTV' },
  { value: 'push', label: '推流' },
]

const statusTabs = [
  { value: '', label: '全部' },
  { value: 'live', label: '直播中' },
  { value: 'idle', label: '待开播' },
]

const groupTabs = computed(() => {
  const tabs: { value: string; label: string; count: number }[] = [
    { value: '', label: '全部栏目', count: totalAll.value },
  ]
  for (const g of groups.value) {
    if (g.count <= 0) continue
    tabs.push({ value: g.name, label: g.name, count: g.count })
  }
  return tabs
})

const totalAll = computed(() => groups.value.reduce((s, g) => s + g.count, 0))

const hasFilters = computed(
  () => !!(roomType.value || statusFilter.value || groupTitle.value || searchQuery.value),
)

function coverStyle(room: LiveRoom) {
  return room.cover_url ? { backgroundImage: `url(${room.cover_url})` } : {}
}

function formatTime(iso: string) {
  return new Date(iso).toLocaleString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

function buildParams() {
  const params: Record<string, string | number> = { page_size: 500 }
  if (roomType.value) params.room_type = roomType.value
  if (statusFilter.value) params.status = statusFilter.value
  if (groupTitle.value) params.group_title = groupTitle.value
  if (searchQuery.value) params.search = searchQuery.value
  return params
}

async function loadGroups() {
  try {
    groups.value = await liveApi.groups()
  } catch {
    groups.value = []
  }
}

async function loadRooms() {
  loading.value = true
  try {
    rooms.value = await liveApi.list(buildParams())
  } catch (e) {
    console.error('加载直播间失败', e)
    rooms.value = []
  } finally {
    loading.value = false
  }
}

async function reload() {
  await Promise.all([loadGroups(), loadRooms()])
}

function setRoomType(v: string) {
  roomType.value = v
  loadRooms()
}

function setStatus(v: string) {
  statusFilter.value = v
  loadRooms()
}

function setGroup(v: string) {
  groupTitle.value = v
  loadRooms()
}

function applySearch() {
  searchQuery.value = searchInput.value.trim()
  loadRooms()
}

watch(searchInput, (val) => {
  if (searchDebounce) clearTimeout(searchDebounce)
  searchDebounce = setTimeout(() => {
    searchQuery.value = val.trim()
    loadRooms()
  }, 400)
})

onMounted(() => {
  reload()
  pollTimer = setInterval(loadRooms, 30000)
})

onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
  if (searchDebounce) clearTimeout(searchDebounce)
})
</script>

<style lang="scss" scoped>
.live-list-page {
  min-height: 100vh;
  background: var(--mh-bg);
  color: var(--mh-text);
}

.live-list-body {
  padding-top: var(--mh-topbar-height);
}

.live-topbar {
  gap: var(--mh-space-4);
}

.back-btn {
  background: none;
  border: none;
  color: var(--mh-text-secondary);
  cursor: pointer;
  font-size: 14px;
  flex-shrink: 0;

  &:hover { color: var(--mh-text); }
}

.page-title {
  font-size: 20px;
  font-weight: 700;
  margin: 0;
  font-family: var(--mh-font-display);
  flex-shrink: 0;
}

.topbar-search {
  flex: 1;
  max-width: 360px;
  margin-left: auto;

  input {
    width: 100%;
    height: 38px;
    padding: 0 14px;
    border-radius: 10px;
    border: 1px solid var(--mh-outline);
    background: rgba(255, 255, 255, 0.06);
    color: var(--mh-text);
    font-size: 14px;
    outline: none;

    &:focus {
      border-color: var(--mh-primary);
      box-shadow: 0 0 0 3px var(--mh-primary-muted);
    }
  }
}

.filter-bar {
  padding: var(--mh-space-4) var(--mh-page-gutter);
  background: rgba(10, 10, 18, 0.6);
  border-bottom: 1px solid var(--mh-outline);
  display: flex;
  flex-direction: column;
  gap: var(--mh-space-3);
}

.filter-row {
  display: flex;
  align-items: center;
  gap: var(--mh-space-3);
  min-width: 0;

  &--groups {
    align-items: flex-start;
  }
}

.filter-label {
  flex-shrink: 0;
  width: 36px;
  font-size: 12px;
  font-weight: 600;
  color: var(--mh-text-muted);
}

.pill-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.pill {
  padding: 6px 14px;
  border-radius: 999px;
  border: 1px solid transparent;
  background: rgba(255, 255, 255, 0.06);
  color: var(--mh-text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;

  &:hover {
    color: var(--mh-text);
    border-color: var(--mh-outline);
  }

  &.active {
    background: var(--mh-primary);
    color: #fff;
    border-color: transparent;
  }
}

.group-scroll {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  flex: 1;
  padding-bottom: 2px;
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.12) transparent;

  &::-webkit-scrollbar {
    height: 3px;
  }

  &::-webkit-scrollbar-track {
    background: transparent;
  }

  &::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.12);
    border-radius: 999px;
  }
}

.group-tab {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: 10px;
  border: 1px solid var(--mh-outline);
  background: var(--mh-surface);
  color: var(--mh-text-secondary);
  cursor: pointer;
  transition: all 0.15s;

  &:hover {
    color: var(--mh-text);
    border-color: rgba(255, 255, 255, 0.2);
  }

  &.active {
    background: linear-gradient(135deg, rgba(99, 102, 241, 0.25), rgba(168, 85, 247, 0.15));
    border-color: rgba(99, 102, 241, 0.5);
    color: var(--mh-text);
  }

  &__name {
    font-size: 13px;
    font-weight: 500;
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__count {
    font-size: 11px;
    padding: 1px 6px;
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.08);
    color: var(--mh-text-muted);
  }

  &.active &__count {
    background: rgba(99, 102, 241, 0.35);
    color: #fff;
  }
}

.content {
  padding: var(--mh-space-5) var(--mh-page-gutter) var(--mh-space-10);
  width: 100%;
}

.empty {
  text-align: center;
  padding: 80px 20px;
  color: var(--mh-text-secondary);

  .hint {
    font-size: 13px;
    margin-top: 8px;
    opacity: 0.7;
  }
}

.loading-more {
  text-align: center;
  padding: var(--mh-space-4);
  font-size: 13px;
  color: var(--mh-text-muted);
}

.room-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--mh-space-4);
  align-items: start;
}

.room-card {
  display: flex;
  flex-direction: column;
  background: var(--mh-surface);
  border-radius: var(--mh-radius-md);
  overflow: hidden;
  cursor: pointer;
  border: 1px solid var(--mh-outline);
  transition: transform 0.2s, box-shadow 0.2s, border-color 0.2s;

  &:hover {
    transform: translateY(-3px);
    box-shadow: var(--mh-shadow-md);
    border-color: rgba(99, 102, 241, 0.35);
  }
}

.room-cover {
  aspect-ratio: 16 / 9;
  background: linear-gradient(135deg, #1a1a2e, #16213e);
  background-size: cover;
  background-position: center;
  position: relative;
}

.live-badge {
  position: absolute;
  top: 10px;
  left: 10px;
  background: #e50914;
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 4px;
  letter-spacing: 0.06em;
  animation: pulse 1.5s infinite;
}

.iptv-badge {
  position: absolute;
  top: 10px;
  left: 10px;
  background: rgba(34, 197, 94, 0.9);
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 4px;
}

.idle-badge {
  position: absolute;
  top: 10px;
  left: 10px;
  background: rgba(255, 255, 255, 0.15);
  color: #fff;
  font-size: 10px;
  padding: 3px 8px;
  border-radius: 4px;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.75; }
}

.room-info {
  padding: var(--mh-space-4);

  h3 {
    margin: 0 0 8px;
    font-size: 15px;
    font-weight: 600;
    line-height: 1.35;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .meta {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    font-size: 12px;
    color: var(--mh-text-muted);
  }

  .group-tag {
    padding: 2px 8px;
    border-radius: 6px;
    background: rgba(99, 102, 241, 0.15);
    color: var(--mh-primary);
    font-weight: 500;
  }

  .time {
    opacity: 0.85;
  }
}
</style>
