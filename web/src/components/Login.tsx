import { useState, type FormEvent } from 'react'
import { FolderTree } from 'lucide-react'
import { ApiError } from '../api/client'

interface LoginProps {
  onLogin: (password: string) => Promise<void>
}

/** The full-screen sign-in form shown when there is no active session. */
export function Login({ onLogin }: LoginProps) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!password || busy) return
    setBusy(true)
    setError(null)
    try {
      await onLogin(password)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'login failed')
      setBusy(false)
    }
  }

  return (
    <div className="login">
      <form className="login-card" onSubmit={submit}>
        <div className="login-brand">
          <FolderTree size={28} strokeWidth={2.2} />
          <span>mdtree</span>
        </div>
        <p className="login-tagline">Self-hosted markdown browser &amp; editor</p>
        <input
          type="password"
          className="login-input"
          placeholder="Password"
          value={password}
          autoFocus
          autoComplete="current-password"
          onChange={(e) => setPassword(e.target.value)}
        />
        {error && <div className="login-error">{error}</div>}
        <button type="submit" className="login-button" disabled={busy || !password}>
          {busy ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </div>
  )
}
