<template>
  <header class="app-topbar mh-topbar" :class="{ 'app-topbar--sub': variant === 'sub' }">
    <div class="app-topbar__start">
      <slot name="start">
        <button v-if="showBack" type="button" class="mh-back-btn" @click="onBack">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            <path d="M15 18l-6-6 6-6" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <span>{{ backLabel }}</span>
        </button>
      </slot>
    </div>

    <div class="app-topbar__center">
      <slot name="center">
        <nav v-if="breadcrumbs.length" class="app-topbar__crumb" aria-label="面包屑">
          <template v-for="(item, idx) in breadcrumbs" :key="idx">
            <button
              v-if="item.to"
              type="button"
              class="crumb-link"
              @click="$router.push(item.to)"
            >
              {{ item.label }}
            </button>
            <span v-else class="crumb-current">{{ item.label }}</span>
            <span v-if="idx < breadcrumbs.length - 1" class="crumb-sep">/</span>
          </template>
        </nav>
        <h1 v-else-if="title" class="app-topbar__title">{{ title }}</h1>
      </slot>
    </div>

    <div class="app-topbar__end">
      <slot name="end" />
    </div>
  </header>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

export type Crumb = { label: string; to?: string }

withDefaults(defineProps<{
  variant?: 'main' | 'sub'
  showBack?: boolean
  backLabel?: string
  title?: string
  breadcrumbs?: Crumb[]
}>(), {
  variant: 'sub',
  showBack: true,
  backLabel: '返回',
  breadcrumbs: () => [],
})

const router = useRouter()

function onBack() {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push('/')
  }
}
</script>

<style scoped lang="scss">
.app-topbar {
  justify-content: space-between;
  gap: var(--mh-space-4);

  &--sub {
    .app-topbar__start {
      flex: 0 0 auto;
    }

    .app-topbar__center {
      flex: 1;
      min-width: 0;
    }

    .app-topbar__end {
      flex: 0 0 auto;
    }
  }
}

.app-topbar__start,
.app-topbar__center,
.app-topbar__end {
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
}

.app-topbar__title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.app-topbar__crumb {
  display: flex;
  align-items: center;
  gap: var(--mh-space-2);
  min-width: 0;
  font-size: 14px;
  color: var(--mh-text-muted);
}

.crumb-link {
  background: none;
  border: none;
  padding: 0;
  color: var(--mh-text-secondary);
  cursor: pointer;
  font: inherit;

  &:hover {
    color: var(--mh-primary);
  }
}

.crumb-current {
  color: var(--mh-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: min(40vw, 320px);
}

.crumb-sep {
  opacity: 0.45;
}
</style>
