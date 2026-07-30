<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  BookmarkPlus,
  BookMarked,
  BookOpenText,
  ChevronLeft,
  ChevronRight,
  Info,
  ListTree,
  Loader2,
  NotebookPen,
  Palette,
  RefreshCw,
  RotateCcw,
  X,
  ZoomIn,
  ZoomOut,
} from '@lucide/vue'
import { bookmarksApi, booksApi, notesApi, pagesApi, readingApi } from '@/services/api'
import type { PageIllustration } from '@/services/api'
import { parseLayoutPreference, reflowText, resolveLayoutMode } from '@/lib/reflow'
import type { LayoutPreference } from '@/lib/reflow'
import type {
  BookPageRecord,
  BookRecord,
  BookmarkRecord,
  BookTocItem,
  NoteRecord,
} from '@/types/models'
import Button from '@/components/ui/Button.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Badge from '@/components/ui/Badge.vue'

const props = defineProps<{ id: string }>()
const router = useRouter()
const route = useRoute()
const book = ref<BookRecord | null>(null)
const pages = ref<BookPageRecord[]>([])
const bookmarks = ref<BookmarkRecord[]>([])
const notes = ref<NoteRecord[]>([])
const page = ref(1)
const pageJump = ref('')
const noteText = ref('')
const activeSidePanel = ref<'index' | 'bookmarks' | 'notes' | null>(null)
const loading = ref(true)
const error = ref('')
const ZOOM_STORAGE_KEY = 'ebook-reader-zoom'
const LAYOUT_STORAGE_KEY = 'ebook-reader-layout'
const THEME_STORAGE_KEY = 'ebook-reader-theme'

type ReaderTheme = 'paper' | 'ink' | 'midnight' | 'coffee' | 'oled'
interface ReaderThemeOption {
  id: ReaderTheme
  name: string
  description: string
  page: string
  text: string
}

const readerThemes: ReaderThemeOption[] = [
  { id: 'paper', name: '纸张', description: '明亮', page: '#ffffff', text: '#142217' },
  { id: 'ink', name: '墨夜', description: '柔和黑', page: '#1b211d', text: '#dbe5dc' },
  { id: 'midnight', name: '深海', description: '深蓝', page: '#131c2b', text: '#dbe7f4' },
  { id: 'coffee', name: '夜棕', description: '暖棕', page: '#29201b', text: '#eadaca' },
  { id: 'oled', name: '纯黑', description: '省电', page: '#050505', text: '#d8d8d8' },
]

function storedReaderTheme(): ReaderTheme {
  const value = window.localStorage.getItem(THEME_STORAGE_KEY)
  return readerThemes.some((theme) => theme.id === value) ? (value as ReaderTheme) : 'paper'
}

function storedZoom() {
  const value = Number(window.localStorage.getItem(ZOOM_STORAGE_KEY))
  return Number.isFinite(value) && value >= 1 && value <= 4 ? value : 1
}
function storedLayoutPreference(): LayoutPreference {
  return parseLayoutPreference(window.localStorage.getItem(LAYOUT_STORAGE_KEY))
}
const narrowViewportQuery = window.matchMedia('(max-width: 768px)')
const narrowViewport = ref(narrowViewportQuery.matches)
const layoutPreference = ref<LayoutPreference>(storedLayoutPreference())
const effectiveLayoutMode = computed(() =>
  resolveLayoutMode(layoutPreference.value, narrowViewport.value),
)
const reflowEnabled = computed(() => effectiveLayoutMode.value === 'reflow')
const zoom = ref(storedZoom())
const readerTheme = ref<ReaderTheme>(storedReaderTheme())
const themeMenuOpen = ref(false)
const startedAt = Date.now()
const parsePollTimer = ref<number | null>(null)
const canAutoSave = ref(false)
const saveTimer = ref<number | null>(null)
const saveQueued = ref(false)
const pageImageUrl = ref('')
const pageImageLoading = ref(false)
const pageIllustrations = ref<PageIllustration[]>([])
const illustrationsLoading = ref(false)
const expandedImage = ref('')
const expandedImageScale = ref(1)
const expandedImageWidth = ref(0)
let pinchStartDistance = 0
let pinchStartScale = 1
let pageImageRequestId = 0
let illustrationsRequestId = 0

