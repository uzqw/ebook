<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, BookmarkPlus, BookMarked, ChevronLeft, ChevronRight, Info, ListTree, Loader2, NotebookPen, RefreshCw, RotateCcw, X, ZoomIn, ZoomOut } from '@lucide/vue'
import { bookmarksApi, booksApi, notesApi, pagesApi, readingApi } from '@/services/api'
import { installCachedCjkFont } from '@/services/font-cache'
import type { BookPageRecord, BookRecord, BookmarkRecord, BookTocItem, NoteRecord } from '@/types/models'
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
const zoom = ref(1)
const panX = ref(0)
const panY = ref(0)
const dragging = ref(false)
const dragStart = ref({ x: 0, y: 0, panX: 0, panY: 0 })
const suppressClick = ref(false)
const startedAt = Date.now()
const parsePollTimer = ref<number | null>(null)
const canAutoSave = ref(false)
const saveTimer = ref<number | null>(null)
const saveQueued = ref(false)
const readerFrame = ref<HTMLIFrameElement | null>(null)
const pageHtml = ref('')
const pageHtmlLoading = ref(false)
let pageHtmlRequestId = 0

const chromeVisible = ref(true)
const chromeTimer = ref<number | null>(null)

interface TocDisplayItem { title: string; page: number; level: number }

const currentPage = computed(() => pages.value.find((item) => item.page_number === page.value))
const pageCount = computed(() => book.value?.page_count || pages.value.length || 1)
const canRenderPage = computed(() => book.value?.parse_status === 'completed')
const iframeHeight = ref(800)
function handleMessage(event: MessageEvent) {
  if (event.data && event.data.type === 'ebook-reader-page-height') {
    iframeHeight.value = event.data.height
  }
}
async function applyCachedFontToFrame() {
  const frame = readerFrame.value
  const doc = frame?.contentDocument
  if (!frame || !doc) return
  try {
    await installCachedCjkFont(doc, true)
    window.setTimeout(() => frame.contentWindow?.dispatchEvent(new Event('resize')), 0)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '字体加载失败'
  }
}
async function loadPageHtml() {
  const currentBook = book.value
  if (!canRenderPage.value || !currentBook) {
    pageHtml.value = ''
    return
  }
  const requestId = ++pageHtmlRequestId
  pageHtmlLoading.value = true
  error.value = ''
  try {
    const html = await booksApi.fetchPageHtml(currentBook.id, page.value)
    if (requestId === pageHtmlRequestId) pageHtml.value = html
  } catch (err) {
    if (requestId === pageHtmlRequestId) {
      pageHtml.value = ''
      error.value = err instanceof Error ? err.message : '页面加载失败'
    }
  } finally {
    if (requestId === pageHtmlRequestId) pageHtmlLoading.value = false
  }
}
const tocItems = computed<TocDisplayItem[]>(() => {
  const result: TocDisplayItem[] = []
  const walk = (items: BookTocItem[] = [], fallbackLevel = 1) => {
    for (const item of items) {
      result.push({ title: item.title || `第 ${item.page} 页`, page: item.page || 1, level: item.level || fallbackLevel })
      if (item.children?.length) walk(item.children, (item.level || fallbackLevel) + 1)
    }
  }
  walk(book.value?.toc || [])
  return result
})
const pageIndexItems = computed(() => pages.value.map((item) => ({
  title: item.text?.replace(/\s+/g, ' ').trim().slice(0, 42) || `第 ${item.page_number} 页`,
  page: item.page_number,
  level: 1,
})))
const navigationItems = computed(() => tocItems.value.length ? tocItems.value : pageIndexItems.value)
const navigationTitle = computed(() => tocItems.value.length ? '文档目录' : '页面索引')
const sidePanelTitle = computed(() => activeSidePanel.value === 'index' ? navigationTitle.value : activeSidePanel.value === 'bookmarks' ? '书签' : activeSidePanel.value === 'notes' ? '笔记' : '')
const imageStyle = computed(() => ({ transform: `translate(${panX.value}px, ${panY.value}px) scale(${zoom.value})` }))
function clampPage(target: number) {
  return Math.min(pageCount.value, Math.max(1, target))
}
const statusText = (status: string) => ({ pending: '待解析', processing: '解析中', completed: '已解析', failed: '解析失败' }[status] || status)
function initialPage() {
  const queryPage = Number(route.query.page)
  return clampPage(Number.isFinite(queryPage) && queryPage >= 1 ? queryPage : book.value?.current_page || 1)
}

