<template>
  <el-dialog
    v-model="visible"
    title="选择媒资"
    width="900px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div class="picker-layout">
      <!-- 左侧搜索 + 结果 -->
      <div class="picker-search">
        <el-input
          v-model="query"
          placeholder="搜索电影 / 剧集..."
          clearable
          @keyup.enter="doSearch"
        >
          <template #append>
            <el-button @click="doSearch">
              <el-icon><Search /></el-icon>
            </el-button>
          </template>
        </el-input>

        <div v-if="loading" class="loading-hint">搜索中...</div>

        <div v-else class="result-grid">
          <div
            v-for="item in results"
            :key="item.id"
            class="result-card"
            :class="{ selected: selectedMap[item.id] }"
            @click="toggleSelect(item)"
          >
            <div class="card-poster">
              <img v-if="item.poster_url" :src="item.poster_url" :alt="item.title" loading="lazy" />
              <span v-else class="poster-placeholder">{{ item.title.slice(0, 2) }}</span>
              <div v-if="selectedMap[item.id]" class="check-overlay">
                <el-icon :size="24"><Check /></el-icon>
              </div>
            </div>
            <div class="card-title">{{ item.title }}</div>
            <div class="card-meta">{{ item.year }} · {{ typeLabel(item.type) }}</div>
          </div>
        </div>

        <div v-if="!loading && !results.length && query" class="empty-hint">
          没有找到「{{ query }}」相关结果
        </div>
      </div>

      <!-- 右侧已选列表 -->
      <div class="picker-selected">
        <div class="selected-header">
          <span>已选 ({{ selectedItems.length }})</span>
          <el-button size="small" text type="danger" @click="clearAll" :disabled="!selectedItems.length">
            清空
          </el-button>
        </div>

        <div v-if="!selectedItems.length" class="empty-selected">
          点击左侧卡片添加
        </div>

        <div class="selected-list">
          <div
            v-for="(item, i) in selectedItems"
            :key="item.id"
            class="selected-item"
          >
            <span class="item-index">{{ i + 1 }}</span>
            <img v-if="item.poster_url" :src="item.poster_url" class="item-thumb" />
            <span class="item-title">{{ item.title }}</span>
            <div class="item-actions">
              <el-button
                size="small"
                text
                :disabled="i === 0"
                @click="moveUp(i)"
              >
                <el-icon><Top /></el-icon>
              </el-button>
              <el-button
                size="small"
                text
                :disabled="i === selectedItems.length - 1"
                @click="moveDown(i)"
              >
                <el-icon><Bottom /></el-icon>
              </el-button>
              <el-button
                size="small"
                text
                type="danger"
                @click="removeAt(i)"
              >
                <el-icon><Close /></el-icon>
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :disabled="!selectedItems.length" @click="confirmSelection">
        确认 ({{ selectedItems.length }})
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Search, Check, Top, Bottom, Close } from '@element-plus/icons-vue'
import { mediaApi, type MediaSummary } from '@/api/media'

const props = defineProps<{
  modelValue: boolean
  selectedIds?: string[]
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'confirm', ids: string[]): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const query = ref('')
const loading = ref(false)
const results = ref<MediaSummary[]>([])
const selectedItems = ref<MediaSummary[]>([])
const selectedMap = ref<Record<string, boolean>>({})

// 初始化已选
watch(() => props.modelValue, async (val) => {
  if (val && props.selectedIds?.length) {
    // 尝试加载已选媒资信息
    selectedItems.value = []
    selectedMap.value = {}
    for (const id of props.selectedIds) {
      try {
        const res = await mediaApi.get(id)
        const item: MediaSummary = {
          id: res.id,
          title: res.title,
          year: res.year,
          type: res.type,
          rating: res.rating,
          poster_url: res.poster_url,
          backdrop_url: res.backdrop_url,
          genres: res.genres || [],
        }
        selectedItems.value.push(item)
        selectedMap.value[id] = true
      } catch {
        // skip
      }
    }
  } else if (val) {
    selectedItems.value = []
    selectedMap.value = {}
  }
})

async function doSearch() {
  if (!query.value.trim()) {
    results.value = []
    return
  }
  loading.value = true
  try {
    const res = await mediaApi.list({ q: query.value, page_size: 20 })
    results.value = res.items || []
  } catch {
    results.value = []
  } finally {
    loading.value = false
  }
}

function toggleSelect(item: MediaSummary) {
  if (selectedMap.value[item.id]) {
    // 取消选中
    selectedMap.value[item.id] = false
    selectedItems.value = selectedItems.value.filter((i) => i.id !== item.id)
  } else {
    selectedMap.value[item.id] = true
    selectedItems.value.push(item)
  }
}

function removeAt(index: number) {
  const item = selectedItems.value[index]
  if (item) {
    selectedMap.value[item.id] = false
    selectedItems.value.splice(index, 1)
  }
}

function moveUp(index: number) {
  if (index <= 0) return
  const arr = selectedItems.value
  ;[arr[index - 1], arr[index]] = [arr[index], arr[index - 1]]
}

function moveDown(index: number) {
  const arr = selectedItems.value
  if (index >= arr.length - 1) return
  ;[arr[index], arr[index + 1]] = [arr[index + 1], arr[index]]
}

function clearAll() {
  selectedItems.value = []
  selectedMap.value = {}
}

function confirmSelection() {
  const ids = selectedItems.value.map((i) => i.id)
  emit('confirm', ids)
  visible.value = false
}

function handleClose() {
  visible.value = false
}

function typeLabel(t: string) {
  return ({ movie: '电影', tvshow: '剧集', anime: '动画', documentary: '纪录片' } as Record<string, string>)[t] || t
}
</script>

<style lang="scss" scoped>
.picker-layout {
  display: flex;
  gap: 16px;
  height: 500px;
}

.picker-search {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow: hidden;
}

.result-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
  gap: 10px;
  overflow-y: auto;
  flex: 1;
}

.result-card {
  cursor: pointer;
  border-radius: 8px;
  transition: transform 0.15s;

  &:hover { transform: translateY(-2px); }

  &.selected .card-poster {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 2px var(--el-color-primary-light-5);
  }
}

.card-poster {
  position: relative;
  aspect-ratio: 2/3;
  background: #f0f2f5;
  border-radius: 6px;
  overflow: hidden;
  border: 2px solid transparent;
  transition: border-color 0.15s, box-shadow 0.15s;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.poster-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  font-size: 20px;
  font-weight: 700;
  color: #c0c4cc;
}

.check-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(99, 102, 241, 0.4);
  color: #fff;
}

.card-title {
  margin-top: 4px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-meta {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.picker-selected {
  width: 280px;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
}

.selected-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  background: var(--el-fill-color-light);
  font-size: 13px;
  font-weight: 600;
}

.selected-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.empty-selected {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

.selected-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 6px;
  transition: background 0.15s;

  &:hover { background: var(--el-fill-color-lighter); }
}

.item-index {
  width: 20px;
  text-align: center;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-weight: 600;
}

.item-thumb {
  width: 28px;
  height: 40px;
  object-fit: cover;
  border-radius: 3px;
}

.item-title {
  flex: 1;
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-actions {
  display: flex;
  gap: 0;
}

.loading-hint, .empty-hint {
  text-align: center;
  padding: 40px 0;
  color: var(--el-text-color-secondary);
  font-size: 14px;
}
</style>
