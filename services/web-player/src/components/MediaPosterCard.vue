<template>
  <component
    :is="tag"
    class="media-poster-card mh-media-card"
    :type="tag === 'button' ? 'button' : undefined"
    @click="onClick"
  >
    <div class="mh-poster-card" :class="{ 'mh-poster-card--landscape': landscape }">
      <img v-if="posterUrl" :src="posterUrl" :alt="title" loading="lazy" />
      <span v-else class="placeholder">{{ title.slice(0, 2) }}</span>
      <div v-if="rating && rating > 0" class="mh-rating-badge">⭐ {{ rating.toFixed(1) }}</div>
      <div v-if="progress != null && duration" class="progress-track">
        <div
          class="progress-fill"
          :style="{ width: `${Math.min(100, (progress / duration) * 100)}%` }"
        />
      </div>
      <slot name="badge" />
    </div>
    <div v-if="showMeta" class="meta">
      <div class="meta-title">{{ title }}</div>
      <div v-if="subtitle" class="meta-sub">{{ subtitle }}</div>
    </div>
  </component>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  title: string
  posterUrl?: string
  rating?: number
  subtitle?: string
  landscape?: boolean
  progress?: number
  duration?: number
  showMeta?: boolean
  tag?: 'button' | 'div'
}>(), {
  showMeta: true,
  tag: 'button',
})

const emit = defineEmits<{ click: [] }>()

function onClick() {
  emit('click')
}
</script>

<style scoped lang="scss">
.placeholder {
  font-size: 28px;
  font-weight: 700;
  color: var(--mh-text-muted);
}

.progress-track {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 3px;
  background: rgba(0, 0, 0, 0.45);
}

.progress-fill {
  height: 100%;
  background: var(--mh-primary);
  border-radius: 0 2px 2px 0;
}

.meta {
  margin-top: var(--mh-space-2);
}

.meta-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--mh-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.meta-sub {
  margin-top: 2px;
  font-size: 12px;
  color: var(--mh-text-muted);
}
</style>
