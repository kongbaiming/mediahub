<template>
  <div class="albums-page">
    <div class="page-header">
      <div>
        <h2 class="page-h2">专题专辑</h2>
        <p class="page-desc">用于布局编辑器中的「专题专辑」数据源，可手动编排一组媒资。</p>
      </div>
      <el-button type="primary" @click="openCreate">
        <el-icon><Plus /></el-icon>
        新建专辑
      </el-button>
    </div>

    <el-table v-loading="loading" :data="albums" stripe>
      <el-table-column label="标题" min-width="220" prop="title" />
      <el-table-column label="简介" min-width="280" show-overflow-tooltip prop="overview" />
      <el-table-column label="作品数" width="100">
        <template #default="{ row }">{{ workCounts[row.id] ?? '—' }}</template>
      </el-table-column>
      <el-table-column label="操作" width="160" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row as Album)">编辑</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-if="!loading && albums.length === 0" description="暂无专题专辑">
      <el-button type="primary" @click="openCreate">创建第一个专辑</el-button>
    </el-empty>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑专辑' : '新建专辑'" width="640" destroy-on-close>
      <el-form label-position="top">
        <el-form-item label="标题" required>
          <el-input v-model="form.title" placeholder="例如：经典港片、周末 binge" />
        </el-form-item>
        <el-form-item label="简介">
          <el-input v-model="form.overview" type="textarea" :rows="2" placeholder="可选" />
        </el-form-item>
        <el-form-item label="包含作品">
          <el-select
            v-model="form.mediaIds"
            multiple
            filterable
            remote
            reserve-keyword
            :remote-method="searchMedia"
            :loading="mediaSearching"
            placeholder="搜索媒资标题…"
            style="width: 100%"
          >
            <el-option
              v-for="m in mediaOptions"
              :key="m.id"
              :label="`${m.title}${m.year ? ` (${m.year})` : ''}`"
              :value="m.id"
            />
          </el-select>
          <p class="form-hint">已选 {{ form.mediaIds.length }} 部作品，顺序即展示顺序。</p>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { catalogApi, type Album } from '@/api/catalog'
import { mediaApi } from '@/api/media'
import type { MediaSummary } from '@/api/types'

const loading = ref(false)
const saving = ref(false)
const albums = ref<Album[]>([])
const workCounts = ref<Record<string, number>>({})
const dialogVisible = ref(false)
const editingId = ref('')
const mediaSearching = ref(false)
const mediaOptions = ref<MediaSummary[]>([])

const form = reactive({
  title: '',
  overview: '',
  mediaIds: [] as string[],
})

async function load() {
  loading.value = true
  try {
    const res = await catalogApi.albums()
    albums.value = res.data || []
    workCounts.value = {}
    await Promise.all(
      albums.value.map(async (a) => {
        try {
          const w = await catalogApi.albumWorks(a.id)
          workCounts.value[a.id] = w.data?.length ?? 0
        } catch {
          workCounts.value[a.id] = 0
        }
      }),
    )
  } finally {
    loading.value = false
  }
}

async function searchMedia(q: string) {
  if (!q.trim()) return
  mediaSearching.value = true
  try {
    const res = await mediaApi.list({ q: q.trim(), page_size: 30 })
    const picked = new Map(mediaOptions.value.map((m) => [m.id, m]))
    for (const m of res.items || []) picked.set(m.id, m)
    mediaOptions.value = [...picked.values()]
  } finally {
    mediaSearching.value = false
  }
}

function openCreate() {
  editingId.value = ''
  form.title = ''
  form.overview = ''
  form.mediaIds = []
  mediaOptions.value = []
  dialogVisible.value = true
}

async function openEdit(album: Album) {
  editingId.value = album.id
  form.title = album.title
  form.overview = album.overview || ''
  form.mediaIds = []
  mediaOptions.value = []
  dialogVisible.value = true
  try {
    const w = await catalogApi.albumWorks(album.id)
    const items = w.data || []
    form.mediaIds = items.map((m) => m.id)
    mediaOptions.value = items
  } catch {
    ElMessage.warning('加载专辑作品失败')
  }
}

async function save() {
  if (!form.title.trim()) {
    ElMessage.warning('请输入标题')
    return
  }
  saving.value = true
  try {
    const payload = {
      title: form.title.trim(),
      overview: form.overview.trim(),
      media_ids: form.mediaIds,
    }
    if (editingId.value) {
      await catalogApi.updateAlbum(editingId.value, payload)
      ElMessage.success('已更新')
    } else {
      await catalogApi.createAlbum(payload)
      ElMessage.success('已创建')
    }
    dialogVisible.value = false
    await load()
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style lang="scss" scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: var(--mh-space-4);
  margin-bottom: var(--mh-space-5);
}

.page-h2 {
  margin: 0 0 var(--mh-space-1);
  font-size: 20px;
  font-weight: 600;
}

.page-desc {
  margin: 0;
  font-size: 13px;
  color: var(--mh-text-secondary);
}

.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--mh-text-muted);
}
</style>
