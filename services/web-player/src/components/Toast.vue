<template>
  <Transition name="fade" appear>
    <div v-if="visible" class="toast-container" :class="`toast-${type}`" role="status" aria-live="polite">
      <span class="toast-icon" aria-hidden="true">{{ icon }}</span>
      <span class="toast-message">{{ message }}</span>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type ToastType = 'info' | 'success' | 'warning' | 'error'

interface Props {
  visible: boolean
  message: string
  type?: ToastType
}

const props = withDefaults(defineProps<Props>(), {
  type: 'info',
})

const icon = computed(() => {
  switch (props.type) {
    case 'success': return '✓'
    case 'warning': return '⚠'
    case 'error':   return '✕'
    default:        return 'ℹ'
  }
})
</script>

<style scoped>
.toast-container {
  position: fixed;
  top: var(--mh-space-6);
  left: 50%;
  transform: translateX(-50%);
  z-index: 9999;
  display: flex;
  align-items: center;
  gap: var(--mh-space-3);
  padding: var(--mh-space-3) var(--mh-space-6);
  border-radius: var(--mh-radius-md);
  box-shadow: var(--mh-shadow-lg);
  font-size: 14px;
  font-weight: 500;
  backdrop-filter: blur(16px) saturate(1.2);
  border: 1px solid var(--mh-outline);
  font-family: var(--mh-font-body);
}

.toast-info    { background: rgba(108, 99, 255, 0.92); color: white; }
.toast-success { background: rgba(52, 211, 153, 0.92); color: white; }
.toast-warning { background: rgba(251, 191, 36, 0.95); color: var(--mh-bg); }
.toast-error   { background: rgba(248, 113, 113, 0.92); color: white; }

.toast-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.25);
  font-size: 13px;
  font-weight: bold;
}

.fade-enter-active, .fade-leave-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-20px);
}
</style>