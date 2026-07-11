<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { Loader2 } from '@lucide/vue'
import { authApi } from '@/services/api'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
const router = useRouter(); const name = ref(''); const email = ref(''); const password = ref('')
const loading = ref(false); const error = ref('')
async function submit() { loading.value = true; error.value = ''; try { await authApi.register(name.value, email.value, password.value); await router.push('/books') } catch (err) { error.value = err instanceof Error ? err.message : '注册失败' } finally { loading.value = false } }
</script>
<template><main class="grid min-h-screen place-items-center bg-[#f4f7ee] p-6"><form class="auth-card w-full max-w-md p-8" @submit.prevent="submit"><p class="text-xs font-extrabold uppercase tracking-widest text-[#705c21]">Ebook Reader</p><h1 class="mb-8 text-2xl font-extrabold text-[#142217]">创建读者账号</h1><div class="space-y-4"><label class="grid gap-1.5 text-sm font-bold text-[#384c3d]">昵称<Input v-model="name" required /></label><label class="grid gap-1.5 text-sm font-bold text-[#384c3d]">邮箱<Input v-model="email" type="email" required /></label><label class="grid gap-1.5 text-sm font-bold text-[#384c3d]">密码<Input v-model="password" type="password" required /></label><p v-if="error" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700">{{ error }}</p><Button type="submit" class="w-full" :disabled="loading"><Loader2 v-if="loading" class="size-4 animate-spin" />注册并登录</Button></div><p class="mt-5 text-center text-sm text-[#384c3d]">已有账号？<RouterLink class="font-extrabold text-[#15803d]" to="/login">返回登录</RouterLink></p></form></main></template>
