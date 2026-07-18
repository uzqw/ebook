<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { Loader2, NotebookPen, ExternalLink } from '@lucide/vue'
import { booksApi, notesApi } from '@/services/api'
import type { BookRecord, NoteRecord } from '@/types/models'
import Button from '@/components/ui/Button.vue'

const books = ref<BookRecord[]>([])
const notesByBook = ref<Record<string, NoteRecord[]>>({})
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const list = await booksApi.list()
    books.value = list
    const entries = await Promise.all(list.map((book) => notesApi.list(book.id).then((notes) => [book.id, notes] as const)))
    notesByBook.value = Object.fromEntries(entries)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="max-w-5xl mx-auto py-2">
    <div class="page-header">
      <div>
        <p class="text-xs font-extrabold uppercase tracking-widest text-[#0f7643]">Notes</p>
        <h1 class="text-3xl font-black text-[#0f1e14] tracking-tight">笔记管理</h1>
        <p class="mt-1 text-sm text-[#4a5c50]">按书籍汇总您在阅读中记录的页面重点与想法心得。</p>
      </div>
    </div>

    <div v-if="error" role="alert" class="mb-4 flex flex-wrap items-center justify-between gap-2 rounded-lg border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700">
      <span>{{ error }}</span>
      <Button size="sm" variant="outline" @click="load">重试</Button>
    </div>

    <div v-if="loading" class="panel flex items-center gap-2 text-[#4a5c50]">
      <Loader2 class="size-4 animate-spin text-[#0f7643]" />正在加载笔记...
    </div>

    <div v-else class="grid gap-6">
      <article v-for="book in books" :key="book.id" class="library-card rounded-2xl p-5 flex flex-col gap-3">
        <div class="flex items-center gap-2.5 pb-2 border-b border-emerald-500/5">
          <NotebookPen class="size-5 text-[#0f7643] opacity-80" />
          <h2 class="text-base font-extrabold text-[#0f1e14] truncate">{{ book.title }}</h2>
          <span v-if="notesByBook[book.id]?.length" class="text-xs font-semibold px-2 py-0.5 rounded-full bg-emerald-50 text-[#0f7643] border border-emerald-500/10">
            {{ notesByBook[book.id].length }} 条笔记
          </span>
        </div>

        <div v-if="!notesByBook[book.id]?.length" class="text-xs text-[#4a5c50]/60 italic py-1">
          暂无笔记。在阅读书籍时选中任意段落即可记录您的读书笔记。
        </div>

        <div v-else class="grid gap-3 mt-1 grid-cols-1 md:grid-cols-2">
          <div v-for="note in notesByBook[book.id]" :key="note.id" class="rounded-xl border border-emerald-500/5 bg-[#fbfcfb] hover:bg-emerald-50/5 p-4 text-sm flex flex-col justify-between gap-2.5 transition-colors">
            <p class="whitespace-pre-wrap text-[#0f1e14] leading-relaxed font-medium">{{ note.content }}</p>
            
            <div class="flex justify-end pt-2 border-t border-dashed border-emerald-500/5">
              <RouterLink 
                class="inline-flex items-center gap-1 text-xs font-bold text-[#0f7643] hover:underline" 
                :to="`/books/${book.id}/read?page=${note.page_number}`"
              >
                <span>第 {{ note.page_number }} 页</span>
                <ExternalLink class="size-3" />
              </RouterLink>
            </div>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>
