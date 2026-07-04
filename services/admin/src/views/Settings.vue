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
        <el-descriptions-item label="家庭成员">
          <el-tag v-for="p in auth.profiles" :key="p.id" class="profile-tag">
            {{ p.name }}
            <el-tag v-if="p.is_kid" size="small" type="warning">儿童</el-tag>
          </el-tag>
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card shadow="never" class="panel">
      <template #header><span>服务信息</span></template>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="API 地址">{{ apiBase }}</el-descriptions-item>
        <el-descriptions-item label="Swagger">
          <el-link :href="`${apiBase}/swagger/index.html`" target="_blank" type="primary">
            打开 API 文档
          </el-link>
        </el-descriptions-item>
        <el-descriptions-item label="版本">0.4.0</el-descriptions-item>
        <el-descriptions-item label="构建">dev</el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const apiBase = import.meta.env.VITE_API_BASE_URL || location.origin
</script>

<style lang="scss" scoped>
.settings-page {
  max-width: 800px;
}

.page-h2 {
  margin: 0 0 var(--mh-space-5);
  font-size: 22px;
  font-weight: 600;
  color: var(--mh-text);
}

.panel {
  margin-bottom: var(--mh-space-4);
  border-radius: var(--mh-radius-lg);
}

.profile-tag {
  margin-right: var(--mh-space-2);
  margin-bottom: 4px;
}
</style>
