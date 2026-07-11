<script setup lang="ts">
import { computed, watch } from 'vue'
import Button from '@/components/ui/Button.vue'

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  description?: string
  confirmText?: string
  cancelText?: string
  loading?: boolean
}>(), {
  description: '',
  confirmText: '确认',
  cancelText: '取消',
  loading: false,
})

const emit = defineEmits<{
  'update:open': [value: boolean]
  confirm: []
}>()

const dialogTitleId = computed(() => `alert-dialog-title-${props.title.replace(/\W+/g, '-').toLowerCase()}`)
const dialogDescriptionId = computed(() => `${dialogTitleId.value}-description`)

function close() {
  if (!props.loading) emit('update:open', false)
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') close()
}

watch(() => props.open, (open) => {
  document.body.style.overflow = open ? 'hidden' : ''
}, { immediate: true })
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity duration-150"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div v-if="open" class="fixed inset-0 bg-slate-950/50 backdrop-blur-sm" @click="close" />
    </Transition>

    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="scale-95 opacity-0"
      enter-to-class="scale-100 opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="scale-100 opacity-100"
      leave-to-class="scale-95 opacity-0"
    >
      <div
        v-if="open"
        role="alertdialog"
        aria-modal="true"
        :aria-labelledby="dialogTitleId"
        :aria-describedby="description ? dialogDescriptionId : undefined"
        class="fixed left-1/2 top-1/2 w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-background p-6 text-foreground shadow-lg"
        tabindex="-1"
        @keydown="onKeydown"
      >
        <div class="flex flex-col gap-2">
          <h2 :id="dialogTitleId" class="text-lg font-semibold leading-none tracking-tight">{{ title }}</h2>
          <p v-if="description" :id="dialogDescriptionId" class="text-sm text-muted-foreground">{{ description }}</p>
        </div>
        <div class="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Button variant="outline" :disabled="loading" @click="close">{{ cancelText }}</Button>
          <Button variant="destructive" :disabled="loading" @click="emit('confirm')">{{ loading ? '删除中...' : confirmText }}</Button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
