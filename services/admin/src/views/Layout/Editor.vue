<template>
  <div class="layout-editor" v-loading="loading">
    <div class="editor-header">
      <div>
        <el-button text @click="$router.back()">
          <el-icon><ArrowLeft /></el-icon>
          返回
        </el-button>
        <span class="title">{{ layout?.name || '加载中...' }}</span>
        <el-tag v-if="layout" :type="statusType(layout.status)" size="small" class="ml-2">
          {{ statusLabel(layout.status) }}
        </el-tag>
        <span v-if="layout" class="version">v{{ layout.version }}</span>
        <span v-if="layout?.parent_id" class="inherit-badge">
          <el-icon><Connection /></el-icon>
          继承自父布局
        </span>
      </div>
      <div class="actions">
        <el-button @click="showInherit = true">
          <el-icon><Connection /></el-icon>
          模板继承
        </el-button>
        <el-button @click="showPublications = true">
          <el-icon><List /></el-icon>
          发布历史
        </el-button>
        <el-button @click="showPublish = true" type="success">
          <el-icon><Promotion /></el-icon>
          发布
        </el-button>
        <el-button @click="onSave" type="primary" :loading="saving">
          <el-icon><Document /></el-icon>
          保存
        </el-button>
      </div>
    </div>

    <div class="editor-body">
      <aside class="component-panel">
        <div class="panel-title">组件</div>
        <VueDraggable
          :model-value="paletteItems"
          :group="{ name: 'layout', pull: 'clone', put: false }"
          :clone="clonePaletteItem"
          :sort="false"
          class="palette"
        >
          <div
            v-for="element in paletteItems"
            :key="element.type"
            class="palette-item"
          >
            <el-icon><component :is="element.icon" /></el-icon>
            <span>{{ element.label }}</span>
          </div>
        </VueDraggable>

        <div class="panel-title mt-16">数据源</div>
        <el-select v-model="selectedDataSource" placeholder="选择数据源类型" size="small">
          <el-option v-for="d in dataSourceTypes" :key="d.value" :label="d.label" :value="d.value" />
        </el-select>
        <el-button size="small" class="mt-8 w-full" @click="addDataSourceRow">
          添加数据源行
        </el-button>

        <div class="panel-title mt-16">视图</div>
        <el-radio-group v-model="previewPlatform" size="small" class="w-full">
          <el-radio-button label="web">Web</el-radio-button>
          <el-radio-button label="android-tv">Android TV</el-radio-button>
          <el-radio-button label="tvos">tvOS</el-radio-button>
        </el-radio-group>
      </aside>

      <main class="canvas-area">
        <div class="canvas-frame" :style="{ transform: `scale(${zoom})` }">
          <div class="canvas-inner" :class="`platform-${previewPlatform}`">
            <VueDraggable
              v-model="layoutRows"
              :group="{ name: 'layout' }"
              handle=".row-handle"
              :animation="200"
              class="rows-container"
              @end="onRowOrderChange"
            >
              <div
                v-for="(element, index) in layoutRows"
                :key="element.id"
                class="layout-row"
                :class="{
                  active: selectedRowIndex === index,
                  inherited: element._inherited,
                }"
                @click="selectedRowIndex = index"
              >
                <div class="row-handle">
                  <el-icon><Rank /></el-icon>
                </div>
                <div class="row-content">
                  <div class="row-header">
                    <el-tag size="small" type="primary">{{ rowTypeLabel(element.type) }}</el-tag>
                    <el-tag v-if="element._inherited" size="small" type="info">继承</el-tag>
                    <span class="row-title">{{ element.title || '(未命名)' }}</span>
                    <span v-if="previewItems(element.id).length" class="row-meta">
                      {{ previewItems(element.id).length }} 项
                    </span>
                    <span v-else class="row-meta row-meta--muted">
                      {{ dataSourceLabel(element.source?.type) }}
                    </span>
                    <div class="row-actions">
                      <el-switch v-model="element.visible" size="small" :disabled="element._inherited" />
                      <el-button v-if="!element._inherited" text size="small" @click.stop="removeRow(index)">
                        <el-icon><Delete /></el-icon>
                      </el-button>
                    </div>
                  </div>
                  <div class="row-preview" :class="`card-style-${element.card_style || 'poster'}`">
                    <template v-if="previewItems(element.id).length">
                      <div
                        v-for="item in previewItems(element.id).slice(0, 8)"
                        :key="item.media_id"
                        class="preview-card preview-card--real"
                        :title="item.title"
                      >
                        <div class="preview-card-media">
                          <img v-if="item.poster_url" :src="item.poster_url" :alt="item.title" loading="lazy" />
                          <span v-else class="preview-card-label">{{ item.title?.slice(0, 6) }}</span>
                        </div>
                        <div class="preview-card-caption">{{ item.title }}</div>
                      </div>
                    </template>
                    <template v-else>
                      <div v-for="i in 5" :key="i" class="preview-card preview-card--placeholder"></div>
                    </template>
                  </div>
                </div>
              </div>
            </VueDraggable>

            <div v-if="!layoutRows.length" class="empty-canvas">
              <el-empty description="从左侧拖入组件开始编辑" />
            </div>
          </div>
        </div>

        <div class="canvas-toolbar">
          <span>预览：</span>
          <el-radio-group v-model="zoom" size="small">
            <el-radio-button :value="0.4">40%</el-radio-button>
            <el-radio-button :value="0.6">60%</el-radio-button>
            <el-radio-button :value="0.8">80%</el-radio-button>
            <el-radio-button :value="1">100%</el-radio-button>
          </el-radio-group>
          <span class="ml-16">平台：{{ platformLabel(previewPlatform) }}</span>
        </div>
      </main>

      <aside class="config-panel">
        <div class="panel-title">属性</div>
        <div v-if="selectedRow" class="config-form">
          <el-form label-position="top" size="small">
            <el-form-item label="类型">
              <el-select v-model="selectedRow.type" @change="onRowTypeChange" :disabled="selectedRow._inherited">
                <el-option v-for="t in rowTypes" :key="t.value" :label="t.label" :value="t.value" />
              </el-select>
            </el-form-item>
            <el-form-item label="标题">
              <el-input v-model="selectedRow.title" :disabled="selectedRow._inherited" />
            </el-form-item>
            <el-form-item label="副标题">
              <el-input v-model="selectedRow.subtitle" :disabled="selectedRow._inherited" />
            </el-form-item>
            <el-form-item label="卡片样式">
              <el-radio-group v-model="selectedRow.card_style" :disabled="selectedRow._inherited">
                <el-radio-button label="poster">海报</el-radio-button>
                <el-radio-button label="landscape">横版</el-radio-button>
                <el-radio-button label="square">方版</el-radio-button>
                <el-radio-button label="banner">Banner</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="数据源类型">
              <el-select
                v-model="selectedRow.source.type"
                :disabled="selectedRow._inherited"
                @change="onSourceTypeChange"
              >
                <el-option v-for="d in dataSourceTypes" :key="d.value" :label="d.label" :value="d.value" />
              </el-select>
            </el-form-item>

            <template v-if="selectedRow.source.type === 'album'">
              <el-form-item label="专题专辑">
                <el-select
                  v-model="selectedRow.source.params!.album_id"
                  filterable
                  placeholder="选择专辑"
                  :disabled="selectedRow._inherited"
                  @change="syncParamsJson"
                >
                  <el-option v-for="a in albums" :key="a.id" :label="a.title" :value="a.id" />
                </el-select>
              </el-form-item>
            </template>

            <template v-else-if="selectedRow.source.type === 'category'">
              <el-form-item label="分类">
                <el-select
                  v-model="selectedRow.source.params!.slug"
                  filterable
                  placeholder="选择分类"
                  :disabled="selectedRow._inherited"
                  @change="syncParamsJson"
                >
                  <el-option
                    v-for="c in categories"
                    :key="c.id"
                    :label="c.name"
                    :value="c.slug"
                  />
                </el-select>
              </el-form-item>
            </template>

            <template v-else-if="selectedRow.source.type === 'tag'">
              <el-form-item label="标签名">
                <el-input
                  v-model="selectedRow.source.params!.tag"
                  placeholder="如：4K"
                  :disabled="selectedRow._inherited"
                  @change="syncParamsJson"
                />
              </el-form-item>
            </template>

            <template v-else-if="selectedRow.source.type === 'similar-to'">
              <el-form-item label="参考媒资 ID">
                <el-input
                  v-model="selectedRow.source.params!.media_id"
                  :disabled="selectedRow._inherited"
                  @change="syncParamsJson"
                />
              </el-form-item>
            </template>

            <template v-else-if="selectedRow.source.type === 'library'">
              <el-form-item label="媒资类型">
                <el-select
                  v-model="selectedRow.source.params!.type"
                  clearable
                  :disabled="selectedRow._inherited"
                  @change="syncParamsJson"
                >
                  <el-option label="电影" value="movie" />
                  <el-option label="剧集" value="tvshow" />
                  <el-option label="动画" value="anime" />
                </el-select>
              </el-form-item>
              <el-form-item label="流派">
                <el-input
                  v-model="selectedRow.source.params!.genre"
                  :disabled="selectedRow._inherited"
                  @change="syncParamsJson"
                />
              </el-form-item>
            </template>

            <el-form-item
              v-if="showLimitField(selectedRow.source.type)"
              label="数量上限"
            >
              <el-input-number
                v-model="selectedRow.source.params!.limit"
                :min="1"
                :max="100"
                :disabled="selectedRow._inherited"
                @change="syncParamsJson"
              />
            </el-form-item>

            <el-form-item label="数据源参数 (JSON)">
              <el-input
                v-model="dataSourceParamsStr"
                type="textarea"
                :rows="6"
                :disabled="selectedRow._inherited"
                @blur="onDataSourceParamsChange"
              />
            </el-form-item>
          </el-form>
        </div>
        <el-empty v-else description="选中一行查看属性" :image-size="80" />
      </aside>
    </div>

    <!-- 发布对话框 -->
    <el-dialog v-model="showPublish" title="发布布局" width="560">
      <el-form label-position="top">
        <el-form-item label="目标平台">
          <el-radio-group v-model="publishPlatform">
            <el-radio-button label="web">Web</el-radio-button>
            <el-radio-button label="android-tv">Android TV</el-radio-button>
            <el-radio-button label="tvos">tvOS</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="AB 流量分配（可选）">
          <el-input v-model="trafficSplitStr" type="textarea" :rows="3" placeholder='{"A":50,"B":50}  留空表示平均分配' />
          <div class="hint">AB 测试：同一布局可发布多个版本，按权重分流</div>
        </el-form-item>

        <el-divider>动态规则（可选）</el-divider>

        <el-form-item label="时段（小时）">
          <el-slider
            v-model="hourRange"
            range
            :min="0"
            :max="23"
            :step="1"
            :marks="hourMarks"
            show-stops
          />
        </el-form-item>

        <el-form-item label="星期">
          <el-checkbox-group v-model="dayOfWeek">
            <el-checkbox :value="0">日</el-checkbox>
            <el-checkbox :value="1">一</el-checkbox>
            <el-checkbox :value="2">二</el-checkbox>
            <el-checkbox :value="3">三</el-checkbox>
            <el-checkbox :value="4">四</el-checkbox>
            <el-checkbox :value="5">五</el-checkbox>
            <el-checkbox :value="6">六</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showPublish = false">取消</el-button>
        <el-button type="primary" @click="doPublish" :loading="publishing">发布</el-button>
      </template>
    </el-dialog>

    <!-- 模板继承对话框 -->
    <el-dialog v-model="showInherit" title="模板继承" width="500">
      <el-form label-position="top">
        <el-form-item label="继承自父布局（留空 = 不继承）">
          <el-select v-model="parentLayoutId" clearable placeholder="选择父布局" filterable>
            <el-option
              v-for="l in templateLayouts"
              :key="l.id"
              :label="l.name"
              :value="l.id"
            >
              <span>{{ l.name }}</span>
              <el-tag size="small" type="info" class="ml-2">v{{ l.version }}</el-tag>
            </el-option>
          </el-select>
          <div class="hint">
            父布局的 rows 会被继承，本布局新增/覆盖的 row 以本布局为准
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showInherit = false">取消</el-button>
        <el-button type="primary" @click="saveInherit">保存</el-button>
      </template>
    </el-dialog>

    <!-- 发布历史对话框 -->
    <el-dialog v-model="showPublications" title="发布历史" width="700">
      <el-table :data="publications" v-loading="loadingPubs" stripe size="small">
        <el-table-column label="平台" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ platformLabel(row.target_platform) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="版本" prop="version" width="60" align="center" />
        <el-table-column label="AB 权重" width="100">
          <template #default="{ row }">
            <span v-if="row.traffic_split">
              {{ Object.values(row.traffic_split).join(' / ') }}
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="动态规则" min-width="180">
          <template #default="{ row }">
            <span v-if="row.dynamic_rules?.hour_of_day" class="rule-tag">
              时段 {{ row.dynamic_rules.hour_of_day.start }}-{{ row.dynamic_rules.hour_of_day.end }}
            </span>
            <span v-if="row.dynamic_rules?.day_of_week" class="rule-tag">
              星期 {{ row.dynamic_rules.day_of_week.join(',') }}
            </span>
            <span v-if="!row.dynamic_rules?.hour_of_day && !row.dynamic_rules?.day_of_week">-</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
              {{ row.enabled ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="160">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.enabled"
              size="small"
              type="danger"
              @click="onDisablePub(row)"
            >禁用</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { VueDraggable } from 'vue-draggable-plus'
import { layoutApi, type Layout, type LayoutRow, type Publication, type DynamicRules, type FeedRow } from '@/api/layout'
import { catalogApi, type Album, type Category } from '@/api/catalog'
import type { Layout as LayoutType } from '@/api/types'

const route = useRoute()
const loading = ref(false)
const saving = ref(false)
const publishing = ref(false)
const loadingPubs = ref(false)

const showPublish = ref(false)
const showInherit = ref(false)
const showPublications = ref(false)

const layout = ref<Layout | null>(null)
const layoutRows = ref<LayoutRow[]>([])
const previewMap = ref<Record<string, FeedRow>>({})
const templateLayouts = ref<LayoutType[]>([])
const publications = ref<Publication[]>([])
const albums = ref<Album[]>([])
const categories = ref<Category[]>([])

const selectedRowIndex = ref(-1)
const selectedDataSource = ref('trending')
const zoom = ref(0.6)
const previewPlatform = ref<'web' | 'android-tv' | 'tvos'>('web')
const dataSourceParamsStr = ref('{}')

// 发布对话框
const publishPlatform = ref<'web' | 'android-tv' | 'tvos'>('web')
const trafficSplitStr = ref('')
const hourRange = ref<[number, number]>([0, 23])
const dayOfWeek = ref<number[]>([])

// 模板继承
const parentLayoutId = ref<string>('')

const hourMarks = {
  0: '0',
  6: '6',
  12: '12',
  18: '18',
  23: '23',
} as any

const selectedRow = computed(() => layoutRows.value[selectedRowIndex.value])

const paletteItems = [
  { type: 'hero-banner', label: 'Hero Banner', icon: 'PictureFilled' },
  { type: 'shelf', label: '横滑 Shelf', icon: 'Operation' },
  { type: 'category-grid', label: '分类网格', icon: 'Grid' },
  { type: 'topic', label: '专题', icon: 'CollectionTag' },
  { type: 'text-banner', label: '文字公告', icon: 'Notification' },
  { type: 'divider', label: '分隔线', icon: 'Minus' },
]

const rowTypes = [
  { value: 'hero-banner', label: 'Hero Banner' },
  { value: 'shelf', label: '横滑 Shelf' },
  { value: 'category-grid', label: '分类网格' },
  { value: 'topic', label: '专题' },
  { value: 'text-banner', label: '文字公告' },
  { value: 'divider', label: '分隔线' },
]

const dataSourceTypes = [
  { value: 'manual', label: '手动选片' },
  { value: 'library', label: '库筛选' },
  { value: 'trending', label: '热门榜单' },
  { value: 'continue-watching', label: '继续观看' },
  { value: 'recently-added', label: '最近添加' },
  { value: 'similar-to', label: '同类推荐' },
  { value: 'recommend-algorithm', label: '推荐算法' },
  { value: 'guess-you-like', label: '猜你喜欢' },
  { value: 'album', label: '专题专辑' },
  { value: 'category', label: '按分类' },
  { value: 'tag', label: '按标签' },
  { value: 'union', label: '并集' },
]

function defaultSourceParams(type: string): Record<string, any> {
  switch (type) {
    case 'album':
      return { album_id: '', limit: 20 }
    case 'category':
      return { slug: '', limit: 20 }
    case 'tag':
      return { tag: '', limit: 20 }
    case 'library':
      return { type: '', genre: '', limit: 20 }
    case 'similar-to':
      return { media_id: '', limit: 20 }
    case 'manual':
      return { ids: [] }
    case 'recommend-algorithm':
      return { algo: 'hybrid', limit: 20 }
    default:
      return { limit: 20 }
  }
}

function ensureRowParams(row: LayoutRow) {
  if (!row.source.params) {
    row.source.params = defaultSourceParams(row.source.type)
  }
}

function showLimitField(type: string) {
  return !['manual', 'union', 'continue-watching'].includes(type)
}

function onSourceTypeChange() {
  if (!selectedRow.value) return
  selectedRow.value.source.params = defaultSourceParams(selectedRow.value.source.type)
  syncParamsJson()
}

function syncParamsJson() {
  if (!selectedRow.value) return
  dataSourceParamsStr.value = JSON.stringify(selectedRow.value.source.params || {}, null, 2)
  loadPreview()
}

async function loadCatalogOptions() {
  try {
    const [a, c] = await Promise.all([
      catalogApi.albums(),
      catalogApi.categories('genre'),
    ])
    albums.value = a.data || []
    categories.value = c.data || []
  } catch {
    albums.value = []
    categories.value = []
  }
}

function clonePaletteItem(item: any) {
  return {
    id: `row-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
    type: item.type,
    title: '',
    card_style: 'poster',
    visible: true,
    source: { type: 'trending', params: {} },
  }
}

function addDataSourceRow() {
  const row = {
    id: `row-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
    type: 'shelf' as const,
    title: '新行',
    card_style: 'poster' as const,
    visible: true,
    source: { type: selectedDataSource.value, params: {} },
  }
  layoutRows.value.push(row)
  selectedRowIndex.value = layoutRows.value.length - 1
}

function removeRow(idx: number) {
  layoutRows.value.splice(idx, 1)
  if (selectedRowIndex.value >= layoutRows.value.length) {
    selectedRowIndex.value = layoutRows.value.length - 1
  }
}

function onRowOrderChange() { /* 由 v-model 自动更新 */ }

function onRowTypeChange() {
  if (selectedRow.value?.type === 'hero-banner') {
    selectedRow.value.card_style = 'banner'
  }
}

function onDataSourceParamsChange() {
  if (!selectedRow.value) return
  try {
    const parsed = JSON.parse(dataSourceParamsStr.value || '{}')
    selectedRow.value.source.params = parsed
    loadPreview()
  } catch {
    ElMessage.warning('JSON 格式错误')
  }
}

function previewItems(rowId: string) {
  return previewMap.value[rowId]?.items || []
}

function rowsForSave(): LayoutRow[] {
  return layoutRows.value
    .filter((r) => !r._inherited)
    .map(({ _inherited, ...r }) => r)
}

function dataSourceLabel(type?: string) {
  return dataSourceTypes.find((d) => d.value === type)?.label || type || '数据源'
}

function fillPreviewMap(rows: FeedRow[]) {
  const map: Record<string, FeedRow> = {}
  for (const row of rows) {
    map[row.id] = row
  }
  previewMap.value = map
}

async function loadPreview() {
  if (!layout.value) return
  try {
    const res = await layoutApi.preview(layout.value.id, previewPlatform.value)
    fillPreviewMap(res.data?.rows || [])
    return
  } catch (e) {
    console.warn('布局预览 API 失败，尝试已发布 Feed', e)
  }

  if (layout.value.status === 'published') {
    try {
      const res = await layoutApi.getPublishedFeed(previewPlatform.value)
      fillPreviewMap(res.data?.rows || [])
      return
    } catch (e) {
      console.warn('公开 Feed 回退失败', e)
    }
  }
  previewMap.value = {}
}

async function detectPreviewPlatform() {
  if (!layout.value) return
  try {
    const pubs = await layoutApi.listPublications(layout.value.id)
    const enabled = pubs.data?.find((p) => p.enabled)
    if (enabled?.target_platform) {
      previewPlatform.value = enabled.target_platform
      return
    }
  } catch {
    // ignore
  }
  if (layout.value.name.includes('TV')) {
    previewPlatform.value = 'android-tv'
  } else if (layout.value.name.includes('Web')) {
    previewPlatform.value = 'web'
  }
}

function unwrapApiData<T>(res: unknown): T {
  if (!res || typeof res !== 'object') return res as T
  const obj = res as Record<string, unknown>
  const inner = obj.data
  if (inner && typeof inner === 'object' && 'id' in (inner as object)) {
    return inner as T
  }
  if ('id' in obj) return res as T
  return (inner ?? res) as T
}

async function load() {
  loading.value = true
  try {
    const [l, t] = await Promise.all([
      layoutApi.get(route.params.id as string, { editor: true }),
      layoutApi.list({ is_template: true }),
    ])
    const layoutData = unwrapApiData<Layout>(l)
    layout.value = layoutData
    parentLayoutId.value = layoutData.parent_id || ''
    layoutRows.value = (layoutData.config?.rows || []).map((r) => ({
      ...r,
      visible: r.visible !== false,
      source: r.source || { type: 'trending', params: {} },
    }))
    templateLayouts.value = (t as { data?: LayoutType[] }).data ?? []
    await detectPreviewPlatform()
    await loadPreview()
  } finally {
    loading.value = false
  }
}

async function onSave() {
  if (!layout.value) return
  saving.value = true
  try {
    const updated = await layoutApi.update(layout.value.id, {
      config: { ...layout.value.config, rows: rowsForSave() },
    } as any)
    layout.value = (updated as any).data
    ElMessage.success('保存成功')
    await load()
  } finally {
    saving.value = false
  }
}

async function saveInherit() {
  if (!layout.value) return
  try {
    const updated = await layoutApi.update(layout.value.id, {
      parent_id: parentLayoutId.value || null,
    } as any)
    layout.value = (updated as any).data
    showInherit.value = false
    ElMessage.success('继承设置已保存')
    await load()
  } catch (e) {
    // toast handled by interceptor
  }
}

async function doPublish() {
  if (!layout.value) return
  publishing.value = true
  try {
    let split: any = undefined
    if (trafficSplitStr.value.trim()) {
      try {
        split = JSON.parse(trafficSplitStr.value)
      } catch {
        ElMessage.warning('AB 流量 JSON 格式错误')
        return
      }
    }

    // 动态规则
    const dynamicRules: DynamicRules = {}
    const [start, end] = hourRange.value
    if (start !== 0 || end !== 23) {
      dynamicRules.hour_of_day = { start, end }
    }
    if (dayOfWeek.value.length > 0) {
      dynamicRules.day_of_week = [...dayOfWeek.value]
    }

    await layoutApi.publish(layout.value.id, {
      target_platform: publishPlatform.value,
      traffic_split: split,
      dynamic_rules: Object.keys(dynamicRules).length > 0 ? dynamicRules : undefined,
    })
    ElMessage.success('发布成功')
    showPublish.value = false
    await load()
  } finally {
    publishing.value = false
  }
}

async function loadPublications() {
  if (!layout.value) return
  loadingPubs.value = true
  try {
    const res = await layoutApi.listPublications(layout.value.id)
    publications.value = res.data
  } finally {
    loadingPubs.value = false
  }
}

async function onDisablePub(pub: Publication) {
  try {
    await ElMessageBox.confirm('确认禁用此发布？', '禁用确认', { type: 'warning' })
    await layoutApi.disablePublication(pub.id)
    ElMessage.success('已禁用')
    loadPublications()
  } catch {
    // cancelled
  }
}

function rowTypeLabel(t: string) {
  return rowTypes.find((r) => r.value === t)?.label || t
}
function statusType(s: string): any {
  return ({ published: 'success', draft: 'info', archived: 'warning' } as any)[s] || ''
}
function statusLabel(s: string) {
  return ({ published: '已发布', draft: '草稿', archived: '归档' } as any)[s] || s
}
function platformLabel(p: string) {
  return ({ web: 'Web', 'android-tv': 'Android TV', tvos: 'tvOS' } as any)[p] || p
}
function formatDate(s: string) {
  if (!s) return '-'
  return new Date(s).toLocaleString('zh-CN')
}

watch(previewPlatform, loadPreview)

watch(showPublications, (val) => {
  if (val) loadPublications()
})

watch(selectedRow, (val) => {
  if (val) {
    ensureRowParams(val)
    dataSourceParamsStr.value = JSON.stringify(val.source.params || {}, null, 2)
  }
}, { immediate: false })

onMounted(async () => {
  await loadCatalogOptions()
  await load()
})
</script>

<style lang="scss" scoped>
.layout-editor {
  height: calc(100vh - 48px);
  display: flex;
  flex-direction: column;
  background: #f8fafc;
  margin: -24px;
}

.editor-header {
  height: 56px;
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;

  .title {
    font-size: 16px;
    font-weight: 600;
    margin-left: 8px;
  }
  .version {
    margin-left: 12px;
    color: #94a3b8;
    font-size: 13px;
  }
}

.inherit-badge {
  margin-left: 12px;
  color: #6366f1;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.actions {
  display: flex;
  gap: 8px;
}

.editor-body {
  flex: 1;
  display: grid;
  grid-template-columns: 240px 1fr 320px;
  overflow: hidden;
}

.component-panel,
.config-panel {
  background: #fff;
  border-right: 1px solid #e2e8f0;
  padding: 16px;
  overflow-y: auto;
}

.config-panel {
  border-right: none;
  border-left: 1px solid #e2e8f0;
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 12px;
}

.mt-8 { margin-top: 8px; }
.mt-16 { margin-top: 16px; }
.w-full { width: 100%; }
.ml-16 { margin-left: 16px; }
.ml-2 { margin-left: 8px; }

.hint {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 4px;
  line-height: 1.5;
}

.rule-tag {
  display: inline-block;
  padding: 2px 6px;
  background: #f1f5f9;
  border-radius: 3px;
  font-size: 11px;
  color: #475569;
  margin-right: 4px;
}

.palette {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.palette-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f1f5f9;
  border-radius: 6px;
  cursor: grab;
  font-size: 13px;
  transition: all 0.15s;

  &:hover {
    background: #e2e8f0;
    transform: translateX(2px);
  }

  &:active {
    cursor: grabbing;
  }
}

.canvas-area {
  background: #0f172a;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.canvas-frame {
  flex: 1;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 32px;
  overflow: auto;
  transform-origin: top center;
}

.canvas-inner {
  width: 1280px;
  background: linear-gradient(180deg, #1e293b, #0f172a);
  border-radius: 12px;
  padding: 24px;
  min-height: 600px;
  color: #fff;
  transition: all 0.3s;

  &.platform-android-tv {
    width: 1920px;
    min-height: 1080px;
    padding: 48px;
  }
  &.platform-tvos {
    width: 1920px;
    min-height: 1080px;
    padding: 60px;
  }
}

.rows-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.layout-row {
  display: flex;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  transition: all 0.15s;
  overflow: hidden;

  &.active {
    border-color: #6366f1;
    background: rgba(99, 102, 241, 0.1);
  }

  &.inherited {
    border-style: dashed;
    background: rgba(99, 102, 241, 0.05);
  }

  &:hover {
    border-color: rgba(99, 102, 241, 0.5);
  }
}

.row-handle {
  width: 32px;
  background: rgba(0, 0, 0, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: grab;
  color: #94a3b8;
  flex-shrink: 0;

  &:active {
    cursor: grabbing;
  }
}

.row-content {
  flex: 1;
  padding: 12px 16px;
}

.row-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.row-title {
  font-size: 14px;
  font-weight: 500;
  flex: 1;
}

.row-meta {
  font-size: 12px;
  color: #94a3b8;
  margin-left: 8px;

  &--muted {
    color: #64748b;
  }
}

.row-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.row-preview {
  display: flex;
  gap: 12px;
  overflow-x: auto;
  padding: 4px 0;
}

.preview-card {
  background: linear-gradient(135deg, #334155, #1e293b);
  border-radius: 6px;
  flex-shrink: 0;
  overflow: hidden;
  position: relative;

  &--real {
    background: #1e293b;
    display: flex;
    flex-direction: column;

    .preview-card-media {
      flex: 1;
      position: relative;
      overflow: hidden;
      min-height: 0;

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
        display: block;
      }
    }

    .preview-card-caption {
      font-size: 10px;
      line-height: 1.2;
      color: #cbd5e1;
      padding: 4px 6px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      background: rgba(0, 0, 0, 0.35);
    }
  }

  &--placeholder {
    opacity: 0.6;
  }

  .preview-card-label {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    color: #94a3b8;
    padding: 4px;
    text-align: center;
  }

  .card-style-poster & {
    width: 120px;
    height: 180px;
  }
  .card-style-landscape & {
    width: 200px;
    height: 113px;
  }
  .card-style-square & {
    width: 120px;
    height: 120px;
  }
  .card-style-banner & {
    width: 100%;
    height: 240px;
  }
}

.empty-canvas {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 400px;

  :deep(.el-empty__description p) {
    color: #94a3b8;
  }
}

.canvas-toolbar {
  height: 40px;
  background: rgba(0, 0, 0, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: #cbd5e1;
  font-size: 13px;
}

.config-form {
  :deep(.el-form-item__label) {
    font-weight: 500;
  }
}
</style>