function showChrome() {
  chromeVisible.value = true
  if (chromeTimer.value !== null) window.clearTimeout(chromeTimer.value)
  chromeTimer.value = window.setTimeout(() => {
    if (!activeSidePanel.value) chromeVisible.value = false
  }, 2500)
}
function toggleChrome() {
  if (chromeVisible.value) {
    chromeVisible.value = false
    if (chromeTimer.value !== null) {
      window.clearTimeout(chromeTimer.value)
      chromeTimer.value = null
    }
  } else {
    showChrome()
  }
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
    await loadPageHtml()
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
            await loadPageHtml()
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
  resetZoom()
}
function prev() { goToPage(page.value - 1) }
function next() { goToPage(page.value + 1) }
function jumpToPage() {
  const target = Number(pageJump.value)
  if (Number.isFinite(target) && target >= 1) goToPage(target)
  pageJump.value = ''
}
function isEditableTarget(target: EventTarget | null) {
  const el = target as HTMLElement | null
  return !!el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.tagName === 'SELECT' || el.isContentEditable)
}
function onGlobalKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && activeSidePanel.value) {
    activeSidePanel.value = null
    return
  }
  if (isEditableTarget(event.target)) return
  if (event.key === 'ArrowLeft' || event.key === 'PageUp') {
    event.preventDefault()
    showChrome()
    prev()
  } else if (event.key === 'ArrowRight' || event.key === 'PageDown' || event.key === ' ') {
    event.preventDefault()
    showChrome()
    next()
  }
}
function toggleSidePanel(panel: 'index' | 'bookmarks' | 'notes') { activeSidePanel.value = activeSidePanel.value === panel ? null : panel }
function resetZoom() { zoom.value = 1; panX.value = 0; panY.value = 0; dragging.value = false }
function zoomIn() { zoom.value = Math.min(4, Number((zoom.value + 0.25).toFixed(2))) }
function zoomOut() { zoom.value = Math.max(1, Number((zoom.value - 0.25).toFixed(2))); if (zoom.value === 1) resetZoom() }
function startDrag(event: PointerEvent) {
  if (zoom.value <= 1) return
  suppressClick.value = false
  dragging.value = true
  dragStart.value = { x: event.clientX, y: event.clientY, panX: panX.value, panY: panY.value }
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}
function onDrag(event: PointerEvent) {
  if (!dragging.value) return
  const deltaX = event.clientX - dragStart.value.x
  const deltaY = event.clientY - dragStart.value.y
  if (Math.abs(deltaX) + Math.abs(deltaY) > 4) suppressClick.value = true
  panX.value = dragStart.value.panX + deltaX
  panY.value = dragStart.value.panY + deltaY
}
function endDrag() { dragging.value = false }
function onPageClick() {
  if (suppressClick.value) {
    suppressClick.value = false
    return
  }
  toggleChrome()
}
async function saveProgress() {
  if (!book.value) return
  const safePage = clampPage(page.value)
  if (safePage !== page.value) {
    page.value = safePage
  }
  await booksApi.update(book.value.id, { current_page: safePage, last_read_at: new Date().toISOString() } as Partial<BookRecord>)
  await readingApi.upsert(book.value.id, safePage, pageCount.value, Math.round((Date.now() - startedAt) / 1000))
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
    await bookmarksApi.create(book.value.id, page.value, `第 ${page.value} 页`, currentPage.value?.text?.slice(0, 80) || '')
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
  void loadPageHtml()
  if (String(page.value) !== String(route.query.page || '')) {
    void router.replace({ query: { ...route.query, page: String(page.value) } })
  }
})
watch(activeSidePanel, (panel) => {
  if (panel) {
    chromeVisible.value = true
    if (chromeTimer.value !== null) {
      window.clearTimeout(chromeTimer.value)
      chromeTimer.value = null
    }
  } else {
    showChrome()
  }
})
onMounted(() => {
  void load()
  showChrome()
  window.addEventListener('message', handleMessage)
  window.addEventListener('keydown', onGlobalKeydown)
})
onBeforeUnmount(() => {
  if (parsePollTimer.value !== null) window.clearInterval(parsePollTimer.value)
  if (saveTimer.value !== null) window.clearTimeout(saveTimer.value)
  if (chromeTimer.value !== null) window.clearTimeout(chromeTimer.value)
  if (saveQueued.value) {
    void saveProgress().catch((err) => {
      error.value = err instanceof Error ? err.message : '保存阅读进度失败'
    })
  }
  window.removeEventListener('message', handleMessage)
  window.removeEventListener('keydown', onGlobalKeydown)
})
</script>

