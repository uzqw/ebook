<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { Loader2, BookOpen, Clock, TrendingUp, History } from '@lucide/vue'
import { readingApi } from '@/services/api'
import type { ReadingRecord } from '@/types/models'
import Button from '@/components/ui/Button.vue'

const records = ref<ReadingRecord[]>([])
const loading = ref(true)
const error = ref('')
const totalBooks = computed(() => records.value.length)
const totalSeconds = computed(() => records.value.reduce((sum, r) => sum + (r.read_seconds || 0), 0))

async function load() {
  loading.value = true
  error.value = ''
  try {
    records.value = await readingApi.list()
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
        <p class="text-xs font-extrabold uppercase tracking-widest text-[#0f7643]">Summary</p>
        <h1 class="text-3xl font-black text-[#0f1e14] tracking-tight">阅读记录汇总</h1>
        <p class="mt-1 text-sm text-[#4a5c50]">追踪并统计您的阅读偏好、时长以及每本书的阅读进度。</p>
      </div>
    </div>

    <div v-if="error" role="alert" class="mb-4 flex flex-wrap items-center justify-between gap-2 rounded-lg border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700">
      <span>{{ error }}</span>
      <Button size="sm" variant="outline" @click="load">重试</Button>
    </div>

    <div v-if="loading" class="panel flex items-center gap-2 text-[#4a5c50]">
      <Loader2 class="size-4 animate-spin text-[#0f7643]" />正在加载阅读记录...
    </div>

    <template v-else>
      <!-- Stats Dashboard Grid -->
      <div class="grid gap-4 md:grid-cols-3">
        <div class="library-card rounded-2xl p-5 flex items-center gap-4">
          <div class="p-3.5 rounded-xl bg-emerald-50 text-[#0f7643] border border-emerald-500/5">
            <BookOpen class="size-6" />
          </div>
          <div>
            <p class="text-xs font-extrabold uppercase tracking-wider text-[#4a5c50]">阅读书籍</p>
            <strong class="mt-0.5 block text-2xl font-black text-[#0f1e14]">{{ totalBooks }} <span class="text-xs font-semibold text-[#4a5c50]">本</span></strong>
          </div>
        </div>

        <div class="library-card rounded-2xl p-5 flex items-center gap-4">
          <div class="p-3.5 rounded-xl bg-emerald-50 text-[#0f7643] border border-emerald-500/5">
            <Clock class="size-6" />
          </div>
          <div>
            <p class="text-xs font-extrabold uppercase tracking-wider text-[#4a5c50]">累计阅读时长</p>
            <strong class="mt-0.5 block text-2xl font-black text-[#0f1e14]">{{ Math.round(totalSeconds / 60) }} <span class="text-xs font-semibold text-[#4a5c50]">分钟</span></strong>
          </div>
        </div>

        <div class="library-card rounded-2xl p-5 flex items-center gap-4">
          <div class="p-3.5 rounded-xl bg-emerald-50 text-[#0f7643] border border-emerald-500/5">
            <TrendingUp class="size-6" />
          </div>
          <div>
            <p class="text-xs font-extrabold uppercase tracking-wider text-[#4a5c50]">平均阅读进度</p>
            <strong class="mt-0.5 block text-2xl font-black text-[#0f1e14]">
              {{ records.length ? Math.round(records.reduce((s, r) => s + r.progress, 0) / records.length * 100) : 0 }}%
            </strong>
          </div>
        </div>
      </div>

      <!-- Recent Records Section -->
      <div class="mt-6 library-card rounded-2xl p-5">
        <div class="flex items-center gap-2 mb-4 pb-2 border-b border-emerald-500/5">
          <History class="size-5 text-[#0f7643] opacity-80" />
          <h2 class="text-base font-extrabold text-[#0f1e14]">最近阅读记录</h2>
        </div>

        <div v-if="!records.length" class="text-xs text-[#4a5c50]/60 italic py-2">
          暂无阅读记录。打开一本电子书，开始记录您的阅读之旅吧。
        </div>

        <div v-else class="divide-y divide-emerald-500/5">
          <div v-for="record in records" :key="record.id" class="flex flex-col sm:flex-row sm:items-center justify-between py-3.5 gap-3 text-sm">
            <RouterLink 
              class="hover:text-[#0f7643] font-bold text-[#0f1e14] transition-colors truncate max-w-lg" 
              :to="`/books/${record.book}/read?page=${record.page_number}`"
            >
              《{{ record.expand?.book?.title || '未知书籍' }}》 · 第 {{ record.page_number }} 页
            </RouterLink>
            
            <!-- Beautiful Horizontal Progress Bar -->
            <div class="flex items-center gap-3 w-full sm:w-48 shrink-0">
              <div class="w-full bg-[#f1f5f3] h-2 rounded-full overflow-hidden border border-emerald-500/5">
                <div 
                  class="bg-[#0f7643] h-full rounded-full transition-all duration-500" 
                  :style="{ width: `${Math.round(record.progress * 100)}%` }"
                ></div>
              </div>
              <strong class="w-10 text-right text-xs font-black text-[#0f1e14]">{{ Math.round(record.progress * 100) }}%</strong>
            </div>
          </div>
        </div>
      </div>
    </template>
  </section>
</template>
