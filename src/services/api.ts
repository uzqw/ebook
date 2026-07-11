import { currentUser, pb } from '@/services/pocketbase'
import type { BookPageRecord, BookRecord, BookmarkRecord, NoteRecord, ReadingRecord } from '@/types/models'

const requireUser = () => {
  const user = currentUser()
  if (!user) throw new Error('Not authenticated')
  return user
}

export const authApi = {
  login(email: string, password: string) {
    return pb.collection('users').authWithPassword(email, password)
  },
  async register(name: string, email: string, password: string) {
    await pb.collection('users').create({ name, email, password, passwordConfirm: password })
    return this.login(email, password)
  },
}

export const booksApi = {
  list() {
    const user = requireUser()
    return pb.collection('books').getFullList<BookRecord>({ filter: `user = "${user.id}"`, sort: '-updated' })
  },
  detail(id: string) {
    return pb.collection('books').getOne<BookRecord>(id)
  },
  upload(payload: { title: string; author?: string; description?: string; file: File }) {
    const user = requireUser()
    const form = new FormData()
    form.append('title', payload.title)
    form.append('author', payload.author || '')
    form.append('description', payload.description || '')
    form.append('parse_status', 'pending')
    form.append('current_page', '1')
    form.append('user', user.id)
    form.append('file', payload.file)
    return pb.collection('books').create<BookRecord>(form)
  },
  async waitForParse(id: string, options?: { timeoutMs?: number; intervalMs?: number }) {
    const timeoutMs = options?.timeoutMs ?? 180000
    const intervalMs = options?.intervalMs ?? 1500
    const startedAt = Date.now()
    let latest = await this.detail(id)
    while (Date.now() - startedAt < timeoutMs) {
      if (latest.parse_status === 'completed' || latest.parse_status === 'failed') {
        return latest
      }
      await new Promise((resolve) => setTimeout(resolve, intervalMs))
      latest = await this.detail(id)
    }
    return latest
  },
  update(id: string, payload: Partial<BookRecord>) {
    return pb.collection('books').update<BookRecord>(id, payload)
  },
  remove(id: string) {
    return pb.collection('books').delete(id)
  },
  fileUrl(book: BookRecord) {
    return pb.files.getURL(book, book.file)
  },
  pageImageUrl(bookId: string, page: number) {
    return `${pb.baseUrl}/api/books/${bookId}/pages/${page}/image?token=${encodeURIComponent(pb.authStore.token)}`
  },
  pageHtmlUrl(bookId: string, page: number) {
    return `${pb.baseUrl}/api/books/${bookId}/pages/${page}/html?token=${encodeURIComponent(pb.authStore.token)}`
  },
  fontUrl() {
    return `${pb.baseUrl}/api/fonts/DroidSansFallback.ttf`
  },
}

export const pagesApi = {
  list(bookId: string) {
    return pb.collection('book_pages').getFullList<BookPageRecord>({ filter: `book = "${bookId}"`, sort: 'page_number' })
  },
}

export const bookmarksApi = {
  list(bookId: string) {
    const user = requireUser()
    return pb.collection('bookmarks').getFullList<BookmarkRecord>({ filter: `book = "${bookId}" && user = "${user.id}"`, sort: 'page_number' })
  },
  create(bookId: string, page: number, title: string, note = '') {
    const user = requireUser()
    return pb.collection('bookmarks').create<BookmarkRecord>({ book: bookId, user: user.id, page_number: page, title, note })
  },
  remove(id: string) { return pb.collection('bookmarks').delete(id) },
}

export const notesApi = {
  list(bookId: string) {
    const user = requireUser()
    return pb.collection('notes').getFullList<NoteRecord>({ filter: `book = "${bookId}" && user = "${user.id}"`, sort: '-updated' })
  },
  create(bookId: string, page: number, content: string) {
    const user = requireUser()
    return pb.collection('notes').create<NoteRecord>({ book: bookId, user: user.id, page_number: page, content })
  },
  update(id: string, content: string) { return pb.collection('notes').update<NoteRecord>(id, { content }) },
  remove(id: string) { return pb.collection('notes').delete(id) },
}

export const readingApi = {
  recordIdCache: new Map<string, string>(),
  async upsert(bookId: string, page: number, totalPages = 1, readSeconds = 0) {
    const user = requireUser()
    const progress = totalPages > 0 ? Math.min(1, Math.max(0, page / totalPages)) : 0
    const cacheKey = `${user.id}:${bookId}`
    const collection = pb.collection('reading_records')

    const updateRecord = (recordId: string, previousReadSeconds = 0) =>
      collection.update<ReadingRecord>(recordId, {
        page_number: page,
        progress,
        read_seconds: previousReadSeconds + readSeconds,
      })

    const cachedRecordId = this.recordIdCache.get(cacheKey)
    if (cachedRecordId) {
      try {
        return await updateRecord(cachedRecordId)
      } catch (error) {
        this.recordIdCache.delete(cacheKey)
        if (!(error instanceof Error && 'status' in error && (error as { status?: number }).status === 404)) {
          throw error
        }
      }
    }

    try {
      const existing = await collection.getFirstListItem<ReadingRecord>(`book = "${bookId}" && user = "${user.id}"`)
      this.recordIdCache.set(cacheKey, existing.id)
      return updateRecord(existing.id, existing.read_seconds)
    } catch (error) {
      if (error instanceof Error && 'status' in error && (error as { status?: number }).status !== 404) {
        throw error
      }
      const created = await collection.create<ReadingRecord>({ book: bookId, user: user.id, page_number: page, progress, read_seconds: readSeconds })
      this.recordIdCache.set(cacheKey, created.id)
      return created
    }
  },
  list() {
    const user = requireUser()
    return pb.collection('reading_records').getFullList<ReadingRecord>({ filter: `user = "${user.id}"`, sort: '-updated', expand: 'book' })
  },
}
