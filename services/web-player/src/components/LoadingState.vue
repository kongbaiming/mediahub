<template>
  <div class="loading-state" :class="{ 'with-bg': background }">
    <div class="spinner" aria-label="加载中"></div>
    <p v-if="message" class="message">{{ message }}</p>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  message?: string
  background?: boolean
}>(), {
  message: '加载中…',
  background: false,
})
</script>

<style scoped>
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 48px 24px;
  min-height: 200px;
}

.loading-state.with-bg {
  background: rgba(30, 41, 59, 0.4);
  border-radius: 12px;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid rgba(99, 102, 241, 0.15);
  border-top-color: #6366f1;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.message {
  margin: 0;
  color: #94a3b8;
  font-size: 14px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>