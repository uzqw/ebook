<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { BookOpen, FileUp, RefreshCw, Trash2, Search, Info } from '@lucide/vue'
import { booksApi } from '@/services/api'
import type { BookRecord } from '@/types/models'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'

const books = ref<BookRecord[]>([])
const loading = ref(false)
const deleting = ref(false)
const error = ref('')
const deleteDialogOpen = ref(false)
const pendingDeleteBook = ref<BookRecord | null>(null)

const searchQuery = ref('')
const statusFilter = ref('all')

const deleteDescription = computed(() => pendingDeleteBook.value
  ? `将永久删除《${pendingDeleteBook.value.title}》以及解析页面、书签、笔记和阅读记录。此操作不可撤销。`
  : '将永久删除这本书以及相关阅读数据。')

const statusTone = (status: string) => status === 'completed' ? 'green' : status === 'failed' ? 'red' : 'amber'
const statusText = (status: string) => ({ pending: '待解析', processing: '解析中', completed: '已解析', failed: '解析失败' }[status] || status)

// Generate deterministic beautiful pastel/dark gradient cover styles based on book title
function getBookCoverStyle(title: string) {
  let hash = 0
  for (let i = 0; i < title.length; i++) {
    hash = title.charCodeAt(i) + ((hash << 5) - hash)
  }
  const hue1 = Math.abs(hash % 360)
  const hue2 = Math.abs((hash + 120) % 360)
  // Emerald, forest, slate, and warm tones look great with 55% saturation and 30-40% lightness
  return {
    background: `linear-gradient(135deg, hsl(${hue1}, 55%, 35%) 0%, hsl(${hue2}, 55%, 25%) 100%)`
  }
}

// Clean title of filename extensions or common brackets for cover layout
function cleanCoverTitle(title: string) {
  return title
    .replace(/\.[a-zA-Z0-9]+$/, '') // remove extensions like .epub, .pdf
    .replace(/\(([^)]+)\)/g, '')   // remove brackets
    .replace(/\[([^\]]+)\]/g, '')
    .trim()
}

