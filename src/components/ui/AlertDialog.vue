<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import Button from '@/components/ui/Button.vue'

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  description?: string
  confirmText?: string
  cancelText?: string
  loading?: boolean
  loadingText?: string
}>(), {
  description: '',
  confirmText: '确认',
  cancelText: '取消',
  loading: false,
  loadingText: '处理中...',
})

const emit = defineEmits<{
  'update:open': [value: boolean]
  confirm: []
}>()

const dialogEl = ref<HTMLElement | null>(null)
let previousActiveElement: HTMLElement | null = null

const dialogTitleId = computed(() => `alert-dialog-title-${props.title.replace(/\W+/g, '-').toLowerCase()}`)
const dialogDescriptionId = computed(() => `${dialogTitleId.value}-description`)

function close() {
  if (!props.loading) emit('update:open', false)
}

function focusableElements(): HTMLElement[] {
  if (!dialogEl.value) return []
  return Array.from(
    dialogEl.value.querySelectorAll<HTMLElement>('button, [href], input, textarea, select, [tabindex]:not([tabindex="-1"])'),
  ).filter((el) => !el.hasAttribute('disabled'))
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    close()
    return
  }
  if (event.key !== 'Tab') return
  const focusable = focusableElements()
  if (!focusable.length) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  const active = document.activeElement as HTMLElement | null
  if (event.shiftKey && (active === first || active === dialogEl.value)) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && active === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(() => props.open, async (open) => {
  document.body.style.overflow = open ? 'hidden' : ''
  if (open) {
    previousActiveElement = document.activeElement as HTMLElement | null
    await nextTick()
    dialogEl.value?.focus()
  } else {
    previousActiveElement?.focus()
    previousActiveElement = null
  }
}, { immediate: true })

onBeforeUnmount(() => {
  document.body.style.overflow = ''
})
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
      <div v-if="open" class="fixed inset-0 z-50 bg-slate-950/50 backdrop-blur-sm" @click="close" />
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
        ref="dialogEl"
        role="alertdialog"
        aria-modal="true"
        :aria-labelledby="dialogTitleId"
        :aria-describedby="description ? dialogDescriptionId : undefined"
        class="fixed left-1/2 top-1/2 z-50 w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-background p-6 text-foreground shadow-lg focus-visible:outline-none"
        tabindex="-1"
        @keydown="onKeydown"
      >
        <div class="flex flex-col gap-2">
          <h2 :id="dialogTitleId" class="text-lg font-semibold leading-none tracking-tight">{{ title }}</h2>
          <p v-if="description" :id="dialogDescriptionId" class="text-sm text-muted-foreground">{{ description }}</p>
        </div>
        <div class="mt-6 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Button variant="outline" :disabled="loading" @click="close">{{ cancelText }}</Button>
          <Button variant="destructive" :disabled="loading" @click="emit('confirm')">{{ loading ? loadingText : confirmText }}</Button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
