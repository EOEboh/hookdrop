import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import type { AuthState, AuthUser } from '../types'

interface AuthContextValue extends AuthState {
  login: (token: string) => void
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

function parseToken(token: string): AuthUser | null {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return { id: payload.user_id, email: payload.email }
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    user: null,
    token: null,
    loading: true,
  })

  useEffect(() => {
    const token = localStorage.getItem('hookdrop_token')
    if (token) {
      const user = parseToken(token)
      // Validate token hasn't expired before trusting it
      try {
        const payload = JSON.parse(atob(token.split('.')[1]))
        if (payload.exp && payload.exp * 1000 < Date.now()) {
          localStorage.removeItem('hookdrop_token')
          setState({ user: null, token: null, loading: false })
          return
        }
      } catch {
        localStorage.removeItem('hookdrop_token')
        setState({ user: null, token: null, loading: false })
        return
      }
      setState({ user, token, loading: false })
    } else {
      setState(s => ({ ...s, loading: false }))
    }
  }, [])


  const login = useCallback((token: string) => {
    localStorage.setItem('hookdrop_token', token)
    const user = parseToken(token)
    setState({ user, token, loading: false })
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem('hookdrop_token')
    localStorage.removeItem('hookdrop_session')
    setState({ user: null, token: null, loading: false })
  }, [])

  return (
    <AuthContext.Provider value={{ ...state, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}