const filteredBooks = computed(() => {
  return books.value.filter(book => {
    const matchesSearch = 
      book.title.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      (book.author || '').toLowerCase().includes(searchQuery.value.toLowerCase())
    
    const matchesStatus = 
      statusFilter.value === 'all' || 
      book.parse_status === statusFilter.value
      
    return matchesSearch && matchesStatus
  })
})

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
        <p class="text-xs font-extrabold uppercase tracking-widest text-[#0f7643]">Library</p>
        <h1 class="text-3xl font-black text-[#0f1e14] tracking-tight">书籍管理</h1>
        <p class="mt-1 text-sm text-[#4a5c50]">管理并阅读您的电子书，上传后自动解析页数、文本，支持多端阅读进度同步。</p>
      </div>
      <div class="flex gap-2">
        <Button variant="outline" @click="load" class="border-emerald-500/10 hover:bg-emerald-50 text-[#0f7643]"><RefreshCw data-icon="inline-start" />刷新</Button>
        <RouterLink to="/books/upload"><Button class="bg-[#0f7643] hover:bg-[#064e2b]"><FileUp data-icon="inline-start" />上传书籍</Button></RouterLink>
      </div>
    </div>

    <!-- Search & Filter Bar -->
    <div class="mb-6 flex flex-col md:flex-row gap-3 items-center justify-between">
      <div class="relative w-full md:w-80">
        <Search class="absolute left-3.5 top-1/2 -translate-y-1/2 size-4 text-[#4a5c50]/60" />
        <Input
          v-model="searchQuery"
          placeholder="搜索书名或作者..."
          class="pl-10 h-10 w-full rounded-xl border border-emerald-500/10 bg-white shadow-sm focus:border-[#0f7643]/30 focus:ring-1 focus:ring-[#0f7643]/30 text-sm"
        />
      </div>
      <div class="flex gap-1.5 overflow-x-auto w-full md:w-auto pb-1 md:pb-0">
        <button 
          v-for="status in ['all', 'completed', 'processing', 'pending', 'failed']" 
          :key="status"
          @click="statusFilter = status"
          class="px-3.5 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all"
          :class="statusFilter === status 
            ? 'bg-[#0f7643] text-white shadow-sm shadow-emerald-700/20' 
            : 'bg-white hover:bg-emerald-50 text-[#4a5c50] border border-emerald-500/5'"
        >
          {{ { all: '全部', completed: '已解析', processing: '解析中', pending: '待解析', failed: '解析失败' }[status] }}
        </button>
      </div>
    </div>

    <div v-if="error" role="alert" class="mb-4 flex flex-wrap items-center justify-between gap-2 rounded-lg border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700">
      <span>{{ error }}</span>
      <Button size="sm" variant="outline" @click="load">重试</Button>
    </div>
    <div v-if="loading" class="panel text-[#4a5c50]">正在加载书架...</div>
    
    <div v-else-if="!books.length" class="panel grid place-items-center py-16 text-center">
      <BookOpen class="mb-3 size-12 text-[#0f7643]" />
      <h2 class="text-xl font-extrabold text-[#0f1e14]">书架还是空的</h2>
      <p class="mt-2 text-sm text-[#4a5c50]">上传第一本书，开始您的阅读之旅。</p>
      <RouterLink class="mt-5" to="/books/upload"><Button class="bg-[#0f7643] hover:bg-[#064e2b]">上传第一本书</Button></RouterLink>
    </div>
    
    <div v-else-if="!filteredBooks.length" class="panel grid place-items-center py-16 text-center">
      <BookOpen class="mb-3 size-12 text-[#4a5c50]/40" />
      <h2 class="text-lg font-bold text-[#0f1e14]">未找到匹配的书籍</h2>
      <p class="mt-1 text-sm text-[#4a5c50]">请尝试更换搜索词或筛选条件。</p>
    </div>
    
    <div v-else class="grid gap-5 md:grid-cols-1 xl:grid-cols-2">
      <article v-for="book in filteredBooks" :key="book.id" class="library-card rounded-2xl p-4 flex gap-4 items-stretch">
        <!-- Left: Book Cover 3D -->
        <div class="book-cover-wrapper">
          <div class="book-cover-3d" :style="getBookCoverStyle(book.title)">
            <div class="book-cover-title">{{ cleanCoverTitle(book.title) }}</div>
            <div class="book-cover-author">{{ book.author || '未知作者' }}</div>
          </div>
        </div>
        
        <!-- Right: Content details -->
        <div class="flex flex-col flex-1 min-w-0">
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <h2 class="line-clamp-2 text-base font-extrabold text-[#0f1e14] hover:text-[#0f7643] transition-colors leading-snug" :title="book.title">
                {{ book.title }}
              </h2>
              <p class="mt-1.5 text-xs font-semibold text-[#4a5c50] flex items-center gap-1.5">
                <span>{{ book.author || '未知作者' }}</span>
                <span class="text-emerald-500/30">•</span>
                <span>{{ book.page_count || 0 }} 页</span>
              </p>
            </div>
            <Badge :tone="statusTone(book.parse_status)" class="shrink-0 text-[10px] px-2 py-0.5 font-bold rounded-md">
              {{ statusText(book.parse_status) }}
            </Badge>
          </div>
          
          <p class="mt-3 line-clamp-2 text-xs text-[#4a5c50]/90 leading-relaxed">
            {{ book.description || book.parse_error || '暂无简介。该书籍解析完成即可在阅读页面查阅其目录及内容。' }}
          </p>
          
          <div class="mt-auto pt-3 flex items-center justify-between border-t border-emerald-500/5">
            <div class="flex gap-2">
              <RouterLink :to="`/books/${book.id}/read`">
                <Button size="sm" class="h-8 rounded-lg px-3 text-xs bg-[#0f7643] hover:bg-[#064e2b]">
                  <BookOpen class="size-3.5 mr-1" />
                  阅读
                </Button>
              </RouterLink>
              <RouterLink :to="`/books/${book.id}/info`">
                <Button size="sm" variant="outline" class="h-8 rounded-lg px-3 text-xs border-emerald-500/10 hover:bg-emerald-50 text-[#0f7643]">
                  <Info class="size-3.5 mr-1" />
                  详情
                </Button>
              </RouterLink>
            </div>
            <Button 
              size="sm" 
              variant="ghost" 
              class="h-8 rounded-lg px-2 text-xs text-red-600 hover:bg-red-50 hover:text-red-700" 
              @click="requestRemove(book)"
            >
              <Trash2 class="size-3.5 mr-1" />
              删除
            </Button>
          </div>
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
