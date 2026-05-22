// Shared types mirroring the mdtree JSON API. See docs/api.md.

/** A single item in a directory listing. */
export interface TreeEntry {
  name: string
  path: string
  type: 'dir' | 'file'
  size: number
  modTime: string
}

/** The markdown-filtered contents of one directory. */
export interface Listing {
  path: string
  parent: string
  entries: TreeEntry[]
}

/** A markdown file's metadata. */
export interface FileInfo {
  path: string
  name: string
  size: number
  modTime: string
}

/** A markdown file together with its text content. */
export interface FileContent extends FileInfo {
  content: string
}

/** A single filename search hit. */
export interface SearchResult {
  name: string
  path: string
  score: number
}

/** The response from the search endpoint. */
export interface SearchResponse {
  query: string
  count: number
  results: SearchResult[]
}

/** Runtime metrics and index statistics from /api/stats. */
export interface Stats {
  metrics: {
    uptimeSeconds: number
    requests: number
    requestErrors: number
    avgLatencyMs: number
    fileReads: number
    fileWrites: number
    searches: number
    indexedFiles: number
    lastIndexAt?: string
    lastIndexBuildMs: number
  }
  index: {
    files: number
    builtAt: string
    buildMillis: number
  }
  sessions: number
}

/** A file open in the editor, tracking saved vs. current content. */
export interface OpenFile {
  path: string
  name: string
  /** Content as last loaded or saved — the baseline for the dirty check. */
  original: string
  /** Current editor content. */
  content: string
  modTime: string
}

/** Editor layout modes. */
export type ViewMode = 'edit' | 'split' | 'preview'

/** Colour themes. */
export type Theme = 'dark' | 'light'
