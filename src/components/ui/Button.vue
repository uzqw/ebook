<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'
const props = withDefaults(defineProps<{ variant?: 'default' | 'secondary' | 'outline' | 'ghost' | 'destructive'; size?: 'sm' | 'md' | 'lg'; class?: string; type?: 'button' | 'submit' | 'reset' }>(), { variant: 'default', size: 'md', type: 'button' })
const classes = computed(() => cn(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-lg font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50',
  props.size === 'sm' && 'h-9 px-3 text-xs',
  props.size === 'md' && 'h-11 px-4 text-sm',
  props.size === 'lg' && 'h-12 px-5 text-base',
  props.variant === 'default' && 'bg-primary text-primary-foreground hover:bg-[#166534] shadow-sm',
  props.variant === 'secondary' && 'bg-secondary text-secondary-foreground hover:bg-[#d8e3cc]',
  props.variant === 'outline' && 'border border-border bg-white/80 text-[#384c3d] hover:bg-[#edf3e8] hover:text-[#14532d]',
  props.variant === 'ghost' && 'text-[#384c3d] hover:bg-[#edf3e8] hover:text-[#14532d]',
  props.variant === 'destructive' && 'bg-destructive text-destructive-foreground hover:bg-red-800',
  props.class,
))
</script>
<template><button :type="type" :class="classes"><slot /></button></template>
