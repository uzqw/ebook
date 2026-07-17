<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { BookOpen, Eye, EyeOff, Loader2 } from '@lucide/vue'
import { authApi } from '@/services/api'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'

const router = useRouter(); const route = useRoute()
const email = ref(import.meta.env.DEV ? 'demo@reader.local' : '')
const password = ref(import.meta.env.DEV ? 'ebook-reader-user-123' : '')
const showPassword = ref(false)
const loading = ref(false); const error = ref('')
async function submit() {
  loading.value = true; error.value = ''
  try { await authApi.login(email.value, password.value); await router.push(String(route.query.redirect || '/books')) }
  catch (err) { error.value = err instanceof Error ? err.message : '登录失败' }
  finally { loading.value = false }
}
</script>
<template>
  <main class="grid min-h-[100dvh] place-items-center bg-[#f4f7ee] p-6">
    <form class="auth-card w-full max-w-md p-8" @submit.prevent="submit">
      <div class="mb-8 flex items-center gap-3">
        <span class="grid size-11 place-items-center rounded-2xl bg-[#15803d] text-white shadow-sm"><BookOpen class="size-6" /></span>
        <div><p class="text-xs font-extrabold uppercase tracking-widest text-[#705c21]">Ebook Reader</p><h1 class="text-2xl font-extrabold text-[#142217]">登录青简书房</h1></div>
      </div>
      <div class="space-y-4">
        <label class="grid gap-1.5 text-sm font-bold text-[#384c3d]">邮箱<Input v-model="email" type="email" autocomplete="email" required /></label>
        <label class="grid gap-1.5 text-sm font-bold text-[#384c3d]">密码
          <span class="relative block">
            <Input v-model="password" :type="showPassword ? 'text' : 'password'" autocomplete="current-password" class="pr-11" required />
            <button type="button" class="absolute right-1 top-1/2 grid size-9 -translate-y-1/2 place-items-center rounded-md text-[#384c3d] hover:bg-[#edf3e8]" :aria-label="showPassword ? '隐藏密码' : '显示密码'" @click="showPassword = !showPassword">
              <EyeOff v-if="showPassword" class="size-4" />
              <Eye v-else class="size-4" />
            </button>
          </span>
        </label>
        <p v-if="error" role="alert" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700">{{ error }}</p>
        <Button type="submit" class="w-full" :disabled="loading"><Loader2 v-if="loading" class="size-4 animate-spin" />登录</Button>
      </div>
      <p class="mt-5 text-center text-sm text-[#384c3d]">还没有账号？<RouterLink class="font-extrabold text-[#15803d]" to="/register">立即注册</RouterLink></p>
    </form>
  </main>
</template>
