<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { FileUp, Loader2 } from '@lucide/vue'
import { booksApi } from '@/services/api'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Textarea from '@/components/ui/Textarea.vue'
const router = useRouter(); const title = ref(''); const author = ref(''); const description = ref(''); const file = ref<File | null>(null); const loading = ref(false); const error = ref('')
function onFile(e: Event) { const f = (e.target as HTMLInputElement).files?.[0] || null; file.value = f; if (f && !title.value) title.value = f.name.replace(/\.(pdf|epub|mobi)$/i, '') }
async function submit() {
  if (!file.value) { error.value = '请选择电子书文件'; return }
  loading.value = true
  error.value = ''
  try {
    const book = await booksApi.upload({ title: title.value, author: author.value, description: description.value, file: file.value })
    await router.push(`/books/${book.id}/read`)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '上传失败'
  } finally {
    loading.value = false
  }
}
</script>
<template>
  <section class="max-w-3xl">
    <div class="page-header">
      <div>
        <p class="text-xs font-extrabold uppercase tracking-widest text-[#705c21]">Upload</p>
        <h1 class="text-3xl font-extrabold text-[#142217]">上传书籍</h1>
        <p class="mt-2 text-sm text-[#384c3d]">支持 PDF、EPUB、MOBI：上传后立即跳转到阅读页，解析在后台进行，完成后自动显示页面。</p>
      </div>
    </div>
    <form class="page-card space-y-5" @submit.prevent="submit">
      <label class="grid gap-1.5 text-sm font-bold text-[#384c3d]">选择电子书 (PDF, EPUB, MOBI)
        <input class="flex h-11 w-full rounded-lg border border-input bg-white px-3 py-2 text-base shadow-sm md:text-sm" type="file" accept="application/pdf,.pdf,application/epub+zip,.epub,application/x-mobipocket-ebook,.mobi" required @change="onFile" />
      </label>
      <label class="grid gap-1.5 text-sm font-bold text-[#384c3d]">书名<Input v-model="title" required placeholder="例如：Java 场景题攻略" /></label>
      <label class="grid gap-1.5 text-sm font-bold text-[#384c3d]">作者<Input v-model="author" placeholder="可选" /></label>
      <label class="grid gap-1.5 text-sm font-bold text-[#384c3d]">简介<Textarea v-model="description" placeholder="可选：写一点这本书的用途或备注" /></label>
      <p v-if="error" role="alert" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700">{{ error }}</p>
      <Button type="submit" :disabled="loading"><Loader2 v-if="loading" class="size-4 animate-spin" /><FileUp v-else class="size-4" />{{ loading ? '上传中...' : '上传并解析' }}</Button>
    </form>
  </section>
</template>
