<template>
  <div class="media-edit-page">
    <h2 class="page-h2">新建媒资</h2>

    <el-card shadow="never" class="form-card">
      <el-alert
        title="提示"
        type="info"
        :closable="false"
        show-icon
        description="手动创建媒资后，系统会自动调用 TMDB 搜索匹配元数据并入库。也可直接保存手动填写的信息。"
      />

      <el-form :model="form" label-position="top" class="form">
        <el-form-item label="类型" required>
          <el-radio-group v-model="form.type">
            <el-radio-button label="movie">电影</el-radio-button>
            <el-radio-button label="tvshow">剧集</el-radio-button>
            <el-radio-button label="anime">动画</el-radio-button>
            <el-radio-button label="documentary">纪录片</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="标题" required>
          <el-input v-model="form.title" placeholder="如：盗梦空间" />
        </el-form-item>

        <el-form-item label="原始标题">
          <el-input v-model="form.original_title" placeholder="如：Inception（可选）" />
        </el-form-item>

        <el-form-item label="年份">
          <el-input-number v-model="form.year" :min="1900" :max="2100" controls-position="right" />
        </el-form-item>

        <el-form-item label="存储路径（NAS 路径）" required>
          <el-input v-model="form.storage_path" placeholder="/media/movies/片名 (2010)/片名.mkv（容器内路径，勿填 /volume1/...）" />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :loading="saving" @click="onSubmit">
            <el-icon><Plus /></el-icon>
            创建并入库
          </el-button>
          <el-button @click="$router.back()">取消</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { mediaApi } from '@/api/media'

const router = useRouter()
const saving = ref(false)

const form = reactive({
  type: 'movie' as 'movie' | 'tvshow' | 'anime' | 'documentary',
  title: '',
  original_title: '',
  year: undefined as number | undefined,
  storage_path: '',
})

async function onSubmit() {
  if (!form.title || !form.storage_path) {
    ElMessage.warning('请填写标题和存储路径')
    return
  }
  saving.value = true
  try {
    const res = await mediaApi.create(form as any)
    ElMessage.success('创建成功，正在后台刮削元数据...')
    router.push(`/media/${(res as any).data.id}`)
  } finally {
    saving.value = false
  }
}
</script>

<style lang="scss" scoped>
.page-h2 {
  margin: 0 0 20px;
  font-size: 22px;
  font-weight: 600;
  color: #1e293b;
}

.form-card {
  border-radius: 12px;
}

.form {
  margin-top: 20px;
}
</style>
