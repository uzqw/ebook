import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { installResilientFetch } from './http'

const BASE_URL = 'https://example.test'
const API_URL = `${BASE_URL}/api/health`
const LIGHT_TIMEOUT_MS = 2_000

function createMockWindow() {
  return {
    fetch: vi.fn(),
    setInterval: vi.fn(),
    location: { origin: BASE_URL } as unknown as Location,
  } as unknown as Window & typeof globalThis
}

function createMockDocument() {
  return {
    visibilityState: 'visible' as DocumentVisibilityState,
    addEventListener: vi.fn(),
  } as unknown as Document
}

describe('installResilientFetch', () => {
  let mockWindow: Window & typeof globalThis
  let mockDocument: Document
  let originalWindow: Window & typeof globalThis
  let originalDocument: Document

  beforeEach(() => {
    originalWindow = window
    originalDocument = document
    mockWindow = createMockWindow()
    mockDocument = createMockDocument()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ;(globalThis as any).window = mockWindow
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ;(globalThis as any).document = mockDocument
    vi.useFakeTimers({ shouldAdvanceTime: true })
  })

  afterEach(() => {
    vi.useRealTimers()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ;(globalThis as any).window = originalWindow
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ;(globalThis as any).document = originalDocument
  })

  it('retries after a timeout with a fresh signal', async () => {
    const signals: AbortSignal[] = []
    let calls = 0

    mockWindow.fetch = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
      calls++
      signals.push(init!.signal!)
      if (calls === 1) {
        return new Promise((_resolve, reject) => {
          init!.signal!.addEventListener('abort', () => reject(init!.signal!.reason), { once: true })
        })
      }
      return new Response('{"ok":true}', { status: 200 })
    })

    installResilientFetch(BASE_URL)

    const promise = mockWindow.fetch(API_URL)
    await vi.advanceTimersByTimeAsync(LIGHT_TIMEOUT_MS + 100)
    const response = (await promise) as Response

    expect(response.ok).toBe(true)
    expect(calls).toBe(2)
    expect(signals[0].aborted).toBe(true)
    expect(signals[1].aborted).toBe(false)
    expect(signals[0]).not.toBe(signals[1])
  })

  it('inherits method and signal from a Request object', async () => {
    const controller = new AbortController()
    let receivedInit: RequestInit | undefined

    mockWindow.fetch = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      receivedInit = init
      return new Response('{"ok":true}', { status: 200 })
    })

    installResilientFetch(BASE_URL)

    const request = new Request(API_URL, { method: 'DELETE', signal: controller.signal })
    await mockWindow.fetch(request)

    expect(receivedInit?.method).toBeUndefined()
    expect(receivedInit?.signal).toBeDefined()
    expect((receivedInit?.signal as AbortSignal).aborted).toBe(false)
  })

  it('does not retry POST requests', async () => {
    let calls = 0
    mockWindow.fetch = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit): Promise<Response> => {
      calls++
      return Promise.reject(new Error('network error'))
    })

    installResilientFetch(BASE_URL)

    await expect(mockWindow.fetch(API_URL, { method: 'POST' })).rejects.toThrow()
    expect(calls).toBe(1)
  })

  it('does not retry PATCH requests', async () => {
    let calls = 0
    mockWindow.fetch = vi.fn(async (_input: RequestInfo | URL, _init?: RequestInit): Promise<Response> => {
      calls++
      return Promise.reject(new Error('network error'))
    })

    installResilientFetch(BASE_URL)

    await expect(mockWindow.fetch(API_URL, { method: 'PATCH' })).rejects.toThrow()
    expect(calls).toBe(1)
  })

  it('passes non-API requests to the original fetch unchanged', async () => {
    const originalFetch = vi.fn(async () => new Response('ok', { status: 200 }))
    mockWindow.fetch = originalFetch

    installResilientFetch(BASE_URL)

    await mockWindow.fetch('https://other.example.com/api/health')
    expect(originalFetch).toHaveBeenCalledTimes(1)
  })
})
