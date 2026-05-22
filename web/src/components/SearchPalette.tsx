import { useEffect, useRef, useState } from 'react'
import { FileText, Search } from 'lucide-react'
import { api } from '../api/client'
import type { SearchResult } from '../types'

interface SearchPaletteProps {
  onSelect: (path: string) => void
  onClose: () => void
}

/** A command-palette style overlay for indexed filename search. */
export function SearchPalette({ onSelect, onClose }: SearchPaletteProps) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[]>([])
  const [active, setActive] = useState(0)
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement | null>(null)
  const listRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  // Debounced query against the server-side index.
  useEffect(() => {
    const q = query.trim()
    if (!q) {
      setResults([])
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    const timer = window.setTimeout(() => {
      api
        .search(q, 50)
        .then((res) => {
          if (cancelled) return
          setResults(res.results)
          setActive(0)
        })
        .catch(() => {
          if (!cancelled) setResults([])
        })
        .finally(() => {
          if (!cancelled) setLoading(false)
        })
    }, 120)
    return () => {
      cancelled = true
      window.clearTimeout(timer)
    }
  }, [query])

  // Keep the active row visible as the selection moves.
  useEffect(() => {
    const el = listRef.current?.querySelector<HTMLElement>('.search-result.active')
    el?.scrollIntoView({ block: 'nearest' })
  }, [active, results])

  return (
    <div className="palette-backdrop" onMouseDown={onClose}>
      <div className="palette" onMouseDown={(e) => e.stopPropagation()}>
        <div className="palette-input-row">
          <Search size={18} />
          <input
            ref={inputRef}
            className="palette-input"
            placeholder="Search markdown files by name…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Escape') {
                onClose()
              } else if (e.key === 'ArrowDown') {
                e.preventDefault()
                setActive((i) => Math.min(i + 1, results.length - 1))
              } else if (e.key === 'ArrowUp') {
                e.preventDefault()
                setActive((i) => Math.max(i - 1, 0))
              } else if (e.key === 'Enter') {
                e.preventDefault()
                const hit = results[active]
                if (hit) onSelect(hit.path)
              }
            }}
          />
        </div>

        <div className="palette-results" ref={listRef}>
          {!query.trim() && (
            <div className="palette-hint">Type to search the indexed file names.</div>
          )}
          {query.trim() && !loading && results.length === 0 && (
            <div className="palette-hint">No matching files.</div>
          )}
          {results.map((r, i) => (
            <button
              type="button"
              key={r.path}
              className={`search-result${i === active ? ' active' : ''}`}
              onMouseEnter={() => setActive(i)}
              onClick={() => onSelect(r.path)}
            >
              <FileText size={15} className="search-result-icon" />
              <span className="search-result-name">{r.name}</span>
              <span className="search-result-path">{r.path}</span>
            </button>
          ))}
        </div>

        <div className="palette-footer">
          <span>
            <kbd>↑</kbd>
            <kbd>↓</kbd> navigate
          </span>
          <span>
            <kbd>↵</kbd> open
          </span>
          <span>
            <kbd>esc</kbd> close
          </span>
        </div>
      </div>
    </div>
  )
}
