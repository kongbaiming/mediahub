<template>
  <div class="login-page">
    <div class="login-bg">
      <div class="bg-blob blob-1"></div>
      <div class="bg-blob blob-2"></div>
      <div class="bg-blob blob-3"></div>
    </div>

    <div class="login-container">
      <div class="login-header">
        <div class="brand">
          <el-icon size="40"><Film /></el-icon>
          <h1>MediaHub</h1>
        </div>
        <p class="subtitle">家庭媒资中心</p>
      </div>

      <el-card class="login-card" shadow="never">
        <el-tabs v-model="mode" class="login-tabs">
          <el-tab-pane label="登录" name="login" />
          <el-tab-pane label="注册" name="register" />
        </el-tabs>

        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          label-position="top"
          @submit.prevent="onSubmit"
        >
          <el-form-item label="用户名" prop="username">
            <el-input
              v-model="form.username"
              placeholder="admin"
              :prefix-icon="User"
              size="large"
            />
          </el-form-item>

          <el-form-item label="密码" prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="••••••"
              :prefix-icon="Lock"
              size="large"
              show-password
              @keyup.enter="onSubmit"
            />
          </el-form-item>

          <el-form-item v-if="mode === 'register'" label="昵称（可选）" prop="displayName">
            <el-input
              v-model="form.displayName"
              placeholder="主人"
              size="large"
            />
          </el-form-item>

          <el-button
            type="primary"
            size="large"
            :loading="loading"
            class="submit-btn"
            @click="onSubmit"
          >
            {{ mode === 'login' ? '登录' : '创建账号' }}
          </el-button>
        </el-form>

        <div class="hint">
          <el-text size="small" type="info">
            首次使用请点「注册」创建家庭账号（默认管理员）
          </el-text>
        </div>
      </el-card>

      <div class="footer">
        <el-text size="small" type="info">© MediaHub · 自建家庭媒资中心</el-text>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const mode = ref<'login' | 'register'>('login')
const loading = ref(false)
const formRef = ref<FormInstance>()

const form = reactive({
  username: '',
  password: '',
  displayName: '',
})

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 50, message: '长度 3-50', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' },
  ],
}

async function onSubmit() {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    if (mode.value === 'login') {
      await auth.login(form.username, form.password)
    } else {
      await auth.register(form.username, form.password, form.displayName)
    }
    ElMessage.success(mode.value === 'login' ? '登录成功' : '注册成功')
    const redirect = (route.query.redirect as string) || '/dashboard'
    router.push(redirect)
  } catch (err: any) {
    // axios interceptor already toasted
  } finally {
    loading.value = false
  }
}
</script>

<style lang="scss" scoped>
.login-page {
  min-height: 100vh;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--mh-bg);
  overflow: hidden;
}

.login-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: radial-gradient(ellipse 80% 60% at 50% 30%, rgba(0, 122, 255, 0.08), transparent),
              radial-gradient(ellipse 60% 40% at 80% 70%, rgba(88, 86, 214, 0.06), transparent);
}

.login-container {
  position: relative;
  z-index: 1;
  width: 380px;
  max-width: 92vw;
}

.login-header {
  text-align: center;
  margin-bottom: 28px;
}

.brand {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;

  h1 {
    margin: 0;
    font-size: 28px;
    font-family: var(--mh-font-display);
    font-weight: 600;
    background: linear-gradient(135deg, var(--mh-primary), var(--mh-secondary));
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
    letter-spacing: -0.02em;
  }
}

.subtitle {
  margin-top: var(--mh-space-2);
  color: var(--mh-text-secondary);
  font-size: 14px;
}

.login-card {
  background: var(--mh-surface);
  border: 1px solid var(--mh-border);
  border-radius: var(--mh-radius-xl);
  padding: var(--mh-space-6);
  box-shadow: var(--mh-shadow-xl);
}

:deep(.el-tabs__item) {
  color: var(--mh-text-secondary) !important;
  font-weight: 500;

  &.is-active {
    color: var(--mh-primary) !important;
  }
}

:deep(.el-tabs__active-bar) {
  background-color: var(--mh-primary) !important;
  height: 2px;
}

:deep(.el-tabs__nav-wrap::after) {
  display: none;
}

:deep(.el-form-item__label) {
  color: var(--mh-text-secondary) !important;
  font-weight: 500;
  font-size: 13px;
}

:deep(.el-input__wrapper) {
  background: var(--mh-surface-secondary) !important;
  border: 1px solid var(--mh-border) !important;
  box-shadow: none !important;
  border-radius: var(--mh-radius-sm) !important;
  transition: all var(--mh-duration-fast) var(--mh-ease);

  &:hover {
    border-color: var(--mh-border-strong) !important;
  }

  &.is-focus {
    border-color: var(--mh-primary) !important;
    box-shadow: 0 0 0 3px var(--mh-primary-muted) !important;
  }
}

:deep(.el-input__inner) {
  color: var(--mh-text) !important;

  &::placeholder {
    color: var(--mh-text-tertiary) !important;
  }
}

.submit-btn {
  width: 100%;
  margin-top: var(--mh-space-4);
  background: var(--mh-primary);
  border: none;
  font-weight: 600;
  font-size: 15px;
  letter-spacing: 0;
  height: 44px;
  border-radius: var(--mh-radius-sm);
  transition: all var(--mh-duration-fast) var(--mh-ease);

  &:hover {
    background: var(--mh-primary-hover);
    transform: translateY(-1px);
  }

  &:active {
    transform: translateY(0) scale(0.99);
  }
}

.hint {
  margin-top: 20px;
  text-align: center;
}

.footer {
  text-align: center;
  margin-top: 24px;
}
</style>
