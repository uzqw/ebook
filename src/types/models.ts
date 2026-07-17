export type ParseStatus = 'pending' | 'processing' | 'completed' | 'failed'

export interface UserRecord {
  id: string
  email: string
  name?: string
}

export interface BookTocItem {
  title: string
  page: number
  level: number
  children?: BookTocItem[]
}

export interface BookRecord {
  id: string
  collectionId: string
  title: string
  author?: string
  description?: string
  file: string
  cover?: string
  page_count?: number
  parse_status: ParseStatus
  parse_error?: string
  current_page?: number
  toc?: BookTocItem[]
  last_read_at?: string
  user: string
  created: string
  updated: string
}

export interface BookPageRecord {
  id: string
  book: string
  page_number: number
  text?: string
  width?: number
  height?: number
}

export interface BookmarkRecord {
  id: string
  book: string
  user: string
  page_number: number
  title: string
  note?: string
  created: string
}

export interface NoteRecord {
  id: string
  book: string
  user: string
  page_number: number
  content: string
  created: string
  updated: string
}

export interface ReadingRecord {
  id: string
  book: string
  user: string
  page_number: number
  progress: number
  read_seconds: number
  updated: string
  expand?: { book?: BookRecord }
}
