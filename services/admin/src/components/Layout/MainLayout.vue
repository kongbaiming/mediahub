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
        <el-menu-item index="/scrape">
          <el-icon><Connection /></el-icon>
          <template #title>刮削中心</template>
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
  background: var(--mh-admin-bg);
}

.sidebar {
  width: var(--mh-sidebar-width);
  background: linear-gradient(180deg, var(--mh-admin-sidebar-from), var(--mh-admin-sidebar-to));
  color: var(--mh-text-on-dark);
  transition: width var(--mh-duration) var(--mh-ease);
  flex-shrink: 0;
  border-right: 1px solid rgba(255, 255, 255, 0.04);

  &.collapsed {
    width: var(--mh-sidebar-collapsed);
  }
}

.sidebar-header {
  height: var(--mh-topbar-height);
  display: flex;
  align-items: center;
  padding: 0 var(--mh-space-5);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.logo {
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
  color: #fff;
  font-weight: 600;
  font-size: 16px;
  font-family: var(--mh-font-display);
  letter-spacing: -0.01em;
}

.logo-text {
  background: linear-gradient(135deg, var(--mh-primary), var(--mh-accent));
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.sidebar-menu {
  border-right: none !important;
  background: transparent !important;
  padding: var(--mh-space-2) 0;

  :deep(.el-menu-item) {
    margin: 2px var(--mh-space-2);
    border-radius: var(--mh-radius-sm);
    transition: background var(--mh-duration) var(--mh-ease);

    &.is-active {
      background: var(--mh-primary-muted) !important;
      border-left: none;
      box-shadow: inset 3px 0 0 var(--mh-primary);
    }

    &:hover {
      background: rgba(255, 255, 255, 0.06) !important;
    }
  }
}

.main-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
}

.topbar {
  height: var(--mh-topbar-height);
  background: var(--mh-admin-surface);
  border-bottom: 1px solid var(--mh-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--mh-space-5);
  box-shadow: var(--mh-shadow-sm);
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: var(--mh-space-3);
}

.page-title {
  font-size: 16px;
  font-weight: 600;
  font-family: var(--mh-font-display);
  color: var(--mh-text);
  letter-spacing: -0.02em;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: var(--mh-space-4);
}

.profile-switcher, .user-info {
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
  cursor: pointer;
  padding: 4px var(--mh-space-3);
  border-radius: var(--mh-radius-sm);
  transition: background var(--mh-duration) var(--mh-ease);

  &:hover {
    background: var(--mh-admin-surface-muted);
  }
}

.profile-name, .user-name {
  font-size: 13px;
  color: var(--mh-text-secondary);
  font-weight: 500;
}

.main-content {
  flex: 1;
  overflow: auto;
  padding: var(--mh-space-6);
}

.fade-enter-active, .fade-leave-active {
  transition: opacity var(--mh-duration) var(--mh-ease);
}

.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
