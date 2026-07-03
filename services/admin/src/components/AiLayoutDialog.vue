<template>
  <el-dialog
    v-model="visible"
    title="AI 布局生成器"
    width="680px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <!-- 输入区 -->
    <div v-if="!result" class="ai-input-section">
      <div class="section-label">描述你想要的首页布局</div>
      <el-input
        v-model="prompt"
        type="textarea"
        :rows="4"
        placeholder="例如：给我一个电影之夜的首页，要有：&#10;- 顶部热门电影轮播&#10;- 恐怖片专区 TOP 10&#10;- 最近添加的新片&#10;- 继续观看"
        resize="none"
      />

      <div class="divider-or">
        <span>或</span>
      </div>

      <div class="section-label">上传 UI 原型图</div>
      <el-upload
        class="image-uploader"
        drag
        :auto-upload="false"
        :show-file-list="false"
        accept="image/*"
        :on-change="handleFileChange"
      >
        <div v-if="!imageFile" class="upload-placeholder">
          <el-icon class="upload-icon"><Plus /></el-icon>
          <div class="upload-text">拖拽图片到此处，或点击上传</div>
          <div class="upload-hint">支持 JPG/PNG，AI 会分析布局结构</div>
        </div>
        <div v-else class="upload-preview">
          <img :src="imagePreview" alt="preview" />
          <div class="image-name">{{ imageFile.name }}</div>
          <el-button size="small" type="danger" @click.stop="clearImage">移除</el-button>
        </div>
      </el-upload>

      <!-- 快速示例 -->
      <div class="quick-examples">
        <div class="section-label">快速示例</div>
        <div class="example-chips">
          <el-tag
            v-for="ex in examples"
            :key="ex"
            class="example-chip"
            effect="plain"
            @click="prompt = ex"
          >
            {{ ex }}
          </el-tag>
        </div>
      </div>
    </div>

    <!-- 结果预览 -->
    <div v-else class="ai-result-section">
      <el-alert
        :title="result.explanation"
        type="success"
        show-icon
        :closable="false"
        style="margin-bottom: 16px"
      />

      <div class="result-rows">
        <div
          v-for="(row, i) in result.config.rows"
          :key="i"
          class="result-row"
        >
          <span class="row-icon">{{ rowIcon(row.type) }}</span>
          <span class="row-title">{{ row.title || rowTypeLabel(row.type) }}</span>
          <span class="row-source">{{ sourceLabel(row.source?.type) }}</span>
          <span class="row-card">{{ row.card_style || 'poster' }}</span>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">
          {{ result ? '关闭' : '取消' }}
        </el-button>
        <template v-if="!result">
          <el-button
            type="primary"
            :loading="loading"
            :disabled="!canGenerate"
            @click="doGenerate"
          >
            生成布局
          </el-button>
        </template>
        <template v-else>
          <el-button @click="resetAndRetry">重新生成</el-button>
          <el-button type="primary" @click="applyToEditor">
            应用到编辑器
          </el-button>
        </template>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { layoutApi, type LayoutConfig } from '@/api/layout'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'apply', config: LayoutConfig): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const prompt = ref('')
const imageFile = ref<File | null>(null)
const imagePreview = ref('')
const loading = ref(false)
const result = ref<{ config: LayoutConfig; explanation: string } | null>(null)

const canGenerate = computed(() => {
  return prompt.value.trim() || imageFile.value
})

const examples = [
  '电影之夜：热门电影榜单 + 恐怖片专区 + 新片',
  '追剧模式：继续观看 + 猜你喜欢 + 高分剧集',
  '儿童专区：动画榜单 + 合家欢 + 教育纪录片',
  '沉浸式：大图轮播 + 热门专题 + 榜单',
  '发现探索：猜你喜欢 + 冷门佳片 + 分类浏览',
]

const ROW_ICONS: Record<string, string> = {
  'hero-banner': '🎬',
  'ranking': '🏆',
  'topic': '📌',
  'shelf': '📚',
  'category-grid': '🗂️',
  'text-banner': '📢',
  'divider': '➖️',
}

const ROW_LABELS: Record<string, string> = {
  'hero-banner': 'Hero 轮播',
  'ranking': '榜单',
  'topic': '专题',
  'shelf': '内容架',
  'category-grid': '分类网格',
  'text-banner': '公告栏',
  'divider': '分隔线',
}

const SOURCE_LABELS: Record<string, string> = {
  'manual': '手动选择',
  'library': '库筛选',
  'trending': '热门',
  'continue-watching': '续播',
  'recently-added': '最近添加',
  'similar-to': '相似推荐',
  'recommend-algorithm': '推荐算法',
  'guess-you-like': '猜你喜欢',
  'album': '专辑',
  'category': '分类',
  'tag': '标签',
  'union': '合并源',
}

function rowIcon(type: string) {
  return ROW_ICONS[type] || '📄'
}

function rowTypeLabel(type: string) {
  return ROW_LABELS[type] || type
}

function sourceLabel(type?: string) {
  if (!type) return '无数据源'
  return SOURCE_LABELS[type] || type
}

function handleFileChange(file: any) {
  imageFile.value = file.raw
  const reader = new FileReader()
  reader.onload = (e) => {
    imagePreview.value = e.target?.result as string
  }
  reader.readAsDataURL(file.raw)
}

function clearImage() {
  imageFile.value = null
  imagePreview.value = ''
}

async function doGenerate() {
  loading.value = true
  try {
    let res
    if (imageFile.value) {
      res = await layoutApi.aiGenerateFromImage(imageFile.value)
    } else {
      res = await layoutApi.aiGenerate(prompt.value)
    }
    result.value = res.data
  } catch (e: any) {
    ElMessage.error(e?.message || 'AI 生成失败，请检查 AI 服务配置')
  } finally {
    loading.value = false
  }
}

function resetAndRetry() {
  result.value = null
}

function applyToEditor() {
  if (result.value) {
    emit('apply', result.value.config)
    visible.value = false
    reset()
  }
}

function reset() {
  prompt.value = ''
  imageFile.value = null
  imagePreview.value = ''
  result.value = null
  loading.value = false
}

function handleClose() {
  visible.value = false
  reset()
}
</script>

<style lang="scss" scoped>
.ai-input-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.divider-or {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--el-text-color-placeholder);
  font-size: 13px;

  &::before, &::after {
    content: '';
    flex: 1;
    height: 1px;
    background: var(--el-border-color-lighter);
  }
}

.image-uploader {
  :deep(.el-upload-dragger) {
    padding: 24px;
  }
}

.upload-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.upload-icon {
  font-size: 40px;
  color: var(--el-text-color-placeholder);
}

.upload-text {
  font-size: 14px;
  color: var(--el-text-color-regular);
}

.upload-hint {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.upload-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;

  img {
    max-width: 100%;
    max-height: 200px;
    object-fit: contain;
    border-radius: 8px;
  }
}

.image-name {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.quick-examples {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.example-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.example-chip {
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    color: var(--el-color-primary);
    border-color: var(--el-color-primary);
  }
}

.ai-result-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.result-rows {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.result-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  font-size: 14px;
}

.row-icon {
  font-size: 20px;
}

.row-title {
  font-weight: 500;
  flex: 1;
}

.row-source {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.row-card {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  background: var(--el-fill-color-lighter);
  padding: 2px 8px;
  border-radius: 4px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
