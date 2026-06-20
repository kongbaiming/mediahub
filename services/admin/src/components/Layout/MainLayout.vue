<template>
  <div class="main-layout">
    <aside class="sidebar" :class="{ collapsed }">
      <div class="sidebar-header">
        <div class="logo">
          <el-icon size="24"><Film /></el-icon>
          <span v-if="!collapsed" class="logo-text">MediaHub</span>
        </div>
      </div>
      <el-menu
        :default-active="route.path"
        :collapse="collapsed"
        router
        class="sidebar-menu"
        background-color="transparent"
        text-color="#cbd5e1"
        active-text-color="#fff"
      >
        <el-menu-item index="/dashboard">
          <el-icon><DataAnalysis /></el-icon>
          <template #title>总览</template>
        </el-menu-item>
        <el-menu-item index="/media">
          <el-icon><Files /></el-icon>
          <template #title>媒资库</template>
        </el-menu-item>
        <el-menu-item index="/layouts">
          <el-icon><Grid /></el-icon>
          <template #title>布局</template>
        </el-menu-item>
        <el-menu-item index="/downloads">
          <el-icon><Download /></el-icon>
          <template #title>下载管理</template>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Setting /></el-icon>
          <template #title>设置</template>
        </el-menu-item>
      </el-menu>
    </aside>

    <div class="main-container">
      <header class="topbar">
        <div class="topbar-left">
          <el-button text @click="toggleSidebar">
            <el-icon size="20"><Fold v-if="!collapsed" /><Expand v-else /></el-icon>
          </el-button>
          <span class="page-title">{{ route.meta?.title || 'MediaHub' }}</span>
        </div>
        <div class="topbar-right">
          <el-dropdown v-if="auth.profiles.length > 1" @command="switchProfile">
            <span class="profile-switcher">
              <el-avatar :size="28" :src="auth.activeProfile?.avatar_url">
                {{ auth.activeProfile?.name?.[0] }}
              </el-avatar>
              <span class="profile-name">{{ auth.activeProfile?.name }}</span>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  v-for="p in auth.profiles"
                  :key="p.id"
                  :command="p.id"
                  :disabled="p.id === auth.activeProfileId"
                >
                  {{ p.name }}
                  <el-tag v-if="p.is_kid" size="small" type="warning">儿童</el-tag>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>

          <el-dropdown @command="handleCommand">
            <span class="user-info">
              <el-avatar :size="28">{{ auth.user?.username?.[0]?.toUpperCase() }}</el-avatar>
              <span class="user-name">{{ auth.user?.display_name || auth.user?.username }}</span>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="logout">
                  <el-icon><SwitchButton /></el-icon>
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <main class="main-content">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const collapsed = ref(false)
const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

function toggleSidebar() {
  collapsed.value = !collapsed.value
}

function switchProfile(profileId: string) {
  auth.switchProfile(profileId)
}

function handleCommand(cmd: string) {
  if (cmd === 'logout') {
    auth.logout()
    router.push('/login')
  }
}
</script>

<style lang="scss" scoped>
.main-layout {
  display: flex;
  height: 100vh;
  background: #f8fafc;
}

.sidebar {
  width: 220px;
  background: linear-gradient(180deg, #1e293b, #0f172a);
  color: #cbd5e1;
  transition: width 0.2s;
  flex-shrink: 0;

  &.collapsed {
    width: 64px;
  }
}

.sidebar-header {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.logo {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #fff;
  font-weight: 600;
  font-size: 16px;
  letter-spacing: 0.5px;
}

.logo-text {
  background: linear-gradient(135deg, #6366f1, #ec4899);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.sidebar-menu {
  border-right: none !important;
  background: transparent !important;

  :deep(.el-menu-item) {
    &.is-active {
      background: rgba(99, 102, 241, 0.15) !important;
      border-left: 3px solid #6366f1;
    }
    &:hover {
      background: rgba(255, 255, 255, 0.05) !important;
    }
  }
}

.main-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.topbar {
  height: 56px;
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-title {
  font-size: 16px;
  font-weight: 500;
  color: #1e293b;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.profile-switcher, .user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 12px;
  border-radius: 6px;
  transition: background 0.15s;

  &:hover {
    background: #f1f5f9;
  }
}

.profile-name, .user-name {
  font-size: 13px;
  color: #475569;
}

.main-content {
  flex: 1;
  overflow: auto;
  padding: 24px;
}

.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
