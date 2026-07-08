<template>
  <div class="settings-page">
    <h2 class="page-h2">设置</h2>

    <el-card shadow="never" class="panel">
      <template #header><span>个人信息</span></template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="用户名">{{ auth.user?.username }}</el-descriptions-item>
        <el-descriptions-item label="昵称">{{ auth.user?.display_name || '-' }}</el-descriptions-item>
        <el-descriptions-item label="角色">
          <el-tag :type="auth.isAdmin ? 'danger' : 'info'">{{ auth.isAdmin ? '管理员' : '成员' }}</el-tag>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card shadow="never" class="panel">
      <template #header>
        <div class="card-header">
          <span>家庭成员 Profile</span>
          <el-button size="small" type="primary" @click="openProfileCreate">添加成员</el-button>
        </div>
      </template>
      <el-table :data="profiles" stripe>
        <el-table-column label="名称" prop="name" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.is_kid" type="warning" size="small">儿童</el-tag>
            <el-tag v-else size="small">普通</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160">
          <template #default="{ row }">
            <el-button size="small" @click="openProfileEdit(row as Profile)">编辑</el-button>
            <el-button size="small" type="danger" @click="removeProfile((row as Profile).id, (row as Profile).name)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <p class="hint">Profile 用于播放端个性化推荐、想看列表与库外推荐；顶栏可切换当前 Profile。</p>
    </el-card>

    <el-card shadow="never" class="panel">
      <template #header>
        <div class="card-header">
          <span>服务状态</span>
          <el-button size="small" :loading="checkingServices" @click="checkServices">刷新</el-button>
        </div>
      </template>
      <el-table :data="serviceRows" stripe>
        <el-table-column label="服务" prop="name" width="140" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="row.ok ? 'success' : row.warn ? 'warning' : 'danger'" size="small">
              {{ row.statusLabel }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="说明" prop="message" show-overflow-tooltip />
      </el-table>
    </el-card>

    <el-card shadow="never" class="panel">
      <template #header><span>媒体扫描</span></template>
      <el-descriptions v-if="scanConfig" :column="2" border>
        <el-descriptions-item label="定时扫描">
          {{ scanConfig.enabled ? '已启用' : '已关闭' }}
        </el-descriptions-item>
        <el-descriptions-item label="间隔">
          {{ scanIntervalLabel(scanConfig.interval_minutes) }}
        </el-descriptions-item>
        <el-descriptions-item label="扫描目录" :span="2">
          <span v-if="scanConfig.roots?.length">{{ scanConfig.roots.join('、') }}</span>
          <span v-else>—</span>
        </el-descriptions-item>
        <el-descriptions-item label="上次扫描">
          {{ scanConfig.last_scan_at || '—' }}
        </el-descriptions-item>
        <el-descriptions-item label="结果">
          {{ scanConfig.last_scan_message || scanConfig.last_scan_status || '—' }}
        </el-descriptions-item>
      </el-descriptions>
      <p class="hint">扫描配置可在「媒资库」页面修改；扫描目录由 API 环境变量 MEDIA_ROOTS 决定。</p>
    </el-card>

    <el-card shadow="never" class="panel">
      <template #header><span>服务信息</span></template>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="API 地址">{{ apiBase }}</el-descriptions-item>
        <el-descriptions-item label="Swagger">
          <el-link :href="`${apiBase}/swagger/index.html`" target="_blank" type="primary">打开 API 文档</el-link>
        </el-descriptions-item>
        <el-descriptions-item label="Admin 版本">0.4.0</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-dialog v-model="profileDialog" :title="profileForm.id ? '编辑成员' : '添加成员'" width="420">
      <el-form label-position="top">
        <el-form-item label="昵称" required>
          <el-input v-model="profileForm.name" maxlength="50" />
        </el-form-item>
        <el-form-item label="儿童模式">
          <el-switch v-model="profileForm.is_kid" />
        </el-form-item>
        <el-form-item v-if="profileForm.is_kid" label="PIN（4-8 位）">
          <el-input v-model="profileForm.pin" maxlength="8" placeholder="新建或修改 PIN" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="profileDialog = false">取消</el-button>
        <el-button type="primary" :loading="profileSaving" @click="saveProfile">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import {
  profileApi,
  dashboardApi,
  downloaderApi,
  indexerApi,
  scannerApi,
  SCAN_INTERVAL_OPTIONS,
  type MediaScanConfig,
} from '@/api/client'
import type { Profile } from '@/api/types'

const auth = useAuthStore()
const apiBase = import.meta.env.VITE_API_BASE_URL || location.origin

const profiles = ref<Profile[]>([])
const checkingServices = ref(false)
const scanConfig = ref<MediaScanConfig | null>(null)

interface ServiceRow {
  name: string
  ok: boolean
  warn: boolean
  statusLabel: string
  message: string
}

const serviceRows = ref<ServiceRow[]>([])

const profileDialog = ref(false)
const profileSaving = ref(false)
const profileForm = reactive({
  id: '',
  name: '',
  is_kid: false,
  pin: '',
})

function scanIntervalLabel(minutes: number) {
  return SCAN_INTERVAL_OPTIONS.find((o) => o.value === minutes)?.label || `${minutes} 分钟`
}

async function loadProfiles() {
  const res = await profileApi.list()
  profiles.value = res.data || []
  await auth.bootstrap()
}

async function checkServices() {
  checkingServices.value = true
  const rows: ServiceRow[] = []
  try {
    const health = await dashboardApi.health()
    const db = health.data?.database
    rows.push({
      name: '数据库',
      ok: db?.status === 'ok',
      warn: false,
      statusLabel: db?.status === 'ok' ? '正常' : '异常',
      message: db?.message || (db?.status === 'ok' ? 'PostgreSQL 连接正常' : ''),
    })
  } catch (e: any) {
    rows.push({
      name: '数据库',
      ok: false,
      warn: false,
      statusLabel: '异常',
      message: e?.message || '无法获取健康状态',
    })
  }

  try {
    const cfg = await scannerApi.getConfig()
    scanConfig.value = cfg.data
    rows.push({
      name: '媒体扫描',
      ok: true,
      warn: !cfg.data?.enabled,
      statusLabel: cfg.data?.enabled ? '已启用' : '已关闭',
      message: cfg.data?.roots?.length
        ? `${cfg.data.roots.length} 个目录`
        : '未配置 MEDIA_ROOTS',
    })
  } catch (e: any) {
    rows.push({
      name: '媒体扫描',
      ok: false,
      warn: false,
      statusLabel: '异常',
      message: e?.message || '',
    })
  }

  try {
    const h = await downloaderApi.health()
    rows.push({
      name: 'qBittorrent',
      ok: h.status === 'ok',
      warn: h.status === 'disabled',
      statusLabel: h.status === 'ok' ? '已连接' : h.status === 'disabled' ? '未启用' : '不可用',
      message:
        h.status === 'ok'
          ? '下载管理可用'
          : h.error || '请配置 DOWNLOADER_ENABLED 与 QBITTORRENT_URL',
    })
  } catch (e: any) {
    rows.push({
      name: 'qBittorrent',
      ok: false,
      warn: false,
      statusLabel: '不可用',
      message: e?.response?.data?.error || e?.message || '',
    })
  }

  try {
    const res = await indexerApi.search({ q: 'test', limit: 1 })
    rows.push({
      name: 'Prowlarr 索引',
      ok: res.status === 'ok',
      warn: res.status === 'unavailable',
      statusLabel: res.status === 'ok' ? '已配置' : '未配置',
      message:
        res.status === 'ok'
          ? '资源搜索可用'
          : res.message || '请配置 INDEXER_URL 与 INDEXER_API_KEY',
    })
  } catch (e: any) {
    rows.push({
      name: 'Prowlarr 索引',
      ok: false,
      warn: true,
      statusLabel: '未配置',
      message: e?.message || '',
    })
  }

  serviceRows.value = rows
  checkingServices.value = false
}

function openProfileCreate() {
  profileForm.id = ''
  profileForm.name = ''
  profileForm.is_kid = false
  profileForm.pin = ''
  profileDialog.value = true
}

function openProfileEdit(p: Profile) {
  profileForm.id = p.id
  profileForm.name = p.name
  profileForm.is_kid = p.is_kid
  profileForm.pin = ''
  profileDialog.value = true
}

async function saveProfile() {
  if (!profileForm.name.trim()) {
    ElMessage.warning('请输入昵称')
    return
  }
  profileSaving.value = true
  try {
    const payload = {
      name: profileForm.name.trim(),
      is_kid: profileForm.is_kid,
      pin: profileForm.pin || undefined,
    }
    if (profileForm.id) {
      await profileApi.update(profileForm.id, payload)
      ElMessage.success('已更新')
    } else {
      await profileApi.create(payload)
      ElMessage.success('已添加')
    }
    profileDialog.value = false
    await loadProfiles()
  } finally {
    profileSaving.value = false
  }
}

async function removeProfile(id: string, name: string) {
  try {
    await ElMessageBox.confirm(`确认删除成员「${name}」？`, '删除确认', { type: 'warning' })
  } catch {
    return
  }
  await profileApi.remove(id)
  ElMessage.success('已删除')
  await loadProfiles()
}

onMounted(async () => {
  await loadProfiles()
  await checkServices()
})
</script>

<style lang="scss" scoped>
.settings-page {
  max-width: 960px;
}

.page-h2 {
  margin: 0 0 var(--mh-space-5);
  font-size: 22px;
  font-weight: 600;
}

.panel {
  margin-bottom: var(--mh-space-4);
  border-radius: var(--mh-radius-lg);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.hint {
  margin: var(--mh-space-3) 0 0;
  font-size: 12px;
  color: var(--mh-text-muted);
}
</style>
