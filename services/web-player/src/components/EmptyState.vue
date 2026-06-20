<template>
  <div class="empty-state">
    <div class="empty-icon" aria-hidden="true">{{ icon }}</div>
    <h3 class="empty-title">{{ title }}</h3>
    <p v-if="description" class="empty-desc">{{ description }}</p>
    <button v-if="action" class="empty-action" @click="action.onClick">
      {{ action.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  icon?: string
  title: string
  description?: string
  action?: { label: string; onClick: () => void }
}>(), {
  icon: '🎬',
})
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 64px 24px;
  text-align: center;
}

.empty-icon {
  font-size: 64px;
  opacity: 0.4;
  margin-bottom: 16px;
}

.empty-title {
  margin: 0 0 8px;
  color: #cbd5e1;
  font-size: 18px;
  font-weight: 600;
}

.empty-desc {
  margin: 0 0 24px;
  color: #94a3b8;
  font-size: 14px;
  max-width: 400px;
  line-height: 1.6;
}

.empty-action {
  padding: 10px 24px;
  background: #6366f1;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s, transform 0.15s;
}

.empty-action:hover {
  background: #4f46e5;
  transform: translateY(-1px);
}

.empty-action:active {
  transform: translateY(0);
}
</style>