<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { booksApi, notesApi } from '@/services/api'
import type { BookRecord, NoteRecord } from '@/types/models'
const books = ref<BookRecord[]>([]); const notesByBook = ref<Record<string, NoteRecord[]>>({})
onMounted(async () => { books.value = await booksApi.list(); for (const b of books.value) notesByBook.value[b.id] = await notesApi.list(b.id) })
</script>
<template><section><div class="page-header"><div><p class="text-xs font-extrabold uppercase tracking-widest text-[#705c21]">Notes</p><h1 class="text-3xl font-extrabold text-[#142217]">笔记管理</h1><p class="mt-2 text-sm text-[#384c3d]">按书籍汇总阅读中沉淀的页面笔记。</p></div></div><div class="grid gap-4"><article v-for="book in books" :key="book.id" class="page-card"><h2 class="text-lg font-extrabold">{{ book.title }}</h2><div v-if="!notesByBook[book.id]?.length" class="mt-3 text-sm text-[#384c3d]">暂无笔记</div><div v-else class="mt-3 grid gap-2"><div v-for="note in notesByBook[book.id]" :key="note.id" class="rounded-lg border border-[#cbe0bf] bg-white p-3 text-sm"><strong>第 {{ note.page_number }} 页</strong><p class="mt-1 whitespace-pre-wrap text-[#384c3d]">{{ note.content }}</p></div></div></article></div></section></template>
