<script setup lang="ts">
defineProps<{ modelValue?: string | number; type?: string; placeholder?: string; required?: boolean; accept?: string; class?: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>
<template>
  <input :type="type || 'text'" :value="modelValue" :placeholder="placeholder" :required="required" :accept="accept" :class="['flex h-10 w-full rounded-lg border border-input bg-white px-3 py-2 text-sm shadow-sm outline-none transition-colors placeholder:text-slate-400 focus:border-primary focus:ring-2 focus:ring-primary/15 disabled:cursor-not-allowed disabled:opacity-50', $props.class]" @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)" />
</template>
