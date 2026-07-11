<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { readingApi } from '@/services/api'
import type { ReadingRecord } from '@/types/models'
const records = ref<ReadingRecord[]>([])
const totalBooks = computed(() => records.value.length)
const totalSeconds = computed(() => records.value.reduce((sum, r) => sum + (r.read_seconds || 0), 0))
onMounted(async () => { records.value = await readingApi.list() })
</script>
<template><section><div class="page-header"><div><p class="text-xs font-extrabold uppercase tracking-widest text-[#705c21]">Summary</p><h1 class="text-3xl font-extrabold text-[#142217]">阅读记录汇总</h1></div></div><div class="grid gap-4 md:grid-cols-3"><div class="page-card"><p class="text-xs font-bold uppercase text-[#384c3d]">阅读书籍</p><strong class="mt-2 block text-3xl">{{ totalBooks }}</strong></div><div class="page-card"><p class="text-xs font-bold uppercase text-[#384c3d]">累计阅读分钟</p><strong class="mt-2 block text-3xl">{{ Math.round(totalSeconds / 60) }}</strong></div><div class="page-card"><p class="text-xs font-bold uppercase text-[#384c3d]">平均进度</p><strong class="mt-2 block text-3xl">{{ records.length ? Math.round(records.reduce((s, r) => s + r.progress, 0) / records.length * 100) : 0 }}%</strong></div></div><div class="mt-5 page-card"><h2 class="mb-3 text-lg font-extrabold">最近记录</h2><div v-if="!records.length" class="text-sm text-[#384c3d]">暂无阅读记录</div><div v-for="record in records" :key="record.id" class="flex items-center justify-between border-b border-[#e2ead8] py-3 text-sm last:border-0"><span>书籍 {{ record.book }} · 第 {{ record.page_number }} 页</span><strong>{{ Math.round(record.progress * 100) }}%</strong></div></div></section></template>