<template>
  <section class="min-h-dvh" @mousemove="showChrome">
    <div v-if="error" role="alert" class="fixed left-1/2 top-4 z-[60] flex w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 items-center justify-between gap-2 rounded-xl border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700 shadow-lg">
      <span>{{ error }}</span>
      <Button size="sm" variant="outline" @click="load">重试</Button>
    </div>

    <Transition name="reader-chrome">
      <header v-if="chromeVisible" class="reader-chrome reader-chrome--top fixed left-1/2 top-3 z-40 flex w-[calc(100%-1.5rem)] max-w-5xl -translate-x-1/2 flex-wrap items-center gap-x-1 gap-y-2 px-2 py-1.5">
        <Button variant="ghost" size="sm" @click="router.push('/books')"><ArrowLeft data-icon="inline-start" />书架</Button>
        <strong class="min-w-0 flex-1 truncate px-1 text-sm text-[#142217]">{{ book?.title || '书籍阅读' }}</strong>
        <Badge v-if="book" :tone="book.parse_status === 'completed' ? 'green' : book.parse_status === 'failed' ? 'red' : 'amber'">{{ statusText(book.parse_status) }}</Badge>
        <div class="flex items-center">
          <Button variant="ghost" size="sm" class="px-2" title="刷新" aria-label="刷新" @click="load"><RefreshCw data-icon="inline-start" /></Button>
          <Button v-if="book" variant="ghost" size="sm" class="px-2" title="书籍信息" aria-label="书籍信息" @click="router.push(`/books/${book.id}/info`)"><Info data-icon="inline-start" /></Button>
        </div>
        <div class="flex items-center">
          <Button variant="ghost" size="sm" class="px-2" title="缩小" aria-label="缩小" @click="zoomOut"><ZoomOut data-icon="inline-start" /></Button>
          <span class="w-10 text-center text-xs font-bold text-[#384c3d]">{{ Math.round(zoom * 100) }}%</span>
          <Button variant="ghost" size="sm" class="px-2" title="放大" aria-label="放大" @click="zoomIn"><ZoomIn data-icon="inline-start" /></Button>
          <Button variant="ghost" size="sm" class="px-2" title="重置缩放" aria-label="重置缩放" @click="resetZoom"><RotateCcw data-icon="inline-start" /></Button>
        </div>
        <div class="flex items-center gap-1">
          <Button variant="ghost" size="sm" :class="activeSidePanel === 'index' ? 'bg-[#dcebdc]' : ''" :aria-pressed="activeSidePanel === 'index'" @click="toggleSidePanel('index')"><ListTree data-icon="inline-start" />目录</Button>
          <Button variant="ghost" size="sm" :class="activeSidePanel === 'bookmarks' ? 'bg-[#dcebdc]' : ''" :aria-pressed="activeSidePanel === 'bookmarks'" @click="toggleSidePanel('bookmarks')"><BookMarked data-icon="inline-start" />书签</Button>
          <Button variant="ghost" size="sm" :class="activeSidePanel === 'notes' ? 'bg-[#dcebdc]' : ''" :aria-pressed="activeSidePanel === 'notes'" @click="toggleSidePanel('notes')"><NotebookPen data-icon="inline-start" />笔记</Button>
        </div>
      </header>
    </Transition>

    <div v-if="loading" class="reader-column px-3 pt-24">
      <div class="panel flex items-center gap-2 text-[#384c3d]"><Loader2 class="size-4 animate-spin" />正在打开书籍...</div>
    </div>

    <main v-else-if="book" class="reader-column px-3 pb-32 pt-20">
      <div v-if="book.parse_status !== 'completed'" class="panel mb-4 text-sm text-[#384c3d]">解析尚未完成。若刚上传，请稍后刷新；失败时可查看书籍信息里的错误。</div>
      <div
        v-if="canRenderPage"
        class="reader-page reader-image-frame relative"
        :class="{ 'reader-image-frame--zoomed': zoom > 1, 'reader-image-frame--dragging': dragging }"
        @pointerdown="startDrag"
        @pointermove="onDrag"
        @pointerup="endDrag"
        @pointercancel="endDrag"
        @pointerleave="endDrag"
        @click="onPageClick"
      >
        <iframe
          ref="readerFrame"
          :srcdoc="pageHtml"
          title="书页内容"
          sandbox="allow-scripts allow-same-origin"
          class="w-full border-0 bg-white"
          :style="{ height: iframeHeight + 'px', ...imageStyle }"
          :class="{ 'pointer-events-none': zoom > 1 || dragging }"
          @load="applyCachedFontToFrame"
        ></iframe>
        <div v-if="pageHtmlLoading" class="absolute inset-0 z-10 flex items-center justify-center gap-2 bg-white/75 text-sm font-semibold text-[#384c3d]"><Loader2 class="size-4 animate-spin" />正在加载页面...</div>
      </div>
      <div v-else class="panel text-sm text-[#384c3d]">解析完成后将显示页面图片，请稍后刷新。</div>
      <article v-if="currentPage?.text" class="panel prose-page mx-auto mt-6">
        <h2 class="mb-3 text-lg font-extrabold text-[#142217]">本页文本</h2>
        {{ currentPage.text }}
      </article>
    </main>

    <Transition name="reader-chrome">
      <div v-if="chromeVisible && book && !loading" class="reader-chrome reader-chrome--bottom fixed bottom-4 left-1/2 z-40 flex -translate-x-1/2 items-center gap-1.5 px-2 py-1.5">
        <Button variant="ghost" size="sm" class="px-2" :disabled="page <= 1" aria-label="上一页" title="上一页" @click="prev"><ChevronLeft data-icon="inline-start" /></Button>
        <strong class="whitespace-nowrap px-1 text-xs text-[#142217]">{{ page }} / {{ pageCount }}</strong>
        <Button variant="ghost" size="sm" class="px-2" :disabled="page >= pageCount" aria-label="下一页" title="下一页" @click="next"><ChevronRight data-icon="inline-start" /></Button>
        <input v-model="pageJump" type="number" min="1" :max="pageCount" class="h-8 w-16 rounded-md border border-input bg-white px-2 text-center text-xs" :placeholder="String(page)" aria-label="跳转到页码" @keyup.enter="jumpToPage" />
      </div>
    </Transition>

    <Transition name="reader-scrim">
      <div v-if="activeSidePanel" class="fixed inset-0 z-40 bg-slate-950/20" @click="activeSidePanel = null" />
    </Transition>

    <Transition name="reader-drawer">
      <aside v-if="activeSidePanel" class="fixed inset-y-0 right-0 z-50 flex w-[min(340px,92vw)] flex-col border-l border-[#cbe0bf] bg-[#f8faf4] p-4 shadow-2xl">
        <div class="mb-3 flex items-center justify-between gap-2">
          <h2 class="font-extrabold text-[#142217]">{{ sidePanelTitle }}</h2>
          <Button variant="ghost" size="sm" class="px-2" title="收起" aria-label="收起面板" @click="activeSidePanel = null"><X data-icon="inline-start" /></Button>
        </div>
        <div class="min-h-0 flex-1 overflow-y-auto">
          <template v-if="activeSidePanel === 'index'">
            <div class="mb-3 flex items-center justify-between gap-2">
              <p class="text-xs text-[#384c3d]">{{ tocItems.length ? '电子书内置目录' : '由页面文本生成' }}</p>
              <Badge tone="slate">{{ navigationItems.length }}</Badge>
            </div>
            <div v-if="!navigationItems.length" class="text-sm text-[#384c3d]">解析完成后显示目录或页面索引。</div>
            <div v-else class="flex flex-col gap-1.5">
              <button
                v-for="item in navigationItems"
                :key="`${item.page}-${item.title}`"
                class="rounded-lg border border-transparent px-3 py-2 text-left text-sm transition hover:border-[#cbe0bf] hover:bg-[#edf3e8]"
                :class="item.page === page ? 'border-[#15803d] bg-[#edf3e8] font-extrabold text-[#14532d]' : 'text-[#384c3d]'"
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
              <Button size="sm" @click="addBookmark"><BookmarkPlus data-icon="inline-start" />添加当前页</Button>
            </div>
            <div v-if="!bookmarks.length" class="text-sm text-[#384c3d]">暂无书签</div>
            <div v-else class="flex flex-col gap-2">
              <button v-for="mark in bookmarks" :key="mark.id" class="block w-full rounded-lg border border-[#cbe0bf] bg-white p-3 text-left text-sm hover:bg-[#edf3e8]" @click="goToPage(mark.page_number)">
                <strong>第 {{ mark.page_number }} 页</strong>
                <p class="line-clamp-2 text-[#384c3d]">{{ mark.note }}</p>
              </button>
            </div>
          </template>

          <template v-else>
            <Textarea v-model="noteText" placeholder="记录这一页的想法..." />
            <Button class="mt-2 w-full" size="sm" @click="addNote">保存笔记</Button>
            <div class="mt-4 flex flex-col gap-2">
              <article v-for="note in notes" :key="note.id" class="rounded-lg border border-[#cbe0bf] bg-white p-3 text-sm">
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