const chromeVisible = ref(true)
const pageViewport = ref<HTMLElement | null>(null)
let touchStartX = 0
let touchStartY = 0

function isInteractiveReaderTarget(target: EventTarget | null) {
  const el = target as HTMLElement | null
  return !!el?.closest('a, button, input, textarea, select, aside, .reader-chrome')
}

function toggleChrome() {
  chromeVisible.value = !chromeVisible.value
}

function openExpandedImage(src: string) {
  expandedImage.value = src
  expandedImageScale.value = 1
  expandedImageWidth.value = Math.min(1100, window.innerWidth - 32)
}

function closeExpandedImage() {
  expandedImage.value = ''
  pinchStartDistance = 0
}

function touchDistance(touches: TouchList) {
  const x = touches[0].clientX - touches[1].clientX
  const y = touches[0].clientY - touches[1].clientY
  return Math.hypot(x, y)
}

function onImageTouchStart(event: TouchEvent) {
  if (event.touches.length !== 2) return
  event.preventDefault()
  pinchStartDistance = touchDistance(event.touches)
  pinchStartScale = expandedImageScale.value
}

function onImageTouchMove(event: TouchEvent) {
  if (event.touches.length !== 2 || !pinchStartDistance) return
  event.preventDefault()
  expandedImageScale.value = Math.min(
    5,
    Math.max(1, pinchStartScale * (touchDistance(event.touches) / pinchStartDistance)),
  )
}

function onImageTouchEnd(event: TouchEvent) {
  if (event.touches.length < 2) pinchStartDistance = 0
}

function handlePageTap(event: MouseEvent) {
  if (!narrowViewport.value || isInteractiveReaderTarget(event.target)) return
  if (!window.getSelection()?.isCollapsed) return

  const bounds = pageViewport.value?.getBoundingClientRect()
  if (!bounds) return
  const relativeX = (event.clientX - bounds.left) / Math.max(bounds.width, 1)
  if (relativeX < 0.22) {
    prev()
  } else if (relativeX > 0.78) {
    next()
  } else {
    toggleChrome()
  }
}

function onTouchStart(event: TouchEvent) {
  if (!narrowViewport.value || isInteractiveReaderTarget(event.target)) return
  const touch = event.changedTouches[0]
  if (!touch) return
  touchStartX = touch.clientX
  touchStartY = touch.clientY
}

function onTouchEnd(event: TouchEvent) {
  if (!narrowViewport.value || isInteractiveReaderTarget(event.target)) return
  const touch = event.changedTouches[0]
  if (!touch) return

  const deltaX = touch.clientX - touchStartX
  const deltaY = touch.clientY - touchStartY
  if (Math.abs(deltaX) < 64 || Math.abs(deltaX) <= Math.abs(deltaY) * 1.25) return
  event.preventDefault()
  if (deltaX < 0) {
    next()
  } else {
    prev()
  }
}

interface TocDisplayItem {
  title: string
  page: number
  level: number
}

