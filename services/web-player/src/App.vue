<template>
  <router-view />
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
</style>
