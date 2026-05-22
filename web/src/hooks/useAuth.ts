import { useCallback, useEffect, useState } from 'react'
import { api, setUnauthorizedHandler } from '../api/client'

/** Authentication lifecycle states. */
export type AuthState = 'loading' | 'authed' | 'anon'

/** Tracks authentication state and exposes login/logout actions. */
export function useAuth() {
  const [state, setState] = useState<AuthState>('loading')

  useEffect(() => {
    // A 401 on any authenticated call drops us back to the login screen.
    setUnauthorizedHandler(() => setState('anon'))

    let cancelled = false
    api
      .authStatus()
      .then((res) => {
        if (!cancelled) setState(res.authenticated ? 'authed' : 'anon')
      })
      .catch(() => {
        if (!cancelled) setState('anon')
      })
    return () => {
      cancelled = true
    }
  }, [])

  const login = useCallback(async (password: string) => {
    await api.login(password)
    setState('authed')
  }, [])

  const logout = useCallback(async () => {
    try {
      await api.logout()
    } catch {
      // Ignore network errors — the local session is cleared regardless.
    }
    setState('anon')
  }, [])

  return { state, login, logout }
}