const currentPage = computed(() => pages.value.find((item) => item.page_number === page.value))
const reflowBlocks = computed(() => reflowText(currentPage.value?.text))
const reflowItems = computed(() => {
  const count = reflowBlocks.value.length
  return [
    ...reflowBlocks.value.map((block, index) => ({ ...block, order: (index + 1) / (count + 1) })),
    ...pageIllustrations.value.map((image, index) => ({
      kind: 'image' as const,
      image,
      order: image.top + index / 10000,
    })),
  ].sort((a, b) => a.order - b.order)
})
const pageCount = computed(() => book.value?.page_count || pages.value.length || 1)
const canRenderPage = computed(() => book.value?.parse_status === 'completed')
const pageImageHeight = ref(800)
function clearPageImage() {
  if (pageImageUrl.value) URL.revokeObjectURL(pageImageUrl.value)
  pageImageUrl.value = ''
}
async function loadPageImage() {
  const requestId = ++pageImageRequestId
  const currentBook = book.value
  if (!canRenderPage.value || !currentBook || reflowEnabled.value) {
    clearPageImage()
    pageImageLoading.value = false
    return
  }
  pageImageLoading.value = true
  error.value = ''
  try {
    const blob = await booksApi.fetchPageImage(currentBook.id, page.value)
    if (requestId === pageImageRequestId) {
      clearPageImage()
      pageImageUrl.value = URL.createObjectURL(blob)
    }
  } catch (err) {
    if (requestId === pageImageRequestId) {
      clearPageImage()
      error.value = err instanceof Error ? err.message : '页面加载失败'
    }
  } finally {
    if (requestId === pageImageRequestId) pageImageLoading.value = false
  }
}
async function loadPageIllustrations() {
  const requestId = ++illustrationsRequestId
  const currentBook = book.value
  if (!canRenderPage.value || !currentBook || !reflowEnabled.value) {
    pageIllustrations.value = []
    illustrationsLoading.value = false
    return
  }
  illustrationsLoading.value = true
  try {
    const images = await booksApi.fetchPageIllustrations(currentBook.id, page.value)
    if (requestId === illustrationsRequestId) pageIllustrations.value = images
  } catch (err) {
    if (requestId === illustrationsRequestId) {
      pageIllustrations.value = []
      error.value = err instanceof Error ? err.message : '页面插图加载失败'
    }
  } finally {
    if (requestId === illustrationsRequestId) illustrationsLoading.value = false
  }
}
async function loadPageMedia() {
  await Promise.all([loadPageImage(), loadPageIllustrations()])
}
function onPageImageLoad(event: Event) {
  pageImageHeight.value = (event.currentTarget as HTMLImageElement).clientHeight
}
const tocItems = computed<TocDisplayItem[]>(() => {
  const result: TocDisplayItem[] = []
  const walk = (items: BookTocItem[] = [], fallbackLevel = 1) => {
    for (const item of items) {
      result.push({
        title: item.title || `第 ${item.page} 页`,
        page: item.page || 1,
        level: item.level || fallbackLevel,
      })
      if (item.children?.length) walk(item.children, (item.level || fallbackLevel) + 1)
    }
  }
  walk(book.value?.toc || [])
  return result
})
const pageIndexItems = computed(() =>
  pages.value.map((item) => ({
    title: item.text?.replace(/\s+/g, ' ').trim().slice(0, 42) || `第 ${item.page_number} 页`,
    page: item.page_number,
    level: 1,
  })),
)
const navigationItems = computed(() =>
  tocItems.value.length ? tocItems.value : pageIndexItems.value,
)
const navigationTitle = computed(() => (tocItems.value.length ? '文档目录' : '页面索引'))
const sidePanelTitle = computed(() =>
  activeSidePanel.value === 'index'
    ? navigationTitle.value
    : activeSidePanel.value === 'bookmarks'
      ? '书签'
      : activeSidePanel.value === 'notes'
        ? '笔记'
        : '',
)
function clampPage(target: number) {
  return Math.min(pageCount.value, Math.max(1, target))
}
const statusText = (status: string) =>
  ({ pending: '待解析', processing: '解析中', completed: '已解析', failed: '解析失败' })[status] ||
  status
function initialPage() {
  const queryPage = Number(route.query.page)
  return clampPage(
    Number.isFinite(queryPage) && queryPage >= 1 ? queryPage : book.value?.current_page || 1,
  )
}

function updateChromeVisibility(clientY: number) {
  if (narrowViewport.value) return
  if (activeSidePanel.value) {
    chromeVisible.value = true
    return
  }
  const viewportHeight = window.innerHeight
  if (clientY < 72 || clientY > viewportHeight - 96) {
    chromeVisible.value = true
  } else {
    chromeVisible.value = false
  }
}
function onSectionMousemove(event: MouseEvent) {
  if (narrowViewport.value) return
  updateChromeVisibility(event.clientY)
}
function onWindowMouseLeave() {
  if (narrowViewport.value || activeSidePanel.value) return
  chromeVisible.value = false
}

