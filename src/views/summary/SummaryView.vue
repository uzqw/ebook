<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { Loader2 } from '@lucide/vue'
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
  <section>
    <div class="page-header">
      <div>
        <p class="text-xs font-extrabold uppercase tracking-widest text-[#705c21]">Summary</p>
        <h1 class="text-3xl font-extrabold text-[#142217]">阅读记录汇总</h1>
      </div>
    </div>
    <div v-if="error" role="alert" class="mb-4 flex flex-wrap items-center justify-between gap-2 rounded-lg border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700">
      <span>{{ error }}</span>
      <Button size="sm" variant="outline" @click="load">重试</Button>
    </div>
    <div v-if="loading" class="panel flex items-center gap-2 text-[#384c3d]"><Loader2 class="size-4 animate-spin" />正在加载阅读记录...</div>
    <template v-else>
      <div class="grid gap-4 md:grid-cols-3">
        <div class="page-card"><p class="text-xs font-bold uppercase text-[#384c3d]">阅读书籍</p><strong class="mt-2 block text-3xl">{{ totalBooks }}</strong></div>
        <div class="page-card"><p class="text-xs font-bold uppercase text-[#384c3d]">累计阅读分钟</p><strong class="mt-2 block text-3xl">{{ Math.round(totalSeconds / 60) }}</strong></div>
        <div class="page-card"><p class="text-xs font-bold uppercase text-[#384c3d]">平均进度</p><strong class="mt-2 block text-3xl">{{ records.length ? Math.round(records.reduce((s, r) => s + r.progress, 0) / records.length * 100) : 0 }}%</strong></div>
      </div>
      <div class="mt-5 page-card">
        <h2 class="mb-3 text-lg font-extrabold">最近记录</h2>
        <div v-if="!records.length" class="text-sm text-[#384c3d]">暂无阅读记录</div>
        <div v-for="record in records" :key="record.id" class="flex items-center justify-between border-b border-[#e2ead8] py-3 text-sm last:border-0">
          <RouterLink class="hover:underline" :to="`/books/${record.book}/read?page=${record.page_number}`">《{{ record.expand?.book?.title || '未知书籍' }}》 · 第 {{ record.page_number }} 页</RouterLink>
          <strong>{{ Math.round(record.progress * 100) }}%</strong>
        </div>
      </div>
    </template>
  </section>
</template>
