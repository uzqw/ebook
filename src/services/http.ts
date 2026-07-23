// Resilient fetch wrapper for API requests.
//
// Browsers reuse a single HTTP/2 connection per origin. When that connection
// dies silently (laptop suspend, WireGuard/NAT state expiring after a long
// idle), every request written into it black-holes until TCP retransmission
// gives up (15+ minutes). A short timeout + automatic retry forces a fresh
// connection and recovers in seconds, invisible to the user.
//
// NOTE: this wrapper only protects in-page `fetch()` calls. It cannot help
// the initial page load, because the browser's connection pool for navigation
// requests is outside JS control. For that, the reverse proxy must avoid
// keeping half-open connections alive (e.g. force HTTP/1.1 + Connection: close).

const LIGHT_TIMEOUT_MS = 2_000
const HEAVY_TIMEOUT_MS = 120_000
const MAX_RETRIES = 2
const RETRY_DELAY_MS = 400
const HEARTBEAT_INTERVAL_MS = 30_000

// Requests that are legitimately slow: font downloads, server-rendered book
// pages, file uploads. These get a generous timeout and no short fuse.
const HEAVY_PATHS = ['/api/fonts/', '/pages/']

// Safe to re-send: reading twice or writing the same value twice is harmless.
const IDEMPOTENT_METHODS = new Set(['GET', 'HEAD', 'PUT', 'PATCH', 'DELETE'])

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

export function installResilientFetch(baseUrl: string) {
  const originalFetch = window.fetch.bind(window)
  const apiPrefix = `${baseUrl}/api/`

  const resolveUrl = (input: RequestInfo | URL) =>
    typeof input === 'string' ? input : input instanceof URL ? input.href : input.url

  const isHeavy = (url: string, init?: RequestInit) =>
    init?.body instanceof FormData || HEAVY_PATHS.some((path) => url.includes(path))

  async function request(input: RequestInfo | URL, init: RequestInit | undefined, attempt: number): Promise<Response> {
    const url = resolveUrl(input)
    const method = (init?.method ?? 'GET').toUpperCase()
    const timeoutMs = isHeavy(url, init) ? HEAVY_TIMEOUT_MS : LIGHT_TIMEOUT_MS
    const timeoutSignal = AbortSignal.timeout(timeoutMs)
    const signal = init?.signal ? AbortSignal.any([init.signal, timeoutSignal]) : timeoutSignal

    try {
      return await originalFetch(input, { ...init, signal })
    } catch (error) {
      const callerAborted = init?.signal?.aborted === true
      const bodyConsumed = input instanceof Request && input.body !== null
      const canRetry = !callerAborted && !bodyConsumed && attempt < MAX_RETRIES && IDEMPOTENT_METHODS.has(method)
      if (!canRetry) throw error
      await sleep(RETRY_DELAY_MS * (attempt + 1))
      // Avoid serving a stale cached response on retry; force a real network round-trip.
      const retryInit = { ...init, cache: 'no-store' as RequestCache, signal }
      return request(input, retryInit, attempt + 1)
    }
  }

  window.fetch = (input, init) => {
    if (!resolveUrl(input).startsWith(apiPrefix)) return originalFetch(input, init)
    return request(input, init, 0)
  }

  // Keep the connection alive: NAT mappings, WireGuard sessions and conntrack
  // entries all expire silently after idle periods. Any periodic traffic
  // refreshes that state before it can die, so the failure never happens.
  // Errors are swallowed — the timeout+retry in request() handles recovery.
  // Use cache-busting so a stale 200 from the browser cache does not hide a
  // dead connection.
  const probe = () => {
    void window.fetch(`${baseUrl}/api/health`, { cache: 'no-store' }).catch(() => {})
  }

  window.setInterval(() => {
    if (document.visibilityState === 'visible') probe()
  }, HEARTBEAT_INTERVAL_MS)

  // Back to the tab after suspend/idle: probe now, before the user's next
  // interaction, so any reconnect happens off the critical path.
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') probe()
  })

  // Network switch (wifi <-> ethernet, VPN up/down): previously pooled
  // connections are almost certainly dead. Probe immediately to force a
  // fresh connection instead of letting the next user request discover it.
  const connection = (navigator as { connection?: EventTarget }).connection
  connection?.addEventListener('change', probe)
}