async function load() {
  if (parsePollTimer.value !== null) {
    window.clearInterval(parsePollTimer.value)
    parsePollTimer.value = null
  }
  canAutoSave.value = false
  saveQueued.value = false
  if (saveTimer.value !== null) {
    window.clearTimeout(saveTimer.value)
    saveTimer.value = null
  }
  loading.value = true
  error.value = ''
  try {
    book.value = await booksApi.detail(props.id)
    pages.value = await pagesApi.list(props.id)
    bookmarks.value = await bookmarksApi.list(props.id)
    notes.value = await notesApi.list(props.id)
    page.value = initialPage()
    await loadPageMedia()
    if (book.value.parse_status === 'pending' || book.value.parse_status === 'processing') {
      parsePollTimer.value = window.setInterval(async () => {
        try {
          const latest = await booksApi.detail(props.id)
          book.value = latest
          if (latest.parse_status === 'completed' || latest.parse_status === 'failed') {
            if (parsePollTimer.value !== null) {
              window.clearInterval(parsePollTimer.value)
              parsePollTimer.value = null
            }
            pages.value = await pagesApi.list(props.id)
            bookmarks.value = await bookmarksApi.list(props.id)
            notes.value = await notesApi.list(props.id)
            await loadPageMedia()
          }
        } catch (err) {
          error.value = err instanceof Error ? err.message : '刷新解析状态失败'
        }
      }, 2000)
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
    canAutoSave.value = true
  }
}
function goToPage(target: number) {
  page.value = clampPage(target)
}
function prev() {
  goToPage(page.value - 1)
}
function next() {
  goToPage(page.value + 1)
}
function jumpToPage() {
  const target = Number(pageJump.value)
  if (Number.isFinite(target) && target >= 1) goToPage(target)
  pageJump.value = ''
}
function isEditableTarget(target: EventTarget | null) {
  const el = target as HTMLElement | null
  return (
    !!el &&
    (el.tagName === 'INPUT' ||
      el.tagName === 'TEXTAREA' ||
      el.tagName === 'SELECT' ||
      el.isContentEditable)
  )
}
function onGlobalKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && expandedImage.value) {
    closeExpandedImage()
    return
  }
  if (event.key === 'Escape' && activeSidePanel.value) {
    activeSidePanel.value = null
    return
  }
  if (isEditableTarget(event.target)) return
  if (event.key === 'ArrowLeft' || event.key === 'PageUp') {
    event.preventDefault()
    prev()
  } else if (event.key === 'ArrowRight' || event.key === 'PageDown' || event.key === ' ') {
    event.preventDefault()
    next()
  }
}
function toggleSidePanel(panel: 'index' | 'bookmarks' | 'notes') {
  activeSidePanel.value = activeSidePanel.value === panel ? null : panel
}
function resetZoom() {
  zoom.value = 1
}
function selectReaderTheme(theme: ReaderTheme) {
  readerTheme.value = theme
  themeMenuOpen.value = false
}
function toggleLayoutMode() {
  layoutPreference.value = reflowEnabled.value ? 'original' : 'reflow'
  window.localStorage.setItem(LAYOUT_STORAGE_KEY, layoutPreference.value)
}
function showOriginalPage() {
  layoutPreference.value = 'original'
  window.localStorage.setItem(LAYOUT_STORAGE_KEY, layoutPreference.value)
}
function zoomIn() {
  zoom.value = Math.min(4, Number((zoom.value + 0.25).toFixed(2)))
}
function zoomOut() {
  zoom.value = Math.max(1, Number((zoom.value - 0.25).toFixed(2)))
}
function onSectionClick(event: MouseEvent) {
  if (narrowViewport.value) return
  const target = event.target as HTMLElement
  if (
    target.closest('.reader-page') ||
    target.closest('.reader-chrome') ||
    target.closest('aside')
  ) {
    return
  }
  const clickX = event.clientX
  const middleX = window.innerWidth / 2
  if (clickX < middleX) {
    prev()
  } else {
    next()
  }
}
async function saveProgress() {
  if (!book.value) return
  const safePage = clampPage(page.value)
  if (safePage !== page.value) {
    page.value = safePage
  }
  await booksApi.update(book.value.id, {
    current_page: safePage,
    last_read_at: new Date().toISOString(),
  } as Partial<BookRecord>)
  await readingApi.upsert(
    book.value.id,
    safePage,
    pageCount.value,
    Math.round((Date.now() - startedAt) / 1000),
  )
}
function scheduleSaveProgress() {
  if (!canAutoSave.value || !book.value) return
  saveQueued.value = true
  if (saveTimer.value !== null) {
    window.clearTimeout(saveTimer.value)
  }
  saveTimer.value = window.setTimeout(async () => {
    saveTimer.value = null
    if (!saveQueued.value) return
    saveQueued.value = false
    try {
      await saveProgress()
    } catch (err) {
      error.value = err instanceof Error ? err.message : '保存阅读进度失败'
    }
  }, 350)
}
async function addBookmark() {
  if (!book.value) return
  try {
    await bookmarksApi.create(
      book.value.id,
      page.value,
      `第 ${page.value} 页`,
      currentPage.value?.text?.slice(0, 80) || '',
    )
    bookmarks.value = await bookmarksApi.list(book.value.id)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '添加书签失败'
  }
}
async function addNote() {
  if (!book.value || !noteText.value.trim()) return
  try {
    await notesApi.create(book.value.id, page.value, noteText.value.trim())
    noteText.value = ''
    notes.value = await notesApi.list(book.value.id)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存笔记失败'
  }
}

