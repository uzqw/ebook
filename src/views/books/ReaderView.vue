<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { BookmarkPlus, BookMarked, ChevronLeft, ChevronRight, Info, ListTree, Loader2, Move, NotebookPen, RefreshCw, RotateCcw, X, ZoomIn, ZoomOut } from '@lucide/vue'
import { bookmarksApi, booksApi, notesApi, pagesApi, readingApi } from '@/services/api'
import { installCachedCjkFont } from '@/services/font-cache'
import type { BookPageRecord, BookRecord, BookmarkRecord, BookTocItem, NoteRecord } from '@/types/models'
import Button from '@/components/ui/Button.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Badge from '@/components/ui/Badge.vue'

const props = defineProps<{ id: string }>()
const router = useRouter()
const book = ref<BookRecord | null>(null)
const pages = ref<BookPageRecord[]>([])
const bookmarks = ref<BookmarkRecord[]>([])
const notes = ref<NoteRecord[]>([])
const page = ref(1)
const noteText = ref('')
const activeSidePanel = ref<'index' | 'bookmarks' | 'notes' | null>(null)
const loading = ref(true)
const error = ref('')
const zoom = ref(1)
const panX = ref(0)
const panY = ref(0)
const dragging = ref(false)
const dragStart = ref({ x: 0, y: 0, panX: 0, panY: 0 })
const startedAt = Date.now()
const parsePollTimer = ref<number | null>(null)
const canAutoSave = ref(false)
const saveTimer = ref<number | null>(null)
const saveQueued = ref(false)
const readerFrame = ref<HTMLIFrameElement | null>(null)
const pageHtml = ref('')
const pageHtmlLoading = ref(false)
let pageHtmlRequestId = 0

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
const readerGridClass = computed(() => activeSidePanel.value ? 'xl:grid-cols-[minmax(0,1fr)_340px]' : 'xl:grid-cols-[minmax(0,1fr)_64px]')
const sidePanelTitle = computed(() => activeSidePanel.value === 'index' ? navigationTitle.value : activeSidePanel.value === 'bookmarks' ? '书签' : activeSidePanel.value === 'notes' ? '笔记' : '')
const imageStyle = computed(() => ({ transform: `translate(${panX.value}px, ${panY.value}px) scale(${zoom.value})` }))
function clampPage(target: number) {
  return Math.min(pageCount.value, Math.max(1, target))
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
    page.value = clampPage(book.value.current_page || 1)
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
function toggleSidePanel(panel: 'index' | 'bookmarks' | 'notes') { activeSidePanel.value = activeSidePanel.value === panel ? null : panel }
function resetZoom() { zoom.value = 1; panX.value = 0; panY.value = 0; dragging.value = false }
function zoomIn() { zoom.value = Math.min(4, Number((zoom.value + 0.25).toFixed(2))) }
function zoomOut() { zoom.value = Math.max(1, Number((zoom.value - 0.25).toFixed(2))); if (zoom.value === 1) resetZoom() }
function startDrag(event: PointerEvent) {
  if (zoom.value <= 1) return
  dragging.value = true
  dragStart.value = { x: event.clientX, y: event.clientY, panX: panX.value, panY: panY.value }
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}
function onDrag(event: PointerEvent) {
  if (!dragging.value) return
  const deltaX = event.clientX - dragStart.value.x
  const deltaY = event.clientY - dragStart.value.y
  panX.value = dragStart.value.panX + deltaX
  panY.value = dragStart.value.panY + deltaY
}
function endDrag() { dragging.value = false }
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

watch(page, () => { scheduleSaveProgress(); void loadPageHtml() })
onMounted(() => {
  void load()
  window.addEventListener('message', handleMessage)
})
onBeforeUnmount(() => {
  if (parsePollTimer.value !== null) window.clearInterval(parsePollTimer.value)
  if (saveTimer.value !== null) window.clearTimeout(saveTimer.value)
  if (saveQueued.value) {
    void saveProgress().catch((err) => {
      error.value = err instanceof Error ? err.message : '保存阅读进度失败'
    })
  }
  window.removeEventListener('message', handleMessage)
})
</script>

<template>
  <section>
    <div class="page-header">
      <div>
        <p class="text-xs font-extrabold uppercase tracking-widest text-[#705c21]">Reader</p>
        <h1 class="text-3xl font-extrabold text-[#142217]">{{ book?.title || '书籍阅读' }}</h1>
        <p class="mt-2 text-sm text-[#384c3d]">右侧按钮可展开目录、书签和笔记；页面可在当前区域缩放拖动。</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <Button variant="outline" @click="load"><RefreshCw data-icon="inline-start" />刷新</Button>
        <Button v-if="book" variant="outline" @click="router.push(`/books/${book.id}/info`)"><Info data-icon="inline-start" />书籍信息</Button>
      </div>
    </div>

    <p v-if="error" class="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm font-semibold text-red-700">{{ error }}</p>
    <div v-if="loading" class="panel flex items-center gap-2 text-[#384c3d]"><Loader2 class="size-4 animate-spin" />正在打开书籍...</div>

    <div v-else-if="book" class="grid gap-5" :class="readerGridClass">
      <main class="min-w-0">
        <div class="panel mb-4 flex flex-wrap items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <Button variant="outline" size="sm" :disabled="page <= 1" @click="prev"><ChevronLeft data-icon="inline-start" />上一页</Button>
            <strong class="text-sm text-[#142217]">第 {{ page }} / {{ pageCount }} 页</strong>
            <Button variant="outline" size="sm" :disabled="page >= pageCount" @click="next">下一页<ChevronRight data-icon="inline-end" /></Button>
          </div>
          <div class="flex items-center gap-2">
            <Button variant="outline" size="sm" @click="zoomOut"><ZoomOut data-icon="inline-start" />缩小</Button>
            <Button variant="outline" size="sm" @click="zoomIn"><ZoomIn data-icon="inline-start" />放大 {{ Math.round(zoom * 100) }}%</Button>
            <Button variant="ghost" size="sm" @click="resetZoom"><RotateCcw data-icon="inline-start" />重置</Button>
            <span v-if="zoom > 1" class="inline-flex items-center gap-1 text-xs font-bold text-[#384c3d]"><Move class="size-3.5" />拖动查看</span>
            <Badge :tone="book.parse_status === 'completed' ? 'green' : book.parse_status === 'failed' ? 'red' : 'amber'">{{ book.parse_status }}</Badge>
          </div>
        </div>
        <div v-if="book.parse_status !== 'completed'" class="panel mb-4 text-sm text-[#384c3d]">解析尚未完成。若刚上传，请稍后刷新；失败时可查看书籍信息里的错误。</div>
        <div
          v-if="canRenderPage"
          class="reader-page reader-image-frame"
          :class="{ 'reader-image-frame--zoomed': zoom > 1, 'reader-image-frame--dragging': dragging }"
          @pointerdown="startDrag"
          @pointermove="onDrag"
          @pointerup="endDrag"
          @pointercancel="endDrag"
          @pointerleave="endDrag"
        >
          <div v-if="pageHtmlLoading" class="flex items-center gap-2 p-4 text-sm font-semibold text-[#384c3d]"><Loader2 class="size-4 animate-spin" />正在加载页面...</div>
          <iframe
            v-else
            ref="readerFrame"
            :srcdoc="pageHtml"
            class="w-full border-0 bg-white"
            :style="{ height: iframeHeight + 'px', ...imageStyle }"
            :class="{ 'pointer-events-none': zoom > 1 || dragging }"
            @load="applyCachedFontToFrame"
          ></iframe>
        </div>
        <div v-else class="panel text-sm text-[#384c3d]">解析完成后将显示页面图片，请稍后刷新。</div>
        <article v-if="currentPage?.text" class="panel prose-page">
          <h2 class="mb-3 text-lg font-extrabold text-[#142217]">本页文本</h2>
          {{ currentPage.text }}
        </article>
      </main>

      <aside class="reader-side-tools">
        <div class="reader-side-buttons">
          <Button variant="outline" size="sm" class="reader-side-button" :class="activeSidePanel === 'index' ? 'bg-[#edf3e8]' : ''" title="页面索引" @click="toggleSidePanel('index')"><ListTree data-icon="inline-start" /><span>索引</span></Button>
          <Button variant="outline" size="sm" class="reader-side-button" :class="activeSidePanel === 'bookmarks' ? 'bg-[#edf3e8]' : ''" title="书签" @click="toggleSidePanel('bookmarks')"><BookMarked data-icon="inline-start" /><span>书签</span></Button>
          <Button variant="outline" size="sm" class="reader-side-button" :class="activeSidePanel === 'notes' ? 'bg-[#edf3e8]' : ''" title="笔记" @click="toggleSidePanel('notes')"><NotebookPen data-icon="inline-start" /><span>笔记</span></Button>
        </div>

        <section v-if="activeSidePanel" class="panel reader-side-panel">
          <div class="mb-3 flex items-center justify-between gap-2">
            <h2 class="font-extrabold text-[#142217]">{{ sidePanelTitle }}</h2>
            <Button variant="ghost" size="sm" class="px-2" title="收起" @click="activeSidePanel = null"><X data-icon="inline-start" /></Button>
          </div>

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
        </section>
      </aside>
    </div>

  </section>
</template>
