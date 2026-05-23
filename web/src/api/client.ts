// Thin typed wrapper over the mdtree JSON API.
import type { FileContent, FileInfo, Listing, SearchResponse, Stats } from '../types'

/** An error carrying the HTTP status of a failed API call. */
export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

// Handler invoked when an authenticated call is rejected with 401, so the UI
// can drop back to the login screen.
let unauthorizedHandler: (() => void) | null = null

/** Register a callback fired when a session expires (a non-login 401). */
export function setUnauthorizedHandler(fn: () => void): void {
  unauthorizedHandler = fn
}

async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { 'X-Requested-With': 'mdtree' }
  const init: RequestInit = { method, headers, credentials: 'same-origin' }
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    init.body = JSON.stringify(body)
  }

  const res = await fetch(url, init)
  const text = await res.text()
  let data: unknown = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = null
    }
  }

  if (!res.ok) {
    if (res.status === 401 && !url.startsWith('/api/auth/')) {
      unauthorizedHandler?.()
    }
    const message =
      (data && typeof data === 'object' && 'error' in data
        ? String((data as { error: unknown }).error)
        : '') || res.statusText || 'request failed'
    throw new ApiError(res.status, message)
  }
  return data as T
}

function query(params: Record<string, string>): string {
  return '?' + new URLSearchParams(params).toString()
}

/** The mdtree API surface. */
export const api = {
  authStatus: () => request<{ authenticated: boolean }>('GET', '/api/auth/status'),
  login: (password: string) =>
    request<{ ok: boolean }>('POST', '/api/auth/login', { password }),
  logout: () => request<{ ok: boolean }>('POST', '/api/auth/logout'),

  tree: (path?: string) =>
    request<Listing>('GET', '/api/tree' + (path ? query({ path }) : '')),
  readFile: (path: string) => request<FileContent>('GET', '/api/file' + query({ path })),
  saveFile: (path: string, content: string) =>
    request<FileInfo>('PUT', '/api/file', { path, content }),
  createFile: (path: string, content: string) =>
    request<FileInfo>('POST', '/api/file', { path, content }),
  deleteFile: (path: string) =>
    request<{ ok: boolean }>('DELETE', '/api/file' + query({ path })),
  renameFile: (from: string, to: string) =>
    request<FileInfo>('POST', '/api/file/rename', { from, to }),
  mkdir: (path: string) => request<{ path: string }>('POST', '/api/dir', { path }),
  deleteDir: (path: string) =>
    request<{ ok: boolean }>('DELETE', '/api/dir' + query({ path })),
  renameDir: (from: string, to: string) =>
    request<{ path: string }>('POST', '/api/dir/rename', { from, to }),

  search: (q: string, limit = 50) =>
    request<SearchResponse>('GET', '/api/search' + query({ q, limit: String(limit) })),
  reindex: () =>
    request<{ files: number; durationMs: number }>('POST', '/api/search/reindex'),
  stats: () => request<Stats>('GET', '/api/stats'),
}