watch(page, () => {
  scheduleSaveProgress()
  closeExpandedImage()
  if (!loading.value) void loadPageMedia()
  if (narrowViewport.value) {
    window.scrollTo({ top: 0, left: 0, behavior: 'auto' })
  }
  if (String(page.value) !== String(route.query.page || '')) {
    void router.replace({ query: { ...route.query, page: String(page.value) } })
  }
})
watch(zoom, (value) => {
  window.localStorage.setItem(ZOOM_STORAGE_KEY, String(value))
})
watch(readerTheme, (value) => {
  window.localStorage.setItem(THEME_STORAGE_KEY, value)
})
watch(effectiveLayoutMode, () => {
  void loadPageMedia()
})
watch(activeSidePanel, (panel) => {
  if (panel) {
    chromeVisible.value = true
  }
})
function onViewportChange(event: MediaQueryListEvent) {
  narrowViewport.value = event.matches
}

onMounted(() => {
  void load()
  narrowViewportQuery.addEventListener('change', onViewportChange)
  window.addEventListener('keydown', onGlobalKeydown)
  window.addEventListener('mouseleave', onWindowMouseLeave)
})
onBeforeUnmount(() => {
  pageImageRequestId += 1
  illustrationsRequestId += 1
  if (parsePollTimer.value !== null) window.clearInterval(parsePollTimer.value)
  if (saveTimer.value !== null) window.clearTimeout(saveTimer.value)
  if (saveQueued.value) {
    void saveProgress().catch((err) => {
      error.value = err instanceof Error ? err.message : '保存阅读进度失败'
    })
  }
  narrowViewportQuery.removeEventListener('change', onViewportChange)
  clearPageImage()
  window.removeEventListener('keydown', onGlobalKeydown)
  window.removeEventListener('mouseleave', onWindowMouseLeave)
})
</script>

