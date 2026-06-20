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
  background: #0f172a;
  color: #e2e8f0;
  font-family: 'PingFang SC', 'Microsoft YaHei', system-ui, sans-serif;
}
</style>
