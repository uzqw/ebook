import { booksApi } from '@/services/api'

const CACHE_NAME = 'ebook-reader-fonts-v1'
const STYLE_ID = 'ebook-reader-cjk-font'
const FONT_FAMILY = 'EbookReaderCJK'

let fontUrlPromise: Promise<string> | null = null
let objectUrl = ''

function cssUrl(url: string) {
  return url.replace(/\\/g, '\\\\').replace(/'/g, "\\'")
}

export async function cachedCjkFontUrl() {
  if (objectUrl) return objectUrl
  fontUrlPromise ??= (async () => {
    const url = booksApi.fontUrl()
    if (!('caches' in window)) return url

    const cache = await caches.open(CACHE_NAME)
    let response = await cache.match(url)
    if (!response) {
      response = await fetch(url, { cache: 'force-cache' })
      if (!response.ok) throw new Error(`字体加载失败: ${response.status}`)
      await cache.put(url, response.clone())
    }
    objectUrl = URL.createObjectURL(await response.blob())
    return objectUrl
  })()
  return fontUrlPromise
}

export async function installCachedCjkFont(doc: Document, applyGlobally = false) {
  const url = await cachedCjkFontUrl()
  const style = doc.getElementById(STYLE_ID) || doc.createElement('style')
  style.id = STYLE_ID
  style.textContent = `@font-face{font-family:'${FONT_FAMILY}';src:url('${cssUrl(url)}');}${applyGlobally ? `html,body,body *{font-family:'${FONT_FAMILY}',sans-serif !important;}` : ''}`
  ;(doc.head || doc.documentElement).appendChild(style)
}