<template>
  <section
    class="reader-surface min-h-dvh select-none"
    :data-reader-theme="readerTheme"
    @mousemove="onSectionMousemove"
    @click="onSectionClick"
  >
    <div
      v-if="error"
      role="alert"
      class="fixed left-1/2 top-4 z-[60] flex w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 items-center justify-between gap-2 rounded-xl border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700 shadow-lg"
    >
      <span>{{ error }}</span>
      <Button size="sm" variant="outline" @click="load">重试</Button>
    </div>

    <Transition name="reader-chrome">
      <header
        v-if="chromeVisible"
        class="reader-chrome reader-chrome--top fixed left-1/2 top-3 z-40 flex w-[calc(100%-1.5rem)] max-w-5xl -translate-x-1/2 flex-wrap items-center gap-x-1 gap-y-2 px-2 py-1.5"
      >
        <Button variant="ghost" size="sm" @click="router.push('/books')"
          ><ArrowLeft data-icon="inline-start" />书架</Button
        >
        <strong class="reader-title min-w-0 flex-1 truncate px-1 text-sm text-[#142217]">{{
          book?.title || '书籍阅读'
        }}</strong>
        <Button
          variant="ghost"
          size="sm"
          class="reader-theme-trigger px-2 md:hidden"
          title="阅读主题"
          aria-label="选择阅读主题"
          :aria-expanded="themeMenuOpen"
          @click="themeMenuOpen = !themeMenuOpen"
          ><Palette data-icon="inline-start"
        /></Button>
        <Badge
          v-if="book"
          class="reader-status"
          :tone="
            book.parse_status === 'completed'
              ? 'green'
              : book.parse_status === 'failed'
                ? 'red'
                : 'amber'
          "
          >{{ statusText(book.parse_status) }}</Badge
        >
        <div class="reader-tools flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            class="px-2"
            title="刷新"
            aria-label="刷新"
            @click="load"
            ><RefreshCw data-icon="inline-start"
          /></Button>
          <Button
            v-if="book"
            variant="ghost"
            size="sm"
            class="px-2"
            title="书籍信息"
            aria-label="书籍信息"
            @click="router.push(`/books/${book.id}/info`)"
            ><Info data-icon="inline-start"
          /></Button>
        </div>
        <div class="flex items-center">
          <Button
            variant="ghost"
            size="sm"
            class="px-2"
            title="缩小"
            aria-label="缩小"
            @click="zoomOut"
            ><ZoomOut data-icon="inline-start"
          /></Button>
          <span class="reader-zoom-value w-10 text-center text-xs font-bold text-[#384c3d]"
            >{{ Math.round(zoom * 100) }}%</span
          >
          <Button
            variant="ghost"
            size="sm"
            class="px-2"
            title="放大"
            aria-label="放大"
            @click="zoomIn"
            ><ZoomIn data-icon="inline-start"
          /></Button>
          <Button
            variant="ghost"
            size="sm"
            class="reader-reset-zoom px-2"
            title="重置缩放"
            aria-label="重置缩放"
            @click="resetZoom"
            ><RotateCcw data-icon="inline-start"
          /></Button>
          <Button
            variant="ghost"
            size="sm"
            :class="reflowEnabled ? 'bg-[#dcebdc]' : ''"
            :aria-pressed="reflowEnabled"
            :title="reflowEnabled ? '切换到原版页面' : '切换到自适应排版'"
            @click="toggleLayoutMode"
            ><BookOpenText data-icon="inline-start" /><span class="reader-layout-label">{{
              reflowEnabled ? '自适应' : '原版'
            }}</span></Button
          >
        </div>
        <div class="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            :class="activeSidePanel === 'index' ? 'bg-[#dcebdc]' : ''"
            :aria-pressed="activeSidePanel === 'index'"
            @click="toggleSidePanel('index')"
            ><ListTree data-icon="inline-start" />目录</Button
          >
          <Button
            variant="ghost"
            size="sm"
            :class="activeSidePanel === 'bookmarks' ? 'bg-[#dcebdc]' : ''"
            :aria-pressed="activeSidePanel === 'bookmarks'"
            @click="toggleSidePanel('bookmarks')"
            ><BookMarked data-icon="inline-start" />书签</Button
          >
          <Button
            variant="ghost"
            size="sm"
            :class="activeSidePanel === 'notes' ? 'bg-[#dcebdc]' : ''"
            :aria-pressed="activeSidePanel === 'notes'"
            @click="toggleSidePanel('notes')"
            ><NotebookPen data-icon="inline-start" />笔记</Button
          >
        </div>
      </header>
    </Transition>

    <Transition name="reader-chrome">
      <div
        v-if="themeMenuOpen && narrowViewport"
        class="reader-chrome reader-theme-menu fixed left-1/2 top-[4.25rem] z-50 grid w-[calc(100%-1.5rem)] -translate-x-1/2 grid-cols-5 gap-1.5 p-2"
        role="group"
        aria-label="阅读主题"
      >
        <button
          v-for="theme in readerThemes"
          :key="theme.id"
          type="button"
          class="reader-theme-option"
          :class="{ 'reader-theme-option--active': readerTheme === theme.id }"
          :aria-pressed="readerTheme === theme.id"
          @click="selectReaderTheme(theme.id)"
        >
          <span
            class="reader-theme-swatch"
            :style="{ background: theme.page, color: theme.text }"
            aria-hidden="true"
            >文</span
          >
          <strong>{{ theme.name }}</strong>
          <small>{{ theme.description }}</small>
        </button>
      </div>
    </Transition>

    <div v-if="loading" class="reader-column px-3 pt-24">
      <div class="panel flex items-center gap-2 text-[#384c3d]">
        <Loader2 class="size-4 animate-spin" />正在打开书籍...
      </div>
    </div>

    <main
      ref="pageViewport"
      v-else-if="book"
      class="reader-column px-3 pb-32 pt-20"
      @touchstart.passive="onTouchStart"
      @touchend="onTouchEnd"
      @click="handlePageTap"
    >
      <div v-if="book.parse_status !== 'completed'" class="panel mb-4 text-sm text-[#384c3d]">
        解析尚未完成。若刚上传，请稍后刷新；失败时可查看书籍信息里的错误。
      </div>
      <article
        v-if="canRenderPage && reflowEnabled && (reflowItems.length || illustrationsLoading)"
        class="reader-page reader-reflow-page select-text"
        :style="{
          fontSize: `clamp(${16 * zoom}px, calc(${14 * zoom}px + 1vw), ${18 * zoom}px)`,
        }"
      >
        <template v-for="(item, index) in reflowItems" :key="`${item.kind}-${index}`">
          <button
            v-if="item.kind === 'image'"
            type="button"
            class="reader-reflow-image"
            aria-label="放大查看书中插图"
            @click.stop="openExpandedImage(item.image.src)"
          >
            <img :src="item.image.src" alt="书中插图" />
          </button>
          <h2 v-else-if="item.kind === 'heading'" class="reader-reflow-heading">
            {{ item.text }}
          </h2>
          <p v-else class="reader-reflow-paragraph">{{ item.text }}</p>
        </template>
        <div
          v-if="illustrationsLoading"
          class="flex items-center justify-center gap-2 py-4 text-sm"
        >
          <Loader2 class="size-4 animate-spin" />正在加载插图...
        </div>
      </article>
      <div
        v-else-if="canRenderPage && reflowEnabled"
        class="reader-page reader-reflow-page flex min-h-72 select-text flex-col items-center justify-center gap-4 text-center"
      >
        <p class="text-sm text-[#384c3d]">当前页没有可重排的文本，可能是扫描页或插图页。</p>
        <Button variant="outline" size="sm" @click="showOriginalPage">查看原版页面</Button>
      </div>
      <div
        v-else-if="canRenderPage"
        class="reader-zoom-stage"
        :style="{ height: `${pageImageHeight * zoom}px` }"
      >
        <div
          class="reader-page reader-image-frame"
          :style="{ transform: `scale(${zoom})`, transformOrigin: 'top center' }"
        >
          <button
            v-if="pageImageUrl"
            type="button"
            class="block w-full"
            aria-label="放大查看原页图片"
            @click.stop="openExpandedImage(pageImageUrl)"
          >
            <img :src="pageImageUrl" alt="原页图片" @load="onPageImageLoad" />
          </button>
          <div
            v-if="pageImageLoading"
            class="absolute inset-0 z-10 flex min-h-72 items-center justify-center gap-2 bg-white/75 text-sm font-semibold text-[#384c3d]"
          >
            <Loader2 class="size-4 animate-spin" />正在加载页面...
          </div>
        </div>
      </div>
      <div v-else class="panel text-sm text-[#384c3d]">解析完成后将显示书页，请稍后刷新。</div>
    </main>

    <Transition name="reader-scrim">
      <div
        v-if="expandedImage"
        class="reader-image-lightbox fixed inset-0 z-[60] overflow-auto bg-black/90 p-4"
        role="dialog"
        aria-modal="true"
        aria-label="图片预览"
        @click="closeExpandedImage"
      >
        <button
          type="button"
          class="fixed right-4 top-4 z-10 rounded-full bg-black/70 p-3 text-white"
          aria-label="关闭图片预览"
          @click="closeExpandedImage"
        >
          <X class="size-6" />
        </button>
        <img
          :src="expandedImage"
          alt="放大的书中图片"
          class="mx-auto max-w-none rounded-lg"
          :style="{ width: `${expandedImageWidth * expandedImageScale}px` }"
          @click.stop
          @touchstart.stop="onImageTouchStart"
          @touchmove.stop="onImageTouchMove"
          @touchend.stop="onImageTouchEnd"
          @touchcancel.stop="onImageTouchEnd"
        />
      </div>
    </Transition>

    <Transition name="reader-chrome">
      <div
        v-if="chromeVisible && book && !loading"
        class="reader-chrome reader-chrome--bottom fixed bottom-[calc(1rem+env(safe-area-inset-bottom))] left-1/2 z-40 flex -translate-x-1/2 items-center gap-1.5 px-2 py-1.5"
      >
        <Button
          variant="ghost"
          size="sm"
          class="px-2"
          :disabled="page <= 1"
          aria-label="上一页"
          title="上一页"
          @click="prev"
          ><ChevronLeft data-icon="inline-start"
        /></Button>
        <strong class="whitespace-nowrap px-1 text-xs text-[#142217]"
          >{{ page }} / {{ pageCount }}</strong
        >
        <Button
          variant="ghost"
          size="sm"
          class="px-2"
          :disabled="page >= pageCount"
          aria-label="下一页"
          title="下一页"
          @click="next"
          ><ChevronRight data-icon="inline-start"
        /></Button>
        <input
          v-model="pageJump"
          type="number"
          min="1"
          :max="pageCount"
          class="h-8 w-16 rounded-md border border-input bg-white px-2 text-center text-xs"
          :placeholder="String(page)"
          aria-label="跳转到页码"
          @keyup.enter="jumpToPage"
        />
      </div>
    </Transition>

    <Transition name="reader-scrim">
      <div
        v-if="activeSidePanel"
        class="fixed inset-0 z-40 bg-slate-950/20"
        @click="activeSidePanel = null"
      />
    </Transition>

    <Transition name="reader-drawer">
      <aside
        v-if="activeSidePanel"
        class="reader-side-panel fixed inset-y-0 right-0 z-50 flex w-[min(340px,92vw)] select-text flex-col border-l border-[#cbe0bf] bg-[#f8faf4] p-4 shadow-2xl"
      >
        <div class="mb-3 flex items-center justify-between gap-2">
          <h2 class="font-extrabold text-[#142217]">{{ sidePanelTitle }}</h2>
          <Button
            variant="ghost"
            size="sm"
            class="px-2"
            title="收起"
            aria-label="收起面板"
            @click="activeSidePanel = null"
            ><X data-icon="inline-start"
          /></Button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto">
          <template v-if="activeSidePanel === 'index'">
            <div class="mb-3 flex items-center justify-between gap-2">
              <p class="text-xs text-[#384c3d]">
                {{ tocItems.length ? '电子书内置目录' : '由页面文本生成' }}
              </p>
              <Badge tone="slate">{{ navigationItems.length }}</Badge>
            </div>
            <div v-if="!navigationItems.length" class="text-sm text-[#384c3d]">
              解析完成后显示目录或页面索引。
            </div>
            <div v-else class="flex flex-col gap-1.5">
              <button
                v-for="item in navigationItems"
                :key="`${item.page}-${item.title}`"
                class="rounded-lg border border-transparent px-3 py-2 text-left text-sm transition hover:border-[#cbe0bf] hover:bg-[#edf3e8]"
                :class="
                  item.page === page
                    ? 'border-[#15803d] bg-[#edf3e8] font-extrabold text-[#14532d]'
                    : 'text-[#384c3d]'
                "
                :style="{ paddingLeft: `${Math.min(item.level - 1, 4) * 14 + 12}px` }"
                @click="goToPage(item.page)"
              >
                <span class="block truncate">{{ item.title }}</span>
                <span class="text-xs text-[#64748b]">第 {{ item.page }} 页</span>
              </button>
            </div>
          </template>

          <template v-else-if="activeSidePanel === 'bookmarks'">
            <div class="mb-3 flex items-center justify-end">
              <Button size="sm" @click="addBookmark"
                ><BookmarkPlus data-icon="inline-start" />添加当前页</Button
              >
            </div>
            <div v-if="!bookmarks.length" class="text-sm text-[#384c3d]">暂无书签</div>
            <div v-else class="flex flex-col gap-2">
              <button
                v-for="mark in bookmarks"
                :key="mark.id"
                class="block w-full rounded-lg border border-[#cbe0bf] bg-white p-3 text-left text-sm hover:bg-[#edf3e8]"
                @click="goToPage(mark.page_number)"
              >
                <strong>第 {{ mark.page_number }} 页</strong>
                <p class="line-clamp-2 text-[#384c3d]">{{ mark.note }}</p>
              </button>
            </div>
          </template>

          <template v-else>
            <Textarea v-model="noteText" placeholder="记录这一页的想法..." />
            <Button class="mt-2 w-full" size="sm" @click="addNote">保存笔记</Button>
            <div class="mt-4 flex flex-col gap-2">
              <article
                v-for="note in notes"
                :key="note.id"
                class="rounded-lg border border-[#cbe0bf] bg-white p-3 text-sm"
              >
                <strong>第 {{ note.page_number }} 页</strong>
                <p class="mt-1 whitespace-pre-wrap text-[#384c3d]">{{ note.content }}</p>
              </article>
            </div>
          </template>
        </div>
      </aside>
    </Transition>
  </section>
</template>
