<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { FileUp, Loader2 } from '@lucide/vue'
import { booksApi } from '@/services/api'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Textarea from '@/components/ui/Textarea.vue'

const router = useRouter()
const title = ref('')
const author = ref('')
const description = ref('')
const file = ref<File | null>(null)
const loading = ref(false)
const error = ref('')
const fileInput = ref<HTMLInputElement | null>(null)

function onFile(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0] || null
  file.value = f
  if (f && !title.value) {
    title.value = f.name.replace(/\.(pdf|epub|mobi)$/i, '')
  }
}

async function submit() {
  if (!file.value) {
    error.value = '请选择电子书文件'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const book = await booksApi.upload({
      title: title.value,
      author: author.value,
      description: description.value,
      file: file.value
    })
    await router.push(`/books/${book.id}/read`)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '上传失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="max-w-2xl mx-auto py-2">
    <div class="page-header">
      <div>
        <p class="text-xs font-extrabold uppercase tracking-widest text-[#0f7643] hidden sm:block">Upload</p>
        <h1 class="text-xl sm:text-3xl font-black text-[#0f1e14] tracking-tight">上传书籍</h1>
        <p class="mt-1 text-sm text-[#4a5c50] hidden sm:block">支持 PDF、EPUB、MOBI。上传后即刻在后台进行解析，完成后自动生成精美图册。</p>
      </div>
    </div>
    
    <form class="panel mt-6 space-y-6" @submit.prevent="submit">
      <div class="grid gap-2">
        <label class="text-sm font-bold text-[#0f1e14]">电子书文件</label>
        
        <div 
          class="border-2 border-dashed border-emerald-500/20 hover:border-[#0f7643]/40 bg-emerald-50/5 hover:bg-emerald-50/15 rounded-2xl p-8 flex flex-col items-center justify-center cursor-pointer transition-all gap-2 text-center"
          @click="fileInput?.click()"
        >
          <FileUp class="size-8 text-[#0f7643] opacity-80" />
          <span class="text-sm font-semibold text-[#0f1e14]">{{ file ? file.name : '点击选择或拖拽电子书文件' }}</span>
          <span class="text-xs text-[#4a5c50]">支持 PDF, EPUB, MOBI 格式</span>
          <input 
            ref="fileInput"
            type="file" 
            class="hidden" 
            accept="application/pdf,.pdf,application/epub+zip,.epub,application/x-mobipocket-ebook,.mobi" 
            required 
            @change="onFile" 
          />
        </div>
      </div>
      
      <div class="grid gap-2">
        <label class="text-sm font-bold text-[#0f1e14]">书名</label>
        <Input v-model="title" required placeholder="例如：Java 场景题攻略" class="rounded-xl border-emerald-500/10 focus:border-[#0f7643]/30 focus:ring-1 focus:ring-[#0f7643]/30" />
      </div>
      
      <div class="grid gap-2">
        <label class="text-sm font-bold text-[#0f1e14]">作者</label>
        <Input v-model="author" placeholder="可选，例如：未知作者" class="rounded-xl border-emerald-500/10 focus:border-[#0f7643]/30 focus:ring-1 focus:ring-[#0f7643]/30" />
      </div>
      
      <div class="grid gap-2">
        <label class="text-sm font-bold text-[#0f1e14]">简介</label>
        <Textarea v-model="description" placeholder="可选：简要介绍或备注信息" class="rounded-xl border-emerald-500/10 focus:border-[#0f7643]/30 focus:ring-1 focus:ring-[#0f7643]/30 min-h-24" />
      </div>
      
      <p v-if="error" role="alert" class="rounded-xl border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700">{{ error }}</p>
      
      <Button type="submit" :disabled="loading" class="w-full bg-[#0f7643] hover:bg-[#064e2b] h-12 rounded-xl text-base shadow-sm">
        <Loader2 v-if="loading" class="size-5 animate-spin mr-2" />
        <FileUp v-else class="size-5 mr-2" />
        {{ loading ? '上传中...' : '开始上传并解析' }}
      </Button>
    </form>
  </section>
</template>
