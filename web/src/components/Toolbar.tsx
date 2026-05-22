import {
  Columns2,
  Eye,
  FilePlus2,
  Moon,
  PencilLine,
  RefreshCw,
  Save,
  SquarePen,
  Sun,
  Trash2,
} from 'lucide-react'
import type { OpenFile, Theme, ViewMode } from '../types'

interface ToolbarProps {
  file: OpenFile | null
  dirty: boolean
  saving: boolean
  viewMode: ViewMode
  theme: Theme
  onSave: () => void
  onViewMode: (mode: ViewMode) => void
  onToggleTheme: () => void
  onNewFile: () => void
  onRename: () => void
  onDelete: () => void
  onReindex: () => void
}

/** The action bar above the editor: file name, file actions and view modes. */
export function Toolbar({
  file,
  dirty,
  saving,
  viewMode,
  theme,
  onSave,
  onViewMode,
  onToggleTheme,
  onNewFile,
  onRename,
  onDelete,
  onReindex,
}: ToolbarProps) {
  return (
    <header className="toolbar">
      <div className="toolbar-file">
        {file ? (
          <>
            <span className="toolbar-name">{file.name}</span>
            {dirty && <span className="dirty-dot" title="Unsaved changes" />}
          </>
        ) : (
          <span className="toolbar-name muted">No file open</span>
        )}
      </div>

      <div className="toolbar-actions">
        <button type="button" className="icon-btn" onClick={onNewFile} title="New file">
          <FilePlus2 size={17} />
        </button>
        <button
          type="button"
          className="icon-btn"
          onClick={onRename}
          disabled={!file}
          title="Rename file"
        >
          <PencilLine size={17} />
        </button>
        <button
          type="button"
          className="icon-btn danger"
          onClick={onDelete}
          disabled={!file}
          title="Delete file"
        >
          <Trash2 size={17} />
        </button>

        <span className="toolbar-sep" />

        <div className="segmented" role="group" aria-label="View mode">
          <button
            type="button"
            className={viewMode === 'edit' ? 'active' : ''}
            onClick={() => onViewMode('edit')}
            title="Editor only"
          >
            <SquarePen size={16} />
          </button>
          <button
            type="button"
            className={viewMode === 'split' ? 'active' : ''}
            onClick={() => onViewMode('split')}
            title="Split view"
          >
            <Columns2 size={16} />
          </button>
          <button
            type="button"
            className={viewMode === 'preview' ? 'active' : ''}
            onClick={() => onViewMode('preview')}
            title="Preview only"
          >
            <Eye size={16} />
          </button>
        </div>

        <span className="toolbar-sep" />

        <button
          type="button"
          className="icon-btn"
          onClick={onReindex}
          title="Rebuild search index"
        >
          <RefreshCw size={16} />
        </button>
        <button
          type="button"
          className="icon-btn"
          onClick={onToggleTheme}
          title="Toggle theme"
        >
          {theme === 'dark' ? <Sun size={17} /> : <Moon size={17} />}
        </button>

        <button
          type="button"
          className="save-btn"
          onClick={onSave}
          disabled={!file || !dirty || saving}
        >
          <Save size={16} />
          {saving ? 'Saving…' : 'Save'}
        </button>
      </div>
    </header>
  )
}
