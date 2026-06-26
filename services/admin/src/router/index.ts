import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/Login.vue'),
    meta: { public: true },
  },
  {
    path: '/',
    component: () => import('@/components/Layout/MainLayout.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '总览' },
      },
      {
        path: 'media',
        name: 'media-list',
        component: () => import('@/views/Media/List.vue'),
        meta: { title: '媒资库' },
      },
      {
        path: 'scrape',
        name: 'scrape-center',
        component: () => import('@/views/ScrapeCenter.vue'),
        meta: { title: '刮削中心' },
      },
      {
        path: 'media/:id',
        name: 'media-detail',
        component: () => import('@/views/Media/Detail.vue'),
        meta: { title: '媒资详情' },
      },
      {
        path: 'media-create',
        name: 'media-create',
        component: () => import('@/views/Media/Edit.vue'),
        meta: { title: '新建媒资' },
      },
      {
        path: 'layouts',
        name: 'layout-list',
        component: () => import('@/views/Layout/List.vue'),
        meta: { title: '布局列表' },
      },
      {
        path: 'layouts/:id',
        name: 'layout-editor',
        component: () => import('@/views/Layout/Editor.vue'),
        meta: { title: '布局编辑器' },
      },
      {
        path: 'downloads',
        name: 'downloads',
        component: () => import('@/views/Downloads.vue'),
        meta: { title: '下载管理' },
      },
      {
        path: 'want-to-watch',
        name: 'want-to-watch',
        component: () => import('@/views/WantToWatch.vue'),
        meta: { title: '播放端想看' },
      },
      {
        path: 'settings',
        name: 'settings',
        component: () => import('@/views/Settings.vue'),
        meta: { title: '设置' },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFound.vue'),
    meta: { public: true },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, _from, next) => {
  const auth = useAuthStore()
  if (to.meta.public) {
    next()
    return
  }
  if (!auth.isLoggedIn) {
    next({ name: 'login', query: { redirect: to.fullPath } })
    return
  }
  next()
})

router.afterEach((to) => {
  const title = (to.meta?.title as string) || 'MediaHub'
  document.title = `${title} · MediaHub Admin`
})

export default router
