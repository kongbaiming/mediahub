<template>
  <router-view v-slot="{ Component, route }">
    <Transition name="mh-route" mode="out-in">
      <component :is="Component" :key="route.fullPath" />
    </Transition>
  </router-view>
  <ToastHost />
</template>

<script setup lang="ts">
import ToastHost from '@/components/ToastHost.vue'
</script>

<style>
#app {
  min-height: 100vh;
  background: var(--mh-bg);
  color: var(--mh-text);
  font-family: var(--mh-font-body);
}

.mh-route-enter-active,
.mh-route-leave-active {
  transition: opacity 0.22s var(--mh-ease-out), transform 0.22s var(--mh-ease-out);
}

.mh-route-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.mh-route-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
