<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { BookOpen, FileUp, RefreshCw, Trash2 } from '@lucide/vue'
import { booksApi } from '@/services/api'
import type { BookRecord } from '@/types/models'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'

const books = ref<BookRecord[]>([])
const loading = ref(false)
const deleting = ref(false)
const error = ref('')
const deleteDialogOpen = ref(false)
const pendingDeleteBook = ref<BookRecord | null>(null)

const deleteDescription = computed(() => pendingDeleteBook.value
  ? `将永久删除《${pendingDeleteBook.value.title}》以及解析页面、书签、笔记和阅读记录。此操作不可撤销。`
  : '将永久删除这本书以及相关阅读数据。')

const statusTone = (status: string) => status === 'completed' ? 'green' : status === 'failed' ? 'red' : 'amber'
const statusText = (status: string) => ({ pending: '待解析', processing: '解析中', completed: '已解析', failed: '解析失败' }[status] || status)

async function load() {
  loading.value = true
  error.value = ''
  try {
    books.value = await booksApi.list()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function requestRemove(book: BookRecord) {
  pendingDeleteBook.value = book
  deleteDialogOpen.value = true
}

async function confirmRemove() {
  if (!pendingDeleteBook.value) return
  deleting.value = true
  error.value = ''
  try {
    await booksApi.remove(pendingDeleteBook.value.id)
    deleteDialogOpen.value = false
    pendingDeleteBook.value = null
    await load()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '删除失败'
  } finally {
    deleting.value = false
  }
}

onMounted(load)
</script>

<template>
  <section>
    <div class="page-header">
      <div>
        <p class="text-xs font-extrabold uppercase tracking-widest text-[#705c21]">Library</p>
        <h1 class="text-3xl font-extrabold text-[#142217]">书籍管理</h1>
        <p class="mt-2 text-sm text-[#384c3d]">上传书籍后由 PocketBase 后端解析页数、文本，并按页渲染阅读。</p>
      </div>
      <div class="flex gap-2">
        <Button variant="outline" @click="load"><RefreshCw data-icon="inline-start" />刷新</Button>
        <RouterLink to="/books/upload"><Button><FileUp data-icon="inline-start" />上传书籍</Button></RouterLink>
      </div>
    </div>

    <p v-if="error" class="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700">{{ error }}</p>
    <div v-if="loading" class="panel text-[#384c3d]">正在加载书架...</div>
    <div v-else-if="!books.length" class="panel grid place-items-center py-16 text-center">
      <BookOpen class="mb-3 size-12 text-[#15803d]" />
      <h2 class="text-xl font-extrabold">书架还是空的</h2>
      <p class="mt-2 text-sm text-[#384c3d]">上传第一本书，开始您的阅读之旅。</p>
      <RouterLink class="mt-5" to="/books/upload"><Button>上传第一本书</Button></RouterLink>
    </div>
    <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <article v-for="book in books" :key="book.id" class="page-card flex flex-col gap-4">
        <div class="flex items-start justify-between gap-3">
          <div>
            <h2 class="line-clamp-2 text-lg font-extrabold text-[#142217]">{{ book.title }}</h2>
            <p class="mt-1 text-sm text-[#384c3d]">{{ book.author || '未知作者' }} · {{ book.page_count || 0 }} 页</p>
          </div>
          <Badge :tone="statusTone(book.parse_status)">{{ statusText(book.parse_status) }}</Badge>
        </div>
        <p class="line-clamp-3 min-h-14 text-sm text-[#384c3d]">{{ book.description || book.parse_error || '暂无简介。' }}</p>
        <div class="mt-auto flex flex-wrap gap-2">
          <RouterLink :to="`/books/${book.id}/read`"><Button size="sm"><BookOpen data-icon="inline-start" />阅读</Button></RouterLink>
          <RouterLink :to="`/books/${book.id}/info`"><Button size="sm" variant="outline">信息</Button></RouterLink>
          <Button size="sm" variant="ghost" class="text-red-700" @click="requestRemove(book)"><Trash2 data-icon="inline-start" />删除</Button>
        </div>
      </article>
    </div>

    <AlertDialog
      v-model:open="deleteDialogOpen"
      title="删除书籍？"
      :description="deleteDescription"
      confirm-text="删除书籍"
      cancel-text="取消"
      :loading="deleting"
      @confirm="confirmRemove"
    />
  </section>
</template>
