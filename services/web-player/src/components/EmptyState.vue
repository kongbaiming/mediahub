<template>
  <div class="empty-state">
    <div class="empty-icon" aria-hidden="true">{{ icon }}</div>
    <h3 class="empty-title">{{ title }}</h3>
    <p v-if="description" class="empty-desc">{{ description }}</p>
    <button v-if="action" class="empty-action mh-btn mh-btn--accent" @click="action.onClick">
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
  padding: var(--mh-space-12) var(--mh-page-gutter);
  min-height: min(420px, 50vh);
  text-align: center;
}

.empty-icon {
  font-size: 56px;
  opacity: 0.35;
  margin-bottom: var(--mh-space-4);
  filter: grayscale(0.2);
}

.empty-title {
  margin: 0 0 var(--mh-space-2);
  color: var(--mh-text);
  font-size: 20px;
  font-weight: 600;
  font-family: var(--mh-font-display);
  letter-spacing: -0.02em;
}

.empty-desc {
  margin: 0 0 var(--mh-space-6);
  color: var(--mh-text-muted);
  font-size: 14px;
  max-width: 420px;
  line-height: 1.65;
}

.empty-action {
  min-width: 160px;
}
</style>
