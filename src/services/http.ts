// Resilient fetch wrapper for API requests.
//
// A browser connection can become stale after laptop suspend or a network
// change. A short timeout plus automatic retry bounds how long in-page API
// calls wait before making another attempt. Connection establishment remains
// under browser control, so this mitigates but cannot eliminate network delay.
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

// Requests that are legitimately slow: font downloads and file uploads. These
// get a generous timeout. Page HTML stays on the short timeout so a dead path
// is detected and retried promptly instead of waiting 120 seconds.
const HEAVY_PATHS = ['/api/fonts/']

// Requests that are safe to re-send without changing server state. PATCH is
// intentionally excluded because an incremental/append patch is not guaranteed
// to be idempotent; callers that know their PATCH is safe can retry explicitly.
const IDEMPOTENT_METHODS = new Set(['GET', 'HEAD', 'PUT', 'DELETE'])

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

function extractRequestInfo(input: RequestInfo | URL, init?: RequestInit) {
  const requestInput = input instanceof Request ? input : undefined
  const method = (init?.method ?? requestInput?.method ?? 'GET').toUpperCase()
  const callerSignal = init?.signal ?? requestInput?.signal
  return { method, callerSignal }
}

export function installResilientFetch(baseUrl: string) {
  const originalFetch = window.fetch.bind(window)
  const apiPrefix = `${baseUrl}/api/`

  const resolveUrl = (input: RequestInfo | URL) =>
    typeof input === 'string' ? input : input instanceof URL ? input.href : input.url

  const isHeavy = (url: string, init?: RequestInit) =>
    init?.body instanceof FormData || HEAVY_PATHS.some((path) => url.includes(path))

  async function request(input: RequestInfo | URL, init: RequestInit | undefined, attempt: number): Promise<Response> {
    const url = resolveUrl(input)
    const { method, callerSignal } = extractRequestInfo(input, init)
    const timeoutMs = isHeavy(url, init) ? HEAVY_TIMEOUT_MS : LIGHT_TIMEOUT_MS
    const timeoutSignal = AbortSignal.timeout(timeoutMs)
    const signal = callerSignal ? AbortSignal.any([callerSignal, timeoutSignal]) : timeoutSignal

    try {
      return await originalFetch(input, { ...init, signal })
    } catch (error) {
      const callerAborted = callerSignal?.aborted === true
      const bodyConsumed = input instanceof Request && input.body !== null
      const canRetry = !callerAborted && !bodyConsumed && attempt < MAX_RETRIES && IDEMPOTENT_METHODS.has(method)
      if (!canRetry) throw error
      await sleep(RETRY_DELAY_MS * (attempt + 1))
      // Preserve only the caller's original signal. The combined `signal`
      // above includes this attempt's timeout and is already aborted after a
      // timeout; reusing it would make the next attempt fail immediately.
      const retryInit: RequestInit = { ...init, cache: 'no-store' }
      return request(input, retryInit, attempt + 1)
    }
  }

  window.fetch = (input, init) => {
    if (!resolveUrl(input).startsWith(apiPrefix)) return originalFetch(input, init)
    return request(input, init, 0)
  }

  // Periodic traffic can refresh NAT, WireGuard and conntrack state while the
  // page is visible, reducing idle expiry. It cannot prevent every stale
  // connection (especially during suspend), so timeout+retry remains necessary.
  // Probe errors are swallowed because request() handles recovery attempts.
  // Use cache-busting so a stale 200 from the browser cache does not hide a
  // dead connection.
  const probe = () => {
    void window.fetch(`${baseUrl}/api/health`, { cache: 'no-store' }).catch(() => {})
  }

  window.setInterval(() => {
    if (document.visibilityState === 'visible') probe()
  }, HEARTBEAT_INTERVAL_MS)

  // Back to the tab after suspend/idle: probe promptly so recovery may begin
  // before the next API request on the user's critical path.
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') probe()
  })

  // Network switch (wifi <-> ethernet, VPN up/down): previously pooled
  // connections are almost certainly dead. Probe immediately to force a
  // fresh connection instead of letting the next user request discover it.
  const connection = (navigator as { connection?: EventTarget }).connection
  connection?.addEventListener('change', probe)
}
