import { useCallback, useEffect, useRef, useState } from 'react'
import { FolderTree, LogOut, Search } from 'lucide-react'
import { api } from './api/client'
import { useAuth } from './hooks/useAuth'
import { Login } from './components/Login'
import { FileTree } from './components/FileTree'
import { Toolbar } from './components/Toolbar'
import { Editor } from './components/Editor'
import { Preview } from './components/Preview'
import { StatusBar } from './components/StatusBar'
import { SearchPalette } from './components/SearchPalette'
import type { OpenFile, Theme, ViewMode } from './types'

/** A pending text-input dialog (new file, rename). */
interface PromptState {
  title: string
  label: string
  initial: string
  confirmText: string
  onConfirm: (value: string) => void
}

/** A transient notification. */
interface Toast {
  kind: 'info' | 'error'
  message: string
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : 'something went wrong'
}

function dirOf(path: string): string {
  return path.slice(0, path.lastIndexOf('/') + 1)
}

function baseName(path: string): string {
  return path.replace(/\/+$/, '').split('/').pop() ?? path
}

/** The mdtree single-page application. */
export function App() {
  const auth = useAuth()

  const [theme, setTheme] = useState<Theme>(() =>
    localStorage.getItem('mdtree.theme') === 'light' ? 'light' : 'dark',
  )
  const [viewMode, setViewMode] = useState<ViewMode>(() => {
    const stored = localStorage.getItem('mdtree.viewMode')
    return stored === 'edit' || stored === 'preview' ? stored : 'split'
  })

  const [file, setFile] = useState<OpenFile | null>(null)
  const [editorKey, setEditorKey] = useState(0)
  const [saving, setSaving] = useState(false)
  const [lastSaved, setLastSaved] = useState<string | null>(null)
  const [treeNonce, setTreeNonce] = useState(0)
  const [searchOpen, setSearchOpen] = useState(false)
  const [prompt, setPrompt] = useState<PromptState | null>(null)
  const [toast, setToast] = useState<Toast | null>(null)
  const [rootPath, setRootPath] = useState<string | null>(null)
  const [indexedFiles, setIndexedFiles] = useState<number | null>(null)

  const dirty = file !== null && file.content !== file.original

  const notify = useCallback((message: string, kind: Toast['kind'] = 'info') => {
    setToast({ kind, message })
  }, [])

  // Persist and apply the colour theme.
  useEffect(() => {
    document.documentElement.dataset.theme = theme
    localStorage.setItem('mdtree.theme', theme)
  }, [theme])

  useEffect(() => {
    localStorage.setItem('mdtree.viewMode', viewMode)
  }, [viewMode])

  // Auto-dismiss notifications.
  useEffect(() => {
    if (!toast) return
    const timer = window.setTimeout(() => setToast(null), 4000)
    return () => window.clearTimeout(timer)
  }, [toast])

  // Warn before leaving with unsaved changes.
  useEffect(() => {
    function onBeforeUnload(e: BeforeUnloadEvent) {
      if (dirty) {
        e.preventDefault()
        e.returnValue = ''
      }
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [dirty])

  // Load the root path and index size once authenticated.
  useEffect(() => {
    if (auth.state !== 'authed') return
    api
      .tree()
      .then((listing) => setRootPath(listing.path))
      .catch(() => {})
    api
      .stats()
      .then((stats) => setIndexedFiles(stats.index.files))
      .catch(() => {})
  }, [auth.state])

  const openFile = useCallback(
    async (path: string) => {
      if (dirty && !window.confirm('Discard unsaved changes in the current file?')) {
        return
      }
      try {
        const fc = await api.readFile(path)
        setFile({
          path: fc.path,
          name: fc.name,
          original: fc.content,
          content: fc.content,
          modTime: fc.modTime,
        })
        setEditorKey((k) => k + 1)
        setLastSaved(null)
      } catch (err) {
        notify(errorMessage(err), 'error')
      }
    },
    [dirty, notify],
  )

  const save = useCallback(async () => {
    if (!file || file.content === file.original || saving) return
    const snapshot = file.content
    setSaving(true)
    try {
      const info = await api.saveFile(file.path, snapshot)
      setFile((f) =>
        f && f.path === file.path ? { ...f, original: snapshot, modTime: info.modTime } : f,
      )
      setLastSaved(new Date().toLocaleTimeString())
    } catch (err) {
      notify(errorMessage(err), 'error')
    } finally {
      setSaving(false)
    }
  }, [file, saving, notify])

  const onEditorChange = useCallback((value: string) => {
    setFile((f) => (f ? { ...f, content: value } : f))
  }, [])

  // Global keyboard shortcuts.
  useEffect(() => {
    if (auth.state !== 'authed') return
    function onKey(e: KeyboardEvent) {
      const mod = e.ctrlKey || e.metaKey
      if (mod && e.key.toLowerCase() === 's') {
        e.preventDefault()
        void save()
      } else if (mod && e.key.toLowerCase() === 'p') {
        e.preventDefault()
        setSearchOpen(true)
      } else if (e.key === 'Escape') {
        setSearchOpen(false)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [auth.state, save])

  function promptNewFile() {
    const dir = file
      ? dirOf(file.path)
      : rootPath
        ? rootPath.replace(/\/*$/, '/')
        : '/'
    setPrompt({
      title: 'New markdown file',
      label: 'Full path (must end in .md)',
      initial: `${dir}untitled.md`,
      confirmText: 'Create',
      onConfirm: (path) => {
        setPrompt(null)
        void (async () => {
          try {
            const info = await api.createFile(path, `# ${baseName(path).replace(/\.[^.]+$/, '')}\n\n`)
            setTreeNonce((n) => n + 1)
            await openFile(info.path)
            setIndexedFiles((n) => (n === null ? n : n + 1))
            notify('File created')
          } catch (err) {
            notify(errorMessage(err), 'error')
          }
        })()
      },
    })
  }

  function promptRename() {
    if (!file) return
    const from = file.path
    setPrompt({
      title: 'Rename file',
      label: 'New full path',
      initial: from,
      confirmText: 'Rename',
      onConfirm: (to) => {
        setPrompt(null)
        void (async () => {
          try {
            const info = await api.renameFile(from, to)
            setTreeNonce((n) => n + 1)
            setFile((f) => (f && f.path === from ? { ...f, path: info.path, name: info.name } : f))
            notify('File renamed')
          } catch (err) {
            notify(errorMessage(err), 'error')
          }
        })()
      },
    })
  }

  async function deleteFile() {
    if (!file) return
    if (!window.confirm(`Delete "${file.name}"? This cannot be undone.`)) return
    const path = file.path
    try {
      await api.deleteFile(path)
      setTreeNonce((n) => n + 1)
      setFile(null)
      setIndexedFiles((n) => (n === null ? n : Math.max(0, n - 1)))
      notify('File deleted')
    } catch (err) {
      notify(errorMessage(err), 'error')
    }
  }

  async function reindex() {
    try {
      const res = await api.reindex()
      setIndexedFiles(res.files)
      notify(`Indexed ${res.files} files in ${Math.round(res.durationMs)} ms`)
    } catch (err) {
      notify(errorMessage(err), 'error')
    }
  }

  function promptNewFileIn(dir: string) {
    const prefix = dir.replace(/\/*$/, '/')
    setPrompt({
      title: 'New markdown file',
      label: 'Full path (must end in .md)',
      initial: `${prefix}untitled.md`,
      confirmText: 'Create',
      onConfirm: (path) => {
        setPrompt(null)
        void (async () => {
          try {
            const info = await api.createFile(path, `# ${baseName(path).replace(/\.[^.]+$/, '')}\n\n`)
            setTreeNonce((n) => n + 1)
            await openFile(info.path)
            setIndexedFiles((n) => (n === null ? n : n + 1))
            notify('File created')
          } catch (err) {
            notify(errorMessage(err), 'error')
          }
        })()
      },
    })
  }

  function promptNewDir(parentDir: string) {
    const prefix = parentDir.replace(/\/*$/, '/')
    setPrompt({
      title: 'New folder',
      label: 'Full path',
      initial: `${prefix}new-folder`,
      confirmText: 'Create',
      onConfirm: (path) => {
        setPrompt(null)
        void (async () => {
          try {
            await api.mkdir(path)
            setTreeNonce((n) => n + 1)
            notify('Folder created')
          } catch (err) {
            notify(errorMessage(err), 'error')
          }
        })()
      },
    })
  }

  function promptRenameFile(path: string) {
    setPrompt({
      title: 'Rename file',
      label: 'New full path',
      initial: path,
      confirmText: 'Rename',
      onConfirm: (to) => {
        setPrompt(null)
        void (async () => {
          try {
            const info = await api.renameFile(path, to)
            setTreeNonce((n) => n + 1)
            setFile((f) => (f && f.path === path ? { ...f, path: info.path, name: info.name } : f))
            notify('File renamed')
          } catch (err) {
            notify(errorMessage(err), 'error')
          }
        })()
      },
    })
  }

  function promptRenameDirAt(path: string) {
    setPrompt({
      title: 'Rename folder',
      label: 'New full path',
      initial: path,
      confirmText: 'Rename',
      onConfirm: (to) => {
        setPrompt(null)
        void (async () => {
          try {
            await api.renameDir(path, to)
            setTreeNonce((n) => n + 1)
            notify('Folder renamed')
          } catch (err) {
            notify(errorMessage(err), 'error')
          }
        })()
      },
    })
  }

  async function deleteFileAt(path: string) {
    const name = baseName(path)
    if (!window.confirm(`Delete "${name}"? This cannot be undone.`)) return
    try {
      await api.deleteFile(path)
      setTreeNonce((n) => n + 1)
      if (file?.path === path) setFile(null)
      setIndexedFiles((n) => (n === null ? n : Math.max(0, n - 1)))
      notify('File deleted')
    } catch (err) {
      notify(errorMessage(err), 'error')
    }
  }

  async function deleteDirAt(path: string) {
    const name = baseName(path)
    if (!window.confirm(`Delete folder "${name}"? It must be empty.`)) return
    try {
      await api.deleteDir(path)
      setTreeNonce((n) => n + 1)
      notify('Folder deleted')
    } catch (err) {
      notify(errorMessage(err), 'error')
    }
  }

  if (auth.state === 'loading') {
    return <div className="boot">Loading mdtree…</div>
  }
  if (auth.state === 'anon') {
    return <Login onLogin={auth.login} />
  }

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="sidebar-head">
          <div className="brand">
            <FolderTree size={20} strokeWidth={2.2} />
            <span>mdtree</span>
          </div>
          <button
            type="button"
            className="search-trigger"
            onClick={() => setSearchOpen(true)}
          >
            <Search size={15} />
            <span>Search files</span>
            <kbd>Ctrl P</kbd>
          </button>
        </div>

        <FileTree
          activePath={file?.path ?? null}
          refreshNonce={treeNonce}
          onOpenFile={(path) => void openFile(path)}
          onNewFile={(dir) => promptNewFileIn(dir)}
          onNewDir={(dir) => promptNewDir(dir)}
          onRenameFile={(path) => promptRenameFile(path)}
          onRenameDir={(path) => promptRenameDirAt(path)}
          onDeleteFile={(path) => void deleteFileAt(path)}
          onDeleteDir={(path) => void deleteDirAt(path)}
        />

        <div className="sidebar-foot">
          <button type="button" className="logout-btn" onClick={() => void auth.logout()}>
            <LogOut size={15} />
            <span>Sign out</span>
          </button>
        </div>
      </aside>

      <main className="main">
        <Toolbar
          file={file}
          dirty={dirty}
          saving={saving}
          viewMode={viewMode}
          theme={theme}
          onSave={() => void save()}
          onViewMode={setViewMode}
          onToggleTheme={() => setTheme((t) => (t === 'dark' ? 'light' : 'dark'))}
          onNewFile={promptNewFile}
          onRename={promptRename}
          onDelete={() => void deleteFile()}
          onReindex={() => void reindex()}
        />

        <div className={`workspace mode-${viewMode}`}>
          {file ? (
            <>
              {viewMode !== 'preview' && (
                <div className="pane pane-editor">
                  <Editor
                    docId={String(editorKey)}
                    initialDoc={file.content}
                    theme={theme}
                    onChange={onEditorChange}
                    onSave={() => void save()}
                  />
                </div>
              )}
              {viewMode !== 'edit' && (
                <div className="pane pane-preview">
                  <Preview content={file.content} />
                </div>
              )}
            </>
          ) : (
            <div className="empty-state">
              <FolderTree size={48} strokeWidth={1.4} />
              <p>Select a markdown file from the tree, or press</p>
              <p>
                <kbd>Ctrl</kbd> <kbd>P</kbd> to search by name.
              </p>
            </div>
          )}
        </div>

        <StatusBar
          file={file}
          dirty={dirty}
          saving={saving}
          lastSaved={lastSaved}
          indexedFiles={indexedFiles}
        />
      </main>

      {searchOpen && (
        <SearchPalette
          onSelect={(path) => {
            setSearchOpen(false)
            void openFile(path)
          }}
          onClose={() => setSearchOpen(false)}
        />
      )}

      {prompt && <PromptModal state={prompt} onCancel={() => setPrompt(null)} />}

      {toast && <div className={`toast toast-${toast.kind}`}>{toast.message}</div>}
    </div>
  )
}

/** A small modal that collects a single line of text. */
function PromptModal({ state, onCancel }: { state: PromptState; onCancel: () => void }) {
  const [value, setValue] = useState(state.initial)
  const inputRef = useRef<HTMLInputElement | null>(null)

  useEffect(() => {
    const input = inputRef.current
    if (!input) return
    input.focus()
    // Select the file name, keeping the directory prefix intact.
    const slash = state.initial.lastIndexOf('/')
    input.setSelectionRange(slash + 1, state.initial.length)
  }, [state.initial])

  function confirm() {
    const trimmed = value.trim()
    if (trimmed) state.onConfirm(trimmed)
  }

  return (
    <div className="palette-backdrop" onMouseDown={onCancel}>
      <div className="modal" onMouseDown={(e) => e.stopPropagation()}>
        <h2 className="modal-title">{state.title}</h2>
        <label className="modal-label">{state.label}</label>
        <input
          ref={inputRef}
          className="modal-input"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              confirm()
            } else if (e.key === 'Escape') {
              onCancel()
            }
          }}
        />
        <div className="modal-buttons">
          <button type="button" className="btn-ghost" onClick={onCancel}>
            Cancel
          </button>
          <button
            type="button"
            className="btn-primary"
            disabled={!value.trim()}
            onClick={confirm}
          >
            {state.confirmText}
          </button>
        </div>
      </div>
    </div>
  )
}
