/// <reference types="vite/client" />

// Vite 环境变量类型
interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

// 全局 window 扩展（由 components/ToastHost.vue 挂载）
interface Window {
  toast: (
    message: string,
    type?: 'info' | 'success' | 'warning' | 'error',
    duration?: number,
  ) => void
}

// 全局组件类型（main.ts 里 app.use(ElementPlus) 后可用）
// vue-tsc 默认不知道全局注册的组件，需要在这里声明
declare module 'vue' {
  export interface GlobalComponents {
    ElMessage: typeof import('element-plus/es')['ElMessage']
    ElInput: typeof import('element-plus/es')['ElInput']
    ElForm: typeof import('element-plus/es')['ElForm']
    ElFormItem: typeof import('element-plus/es')['ElFormItem']
    ElButton: typeof import('element-plus/es')['ElButton']
    ElDialog: typeof import('element-plus/es')['ElDialog']
    ElIcon: typeof import('element-plus/es')['ElIcon']
  }
}