<template>
  <el-dialog v-model="visible" title="切换 Profile" width="500" :show-close="false" class="profile-dialog">
    <div class="profile-list">
      <div
        v-for="p in profiles"
        :key="p.id"
        class="profile-card"
        :class="{ active: p.id === activeId }"
        @click="onSelect(p)"
      >
        <div class="avatar">
          {{ p.name.slice(0, 1) }}
          <el-tag v-if="p.is_kid" size="small" type="warning" class="kid-badge">儿童</el-tag>
        </div>
        <div class="profile-name">{{ p.name }}</div>
      </div>

      <div class="profile-card add-new" @click="onCreate">
        <div class="avatar-add">+</div>
        <div class="profile-name">添加</div>
      </div>
    </div>

    <template #footer>
      <el-button @click="visible = false">关闭</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="createVisible" title="添加家庭成员" width="400" append-to-body>
    <el-form :model="newProfile" label-position="top">
      <el-form-item label="昵称">
        <el-input v-model="newProfile.name" placeholder="如：老婆、孩子" />
      </el-form-item>
      <el-form-item label="儿童模式">
        <el-switch v-model="newProfile.is_kid" />
        <div class="hint">开启后会过滤成人内容（需要先去 CMS 标记媒资为成人内容）</div>
      </el-form-item>
      <el-form-item v-if="newProfile.is_kid" label="家长 PIN（4-8 位）">
        <el-input v-model="newProfile.pin" type="password" placeholder="退出儿童模式时需要" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="createVisible = false">取消</el-button>
      <el-button type="primary" @click="onCreateConfirm" :loading="creating">创建</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import { profileApi } from '@/api'
import {
  syncProfiles,
  setActiveProfileId,
  getActiveProfileId,
  saveProfilesLocal,
  type LocalProfile,
} from '@/utils/profiles'

const props = defineProps<{
  modelValue: boolean
}>()
const emit = defineEmits<{
  'update:modelValue': [v: boolean]
  switched: [profile: LocalProfile]
}>()

const visible = ref(false)
const createVisible = ref(false)
const profiles = ref<LocalProfile[]>([])
const activeId = ref<string>('')
const creating = ref(false)
const newProfile = reactive({
  name: '',
  is_kid: false,
  pin: '',
})

watch(
  () => props.modelValue,
  (v) => {
    visible.value = v
    if (v) loadProfiles()
  },
)

watch(visible, (v) => emit('update:modelValue', v))

async function loadProfiles() {
  profiles.value = await syncProfiles()
  activeId.value = getActiveProfileId()
}

async function onSelect(p: LocalProfile) {
  if (p.is_kid) {
    const pin = prompt(`切换到「${p.name}」需要输入家长 PIN：`)
    if (!pin) return
    try {
      await profileApi.verifyPin(p.id, pin)
    } catch {
      window.toast?.('PIN 错误', 'error', 2500)
      return
    }
  }
  activeId.value = p.id
  setActiveProfileId(p.id)
  emit('switched', p)
  window.toast?.(`已切换到「${p.name}」`, 'success', 2000)
  visible.value = false
}

function onCreate() {
  newProfile.name = ''
  newProfile.is_kid = false
  newProfile.pin = ''
  createVisible.value = true
}

async function onCreateConfirm() {
  if (!newProfile.name.trim()) {
    window.toast?.('请输入昵称', 'warning', 2000)
    return
  }
  if (newProfile.is_kid && newProfile.pin.length < 4) {
    window.toast?.('儿童 Profile 请设置 4-8 位 PIN', 'warning', 2500)
    return
  }
  creating.value = true
  try {
    const p = await profileApi.create({
      name: newProfile.name.trim(),
      is_kid: newProfile.is_kid,
      pin: newProfile.is_kid ? newProfile.pin : undefined,
    })
    const local: LocalProfile = { id: p.id, name: p.name, is_kid: p.is_kid }
    profiles.value.push(local)
    saveProfilesLocal(profiles.value)
    createVisible.value = false
    window.toast?.(`已添加「${p.name}」`, 'success', 2000)
  } catch (e: any) {
    window.toast?.(e?.response?.data?.message || '创建失败', 'error', 3000)
  } finally {
    creating.value = false
  }
}
</script>

<style lang="scss" scoped>
.profile-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
  gap: 16px;
  padding: 8px 0;
}

.profile-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--mh-space-2);
  padding: var(--mh-space-4);
  background: rgba(255, 255, 255, 0.04);
  border: 2px solid transparent;
  border-radius: var(--mh-radius-md);
  cursor: pointer;
  transition: background var(--mh-duration) var(--mh-ease),
    transform var(--mh-duration) var(--mh-ease),
    border-color var(--mh-duration) var(--mh-ease);

  &:hover {
    background: rgba(255, 255, 255, 0.08);
    transform: translateY(-2px);
  }

  &.active {
    border-color: var(--mh-primary);
    background: var(--mh-primary-muted);
    box-shadow: var(--mh-shadow-glow);
  }
}

.avatar {
  width: 64px;
  height: 64px;
  border-radius: var(--mh-radius-full);
  background: linear-gradient(135deg, var(--mh-primary), var(--mh-accent));
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  font-weight: 600;
  position: relative;
}

.avatar-add {
  width: 64px;
  height: 64px;
  border-radius: var(--mh-radius-full);
  background: rgba(255, 255, 255, 0.06);
  border: 1px dashed var(--mh-outline-strong);
  color: var(--mh-text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 36px;
  font-weight: 300;
}

.kid-badge {
  position: absolute;
  bottom: -4px;
  right: -4px;
}

.profile-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--mh-text-secondary);
}

.add-new {
  background: transparent;
  border: 2px dashed var(--mh-outline-strong);

  &:hover {
    background: rgba(255, 255, 255, 0.04);
    border-color: var(--mh-primary);
  }
}

.hint {
  font-size: 12px;
  color: var(--mh-text-muted);
  margin-top: var(--mh-space-1);
}
</style>
