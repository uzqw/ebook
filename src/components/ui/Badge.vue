<script setup lang="ts">
defineProps<{ tone?: 'green' | 'amber' | 'slate' | 'red' }>()
</script>
<template>
  <span :class="[
    'inline-flex items-center rounded-full px-2.5 py-1 text-xs font-extrabold',
    tone === 'amber' ? 'bg-amber-50 text-amber-700 border border-amber-200' : tone === 'red' ? 'bg-red-50 text-red-700 border border-red-200' : tone === 'slate' ? 'bg-slate-100 text-slate-600 border border-slate-200' : 'bg-green-50 text-green-700 border border-green-200'
  ]"><slot /></span>
</template>
