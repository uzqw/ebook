<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { Loader2 } from '@lucide/vue'
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
  <section>
    <div class="page-header">
      <div>
        <p class="text-xs font-extrabold uppercase tracking-widest text-[#705c21]">Notes</p>
        <h1 class="text-3xl font-extrabold text-[#142217]">笔记管理</h1>
        <p class="mt-2 text-sm text-[#384c3d]">按书籍汇总阅读中沉淀的页面笔记。</p>
      </div>
    </div>
    <div v-if="error" role="alert" class="mb-4 flex flex-wrap items-center justify-between gap-2 rounded-lg border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700">
      <span>{{ error }}</span>
      <Button size="sm" variant="outline" @click="load">重试</Button>
    </div>
    <div v-if="loading" class="panel flex items-center gap-2 text-[#384c3d]"><Loader2 class="size-4 animate-spin" />正在加载笔记...</div>
    <div v-else class="grid gap-4">
      <article v-for="book in books" :key="book.id" class="page-card">
        <h2 class="text-lg font-extrabold">{{ book.title }}</h2>
        <div v-if="!notesByBook[book.id]?.length" class="mt-3 text-sm text-[#384c3d]">暂无笔记</div>
        <div v-else class="mt-3 grid gap-2">
          <div v-for="note in notesByBook[book.id]" :key="note.id" class="rounded-lg border border-[#cbe0bf] bg-white p-3 text-sm">
            <RouterLink class="font-extrabold text-[#15803d] hover:underline" :to="`/books/${book.id}/read?page=${note.page_number}`">第 {{ note.page_number }} 页</RouterLink>
            <p class="mt-1 whitespace-pre-wrap text-[#384c3d]">{{ note.content }}</p>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>
