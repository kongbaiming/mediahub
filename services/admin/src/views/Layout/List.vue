<template>
  <div class="layout-list-page">
    <div class="page-header">
      <h2 class="page-h2">布局列表</h2>
      <el-button @click="showCreatePreset = true">
        从模版创建
      </el-button>
      <el-button type="primary" @click="onCreate">
        <el-icon><Plus /></el-icon>
        新建布局
      </el-button>
    </div>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        class="layout-list-tip"
        title="模版说明"
        description="支持榜单行（TOP 排行）、专题行（专辑 + 沉浸式头图）、沉浸式页面模版。可「从模版创建」或在编辑器右侧配置页面模版。"
      />

      <el-tabs v-model="activeTab" @tab-change="reload">
      <el-tab-pane label="全部" name="all" />
      <el-tab-pane label="已发布" name="published" />
      <el-tab-pane label="草稿" name="draft" />
      <el-tab-pane label="模板" name="template" />
    </el-tabs>

    <el-table v-loading="loading" :data="items" stripe @row-click="onEdit">
      <el-table-column label="名称" prop="name" min-width="200">
        <template #default="{ row }">
          <el-link type="primary" :underline="false">{{ row.name }}</el-link>
          <el-tag v-if="row.is_template" size="small" type="info" class="ml-2">模板</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="描述" prop="description" min-width="200" show-overflow-tooltip />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="版本" prop="version" width="80" align="center" />
      <el-table-column label="行数" width="80" align="center">
        <template #default="{ row }">{{ row.config?.rows?.length || 0 }}</template>
      </el-table-column>
      <el-table-column label="最近发布" width="180">
        <template #default="{ row }">
          {{ row.last_published_at ? formatDate(row.last_published_at) : '-' }}
        </template>
      </el-table-column>
      <el-table-column label="更新时间" width="180">
        <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="280" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click.stop="onEdit(row as Layout)">编辑</el-button>
          <el-button size="small" type="success" @click.stop="onPublish(row as Layout)">发布</el-button>
          <el-button size="small" type="danger" @click.stop="onDelete(row as Layout)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="showCreatePreset" title="从模版创建布局" width="480">
      <el-form label-position="top">
        <el-form-item label="布局名称">
          <el-input v-model="presetLayoutName" placeholder="如：沉浸式首页" />
        </el-form-item>
        <el-form-item label="选择模版">
          <el-radio-group v-model="selectedPresetKey" class="preset-radio-group">
            <el-radio
              v-for="p in LAYOUT_PRESETS"
              :key="p.name"
              :value="p.name"
              class="preset-radio"
            >
              <strong>{{ p.name }}</strong>
              <span class="preset-desc">{{ p.description }}</span>
            </el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreatePreset = false">取消</el-button>
        <el-button type="primary" @click="onCreateFromPreset">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { layoutApi } from '@/api/layout'
import type { Layout, LayoutRow } from '@/api/types'
import { LAYOUT_PRESETS } from '@/utils/layoutPresets'

const router = useRouter()
const loading = ref(false)
const activeTab = ref('all')
const items = ref<Layout[]>([])
const showCreatePreset = ref(false)
const presetLayoutName = ref('')
const selectedPresetKey = ref(LAYOUT_PRESETS[0]?.name || '')

async function reload() {
  loading.value = true
  try {
    const params: any = {}
    if (activeTab.value === 'published') params.status = 'published'
    if (activeTab.value === 'draft') params.status = 'draft'
    if (activeTab.value === 'template') params.is_template = true
    const res = await layoutApi.list(params)
    items.value = res.data
  } finally {
    loading.value = false
  }
}

async function onCreateFromPreset() {
  const name = presetLayoutName.value.trim()
  if (!name) {
    ElMessage.warning('请输入布局名称')
    return
  }
  const preset = LAYOUT_PRESETS.find((p) => p.name === selectedPresetKey.value)
  if (!preset) return
  try {
    const res = await layoutApi.create({
      name,
      description: preset.description,
      config: {
        theme: preset.theme as 'dark' | 'light',
        global: preset.global,
        rows: preset.rows as LayoutRow[],
      },
    })
    ElMessage.success('创建成功')
    showCreatePreset.value = false
    presetLayoutName.value = ''
    router.push(`/layouts/${(res as any).data.id}`)
  } catch {
    // handled by interceptor
  }
}

async function onCreate() {
  try {
    const { value } = await ElMessageBox.prompt('布局名称', '新建布局', {
      inputPlaceholder: '如：家庭主版',
      inputValidator: (v) => (v && v.length > 0 ? true : '名称不能为空'),
    })
    const res = await layoutApi.create({
      name: value,
      config: { theme: 'dark', rows: [] },
    })
    ElMessage.success('创建成功')
    router.push(`/layouts/${(res as any).data.id}`)
  } catch {
    // user cancelled
  }
}

function onEdit(row: Layout) {
  router.push(`/layouts/${row.id}`)
}

async function onPublish(row: Layout) {
  try {
    const { value } = await ElMessageBox.prompt('选择发布平台', '发布布局', {
      inputValue: 'web',
      inputValidator: (v) =>
        ['web', 'android-tv', 'tvos'].includes(v) ? true : '必须是 web / android-tv / tvos',
    })
    await layoutApi.publish(row.id, { target_platform: value as any })
    ElMessage.success('发布成功')
    reload()
  } catch {
    // cancelled
  }
}

async function onDelete(row: Layout) {
  try {
    await ElMessageBox.confirm(`确认删除布局「${row.name}」？`, '删除确认', { type: 'warning' })
    await layoutApi.delete(row.id)
    ElMessage.success('已删除')
    reload()
  } catch {
    // cancelled
  }
}

function statusType(s: string): any {
  return ({ published: 'success', draft: 'info', archived: 'warning' } as any)[s] || ''
}
function statusLabel(s: string) {
  return ({ published: '已发布', draft: '草稿', archived: '归档' } as any)[s] || s
}
function formatDate(s: string) {
  if (!s) return '-'
  return new Date(s).toLocaleString('zh-CN')
}

onMounted(reload)
</script>

<style lang="scss" scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.page-h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  color: #1e293b;
}

.layout-list-tip {
  margin-bottom: 16px;
}

.ml-2 {
  margin-left: 8px;
}

.preset-radio-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}

.preset-radio {
  display: flex;
  align-items: flex-start;
  height: auto;
  margin-right: 0;
  white-space: normal;

  .preset-desc {
    display: block;
    font-size: 12px;
    color: #64748b;
    font-weight: 400;
    margin-top: 4px;
  }
}
</style>
