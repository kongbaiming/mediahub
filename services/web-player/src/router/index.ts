import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/Home.vue'),
    },
    {
      path: '/search',
      name: 'search',
      component: () => import('@/views/Search.vue'),
    },
    {
      path: '/media/:id',
      name: 'media-detail',
      component: () => import('@/views/Detail.vue'),
    },
    {
      path: '/library',
      name: 'library',
      component: () => import('@/views/Library.vue'),
    },
    {
      path: '/play/:id',
      name: 'play',
      component: () => import('@/views/Player.vue'),
    },
  ],
})

export default router
