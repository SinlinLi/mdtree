import { useEffect, useRef } from 'react'
import { EditorState } from '@codemirror/state'
import { EditorView, keymap } from '@codemirror/view'
import { basicSetup } from 'codemirror'
import { markdown, markdownLanguage } from '@codemirror/lang-markdown'
import { languages } from '@codemirror/language-data'
import { oneDark } from '@codemirror/theme-one-dark'
import type { Theme } from '../types'

interface EditorProps {
  /** Unique key for the open file; changing it reloads the document. */
  docId: string
  /** The document text to seed the editor with. */
  initialDoc: string
  theme: Theme
  onChange: (value: string) => void
  onSave: () => void
}

/** A CodeMirror 6 markdown source editor. */
export function Editor({ docId, initialDoc, theme, onChange, onSave }: EditorProps) {
  const host = useRef<HTMLDivElement | null>(null)
  // Callbacks are held in refs so the editor is not recreated when they change.
  const onChangeRef = useRef(onChange)
  const onSaveRef = useRef(onSave)
  onChangeRef.current = onChange
  onSaveRef.current = onSave

  useEffect(() => {
    const parent = host.current
    if (!parent) return

    const extensions = [
      basicSetup,
      keymap.of([
        {
          key: 'Mod-s',
          preventDefault: true,
          run: () => {
            onSaveRef.current()
            return true
          },
        },
      ]),
      markdown({ base: markdownLanguage, codeLanguages: languages }),
      EditorView.lineWrapping,
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          onChangeRef.current(update.state.doc.toString())
        }
      }),
    ]
    if (theme === 'dark') {
      extensions.push(oneDark)
    }

    const view = new EditorView({
      state: EditorState.create({ doc: initialDoc, extensions }),
      parent,
    })
    view.focus()
    return () => view.destroy()
    // The editor is rebuilt only when the open file or theme changes;
    // initialDoc is just the seed and intentionally not a dependency.
  }, [docId, theme])

  return <div className="editor-host" ref={host} />
}
