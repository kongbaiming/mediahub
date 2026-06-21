/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

interface Window {
  toast: (
    message: string,
    type?: 'info' | 'success' | 'warning' | 'error',
    duration?: number,
  ) => void
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<object, object, unknown>
  export default component
}

// 全局 Element Plus 组件（app.use(ElementPlus) 后模板可用）
import type {} from 'vue'

declare module 'vue' {
  export interface GlobalComponents {
    ElMessage: typeof import('element-plus/es')['ElMessage']
    ElInput: typeof import('element-plus/es')['ElInput']
    ElForm: typeof import('element-plus/es')['ElForm']
    ElFormItem: typeof import('element-plus/es')['ElFormItem']
    ElButton: typeof import('element-plus/es')['ElButton']
    ElDialog: typeof import('element-plus/es')['ElDialog']
    ElIcon: typeof import('element-plus/es')['ElIcon']
    ElTag: typeof import('element-plus/es')['ElTag']
    ElSwitch: typeof import('element-plus/es')['ElSwitch']
  }
}
