import type { OpenFile } from '../types'

interface StatusBarProps {
  file: OpenFile | null
  dirty: boolean
  saving: boolean
  lastSaved: string | null
  indexedFiles: number | null
}

/** The thin status strip along the bottom of the workspace. */
export function StatusBar({ file, dirty, saving, lastSaved, indexedFiles }: StatusBarProps) {
  const lines = file ? file.content.split('\n').length : 0
  const chars = file ? file.content.length : 0

  let state = 'Ready'
  if (saving) state = 'Saving…'
  else if (dirty) state = 'Unsaved changes'
  else if (lastSaved) state = `Saved ${lastSaved}`

  return (
    <footer className="status-bar">
      <span className="status-path" title={file?.path}>
        {file ? file.path : 'No file open'}
      </span>
      <span className="status-right">
        {file && (
          <span className="status-item">
            {lines} {lines === 1 ? 'line' : 'lines'} · {chars} chars
          </span>
        )}
        <span className={`status-item${dirty ? ' status-dirty' : ''}`}>{state}</span>
        {indexedFiles !== null && (
          <span className="status-item">{indexedFiles} indexed</span>
        )}
      </span>
    </footer>
  )
}
