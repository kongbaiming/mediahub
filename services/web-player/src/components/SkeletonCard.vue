<template>
  <div class="skeleton-card" :style="cardStyle">
    <div class="skeleton-poster shimmer" />
    <div class="skeleton-text">
      <div class="skeleton-line shimmer" :style="{ width: '85%' }" />
      <div class="skeleton-line shimmer" :style="{ width: '60%' }" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  width?: number | string
  height?: number | string
}>(), {
  width: 180,
  height: 270,
})

const cardStyle = computed(() => ({
  width: typeof props.width === 'number' ? `${props.width}px` : props.width,
}))
</script>

<style scoped>
.skeleton-card {
  display: flex;
  flex-direction: column;
  gap: var(--mh-space-2);
}

.skeleton-poster {
  width: 100%;
  height: 270px;
  background: var(--mh-surface-variant);
  border-radius: var(--mh-radius-md);
  border: 1px solid var(--mh-outline);
}

.skeleton-text {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 0 var(--mh-space-2);
}

.skeleton-line {
  height: 12px;
  background: var(--mh-surface-variant);
  border-radius: 6px;
}

.shimmer {
  position: relative;
  overflow: hidden;
}

.shimmer::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(108, 99, 255, 0.08) 50%,
    transparent 100%
  );
  animation: shimmer 1.5s infinite;
}

@keyframes shimmer {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(100%); }
}
</style>
