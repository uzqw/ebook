<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { BookOpen, Loader2 } from '@lucide/vue'
import { authApi } from '@/services/api'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'

const router = useRouter(); const route = useRoute()
const email = ref('demo@reader.local'); const password = ref('ebook-reader-user-123')
const loading = ref(false); const error = ref('')
async function submit() {
  loading.value = true; error.value = ''
  try { await authApi.login(email.value, password.value); await router.push(String(route.query.redirect || '/books')) }
  catch (err) { error.value = err instanceof Error ? err.message : '登录失败' }
  finally { loading.value = false }
}
</script>
<template>
  <main class="grid min-h-screen place-items-center bg-[#f4f7ee] p-6">
    <form class="auth-card w-full max-w-md p-8" @submit.prevent="submit">
      <div class="mb-8 flex items-center gap-3">
        <span class="grid size-11 place-items-center rounded-2xl bg-[#15803d] text-white shadow-sm"><BookOpen class="size-6" /></span>
        <div><p class="text-xs font-extrabold uppercase tracking-widest text-[#705c21]">Ebook Reader</p><h1 class="text-2xl font-extrabold text-[#142217]">登录青简书房</h1></div>
      </div>
      <div class="space-y-4">
        <label class="grid gap-1.5 text-sm font-bold text-[#384c3d]">邮箱<Input v-model="email" type="email" required /></label>
        <label class="grid gap-1.5 text-sm font-bold text-[#384c3d]">密码<Input v-model="password" type="password" required /></label>
        <p v-if="error" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700">{{ error }}</p>
        <Button type="submit" class="w-full" :disabled="loading"><Loader2 v-if="loading" class="size-4 animate-spin" />登录</Button>
      </div>
      <p class="mt-5 text-center text-sm text-[#384c3d]">还没有账号？<RouterLink class="font-extrabold text-[#15803d]" to="/register">立即注册</RouterLink></p>
    </form>
  </main>
</template>
