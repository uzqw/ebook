<script setup lang="ts">
defineProps<{ modelValue?: string; placeholder?: string; rows?: number; class?: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>
<template>
  <textarea :value="modelValue" :placeholder="placeholder" :rows="rows || 4" :class="['w-full rounded-lg border border-input bg-white px-3 py-2 text-sm shadow-sm outline-none transition-colors placeholder:text-slate-400 focus:border-primary focus:ring-2 focus:ring-primary/15', $props.class]" @input="$emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)" />
</template>
