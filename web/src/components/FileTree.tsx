import { useCallback, useEffect, useRef, useState } from 'react'
import { ChevronRight, FileText, Folder, FolderOpen, Loader2, FilePlus, FolderPlus, Pencil, Trash2 } from 'lucide-react'
import { api } from '../api/client'
import type { TreeEntry } from '../types'

interface ContextMenuTarget {
  type: 'dir' | 'file'
  path: string
  name: string
  x: number
  y: number
}

interface FileTreeProps {
  activePath: string | null
  refreshNonce: number
  onOpenFile: (path: string) => void
  onNewFile: (dirPath: string) => void
  onNewDir: (dirPath: string) => void
  onRenameFile: (path: string) => void
  onRenameDir: (path: string) => void
  onDeleteFile: (path: string) => void
  onDeleteDir: (path: string) => void
}

export function FileTree({
  activePath,
  refreshNonce,
  onOpenFile,
  onNewFile,
  onNewDir,
  onRenameFile,
  onRenameDir,
  onDeleteFile,
  onDeleteDir,
}: FileTreeProps) {
  const [menu, setMenu] = useState<ContextMenuTarget | null>(null)

  const closeMenu = useCallback(() => setMenu(null), [])

  useEffect(() => {
    if (!menu) return
    const handler = () => setMenu(null)
    window.addEventListener('click', handler)
    window.addEventListener('contextmenu', handler)
    return () => {
      window.removeEventListener('click', handler)
      window.removeEventListener('contextmenu', handler)
    }
  }, [menu])

  const onContextMenu = useCallback((e: React.MouseEvent, entry: ContextMenuTarget) => {
    e.preventDefault()
    e.stopPropagation()
    setMenu(entry)
  }, [])

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
        onContextMenu={onContextMenu}
      />
      {menu && (
        <ContextMenu
          target={menu}
          onClose={closeMenu}
          onNewFile={() => { closeMenu(); onNewFile(menu.type === 'dir' ? menu.path : dirOf(menu.path)) }}
          onNewDir={() => { closeMenu(); onNewDir(menu.type === 'dir' ? menu.path : dirOf(menu.path)) }}
          onRename={() => {
            closeMenu()
            if (menu.type === 'file') onRenameFile(menu.path)
            else onRenameDir(menu.path)
          }}
          onDelete={() => {
            closeMenu()
            if (menu.type === 'file') onDeleteFile(menu.path)
            else onDeleteDir(menu.path)
          }}
        />
      )}
    </div>
  )
}

function dirOf(path: string): string {
  return path.slice(0, path.lastIndexOf('/') + 1)
}

interface DirNodeProps {
  path: string | null
  name: string
  depth: number
  defaultExpanded?: boolean
  activePath: string | null
  refreshNonce: number
  onOpenFile: (path: string) => void
  onContextMenu: (e: React.MouseEvent, target: ContextMenuTarget) => void
}

function DirNode({
  path,
  name,
  depth,
  defaultExpanded,
  activePath,
  refreshNonce,
  onOpenFile,
  onContextMenu,
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
  const resolvedPath = path ?? label

  return (
    <div className="tree-node">
      <button
        type="button"
        className="tree-row tree-dir"
        style={indent}
        onClick={() => setExpanded((v) => !v)}
        onContextMenu={(e) => {
          if (path !== null) {
            onContextMenu(e, { type: 'dir', path: resolvedPath, name: label, x: e.clientX, y: e.clientY })
          }
        }}
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
                onContextMenu={onContextMenu}
              />
            ) : (
              <button
                type="button"
                key={entry.path}
                className={`tree-row tree-file${entry.path === activePath ? ' active' : ''}`}
                style={{ paddingLeft: `${(depth + 1) * 14 + 8}px` }}
                onClick={() => onOpenFile(entry.path)}
                onContextMenu={(e) =>
                  onContextMenu(e, { type: 'file', path: entry.path, name: entry.name, x: e.clientX, y: e.clientY })
                }
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

interface ContextMenuProps {
  target: ContextMenuTarget
  onClose: () => void
  onNewFile: () => void
  onNewDir: () => void
  onRename: () => void
  onDelete: () => void
}

function ContextMenu({ target, onNewFile, onNewDir, onRename, onDelete }: ContextMenuProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = ref.current
    if (!el) return
    const rect = el.getBoundingClientRect()
    const vw = window.innerWidth
    const vh = window.innerHeight
    if (rect.right > vw) el.style.left = `${target.x - rect.width}px`
    if (rect.bottom > vh) el.style.top = `${target.y - rect.height}px`
  }, [target.x, target.y])

  return (
    <div
      ref={ref}
      className="ctx-menu"
      style={{ left: target.x, top: target.y }}
      onMouseDown={(e) => e.stopPropagation()}
      onClick={(e) => e.stopPropagation()}
      onContextMenu={(e) => { e.preventDefault(); e.stopPropagation() }}
    >
      <button type="button" className="ctx-item" onClick={onNewFile}>
        <FilePlus size={14} />
        <span>New File</span>
      </button>
      <button type="button" className="ctx-item" onClick={onNewDir}>
        <FolderPlus size={14} />
        <span>New Folder</span>
      </button>
      <div className="ctx-sep" />
      <button type="button" className="ctx-item" onClick={onRename}>
        <Pencil size={14} />
        <span>Rename</span>
      </button>
      <button type="button" className="ctx-item ctx-danger" onClick={onDelete}>
        <Trash2 size={14} />
        <span>Delete</span>
      </button>
    </div>
  )
}
