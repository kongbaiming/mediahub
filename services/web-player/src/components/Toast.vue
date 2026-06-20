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

const props = defineProps<{
  visible: boolean
  message: string
  type?: 'info' | 'success' | 'warning' | 'error'
}>()

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
  top: 24px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 9999;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 24px;
  border-radius: 8px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.4);
  font-size: 14px;
  font-weight: 500;
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.toast-info    { background: rgba(99, 102, 241, 0.95); color: white; }
.toast-success { background: rgba(34, 197, 94, 0.95); color: white; }
.toast-warning { background: rgba(251, 191, 36, 0.95); color: #1e293b; }
.toast-error   { background: rgba(239, 68, 68, 0.95); color: white; }

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