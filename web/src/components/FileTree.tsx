import { useEffect, useState } from 'react'
import { ChevronRight, FileText, Folder, FolderOpen, Loader2 } from 'lucide-react'
import { api } from '../api/client'
import type { TreeEntry } from '../types'

interface FileTreeProps {
  activePath: string | null
  /** Bumped by the parent to force a reload after structural changes. */
  refreshNonce: number
  onOpenFile: (path: string) => void
}

/** A lazily-loaded, markdown-only file tree rooted at the server's root. */
export function FileTree({ activePath, refreshNonce, onOpenFile }: FileTreeProps) {
  return (
    <div className="file-tree" role="tree">
      <DirNode
        path={null}
        name="/"
        depth={0}
        defaultExpanded
        activePath={activePath}
        refreshNonce={refreshNonce}
        onOpenFile={onOpenFile}
      />
    </div>
  )
}

interface DirNodeProps {
  /** Absolute directory path, or null for the configured root. */
  path: string | null
  name: string
  depth: number
  defaultExpanded?: boolean
  activePath: string | null
  refreshNonce: number
  onOpenFile: (path: string) => void
}

function DirNode({
  path,
  name,
  depth,
  defaultExpanded,
  activePath,
  refreshNonce,
  onOpenFile,
}: DirNodeProps) {
  const [expanded, setExpanded] = useState(Boolean(defaultExpanded))
  const [entries, setEntries] = useState<TreeEntry[] | null>(null)
  const [label, setLabel] = useState(name)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!expanded) return
    let cancelled = false
    setLoading(true)
    setError(null)
    api
      .tree(path ?? undefined)
      .then((listing) => {
        if (cancelled) return
        setEntries(listing.entries)
        if (path === null && listing.path) setLabel(listing.path)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'failed to load')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [expanded, path, refreshNonce])

  const indent = { paddingLeft: `${depth * 14 + 8}px` }

  return (
    <div className="tree-node">
      <button
        type="button"
        className="tree-row tree-dir"
        style={indent}
        onClick={() => setExpanded((v) => !v)}
        aria-expanded={expanded}
      >
        <ChevronRight
          size={14}
          className={`tree-chevron${expanded ? ' expanded' : ''}`}
        />
        {expanded ? <FolderOpen size={15} /> : <Folder size={15} />}
        <span className="tree-label">{label}</span>
        {loading && <Loader2 size={13} className="tree-spinner" />}
      </button>

      {expanded && (
        <div className="tree-children">
          {error && (
            <div className="tree-message" style={{ paddingLeft: `${depth * 14 + 28}px` }}>
              {error}
            </div>
          )}
          {entries?.length === 0 && !error && (
            <div className="tree-message" style={{ paddingLeft: `${depth * 14 + 28}px` }}>
              empty
            </div>
          )}
          {entries?.map((entry) =>
            entry.type === 'dir' ? (
              <DirNode
                key={entry.path}
                path={entry.path}
                name={entry.name}
                depth={depth + 1}
                activePath={activePath}
                refreshNonce={refreshNonce}
                onOpenFile={onOpenFile}
              />
            ) : (
              <button
                type="button"
                key={entry.path}
                className={`tree-row tree-file${entry.path === activePath ? ' active' : ''}`}
                style={{ paddingLeft: `${(depth + 1) * 14 + 8}px` }}
                onClick={() => onOpenFile(entry.path)}
                title={entry.path}
              >
                <FileText size={15} className="tree-file-icon" />
                <span className="tree-label">{entry.name}</span>
              </button>
            ),
          )}
        </div>
      )}
    </div>
  )
}
