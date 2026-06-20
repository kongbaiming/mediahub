<template>
  <Teleport to="body">
    <Toast
      v-for="t in toasts"
      :key="t.id"
      :visible="true"
      :message="t.message"
      :type="t.type"
    />
  </Teleport>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import Toast from './Toast.vue'

interface ToastItem {
  id: number
  message: string
  type: 'info' | 'success' | 'warning' | 'error'
}

const toasts = ref<ToastItem[]>([])
let counter = 0

function show(message: string, type: ToastItem['type'] = 'info', duration = 3000) {
  const id = ++counter
  toasts.value.push({ id, message, type })
  setTimeout(() => {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }, duration)
}

function handleEvent(e: Event) {
  const detail = (e as CustomEvent).detail
  show(detail.message, detail.type, detail.duration)
}

onMounted(() => window.addEventListener('toast', handleEvent))
onUnmounted(() => window.removeEventListener('toast', handleEvent))

// 暴露到 window 方便任何地方调用
declare global {
  interface Window {
    toast: (message: string, type?: ToastItem['type'], duration?: number) => void
  }
}

window.toast = show
</script>

<style scoped>
/* 多个 toast 纵向排列 */
:deep(.toast-container) {
  position: fixed;
}
:deep(.toast-container):nth-child(n+2) {
  top: calc(24px + 60px * var(--toast-index, 1));
}
</style>