import { useEffect } from 'react'
import { useAuth } from '../context/AuthContext'
import { Spinner } from '../components/ui/Spinner'

export function AuthCallbackPage() {
  const { login } = useAuth()

  useEffect(() => {
    // Token lives in the URL fragment (#token=xxx)
    // Fragment is never sent to the server — safe
    const hash = window.location.hash
    const token = new URLSearchParams(hash.slice(1)).get('token')

    if (token) {
      login(token)
      // Clean the token out of the URL
      window.history.replaceState(null, '', '/auth/callback')
    } else {
      window.location.href = '/'
    }
  }, [login])

  return (
    <div className="min-h-screen bg-zinc-950 flex items-center justify-center gap-3">
      <Spinner size={5} />
      <span className="text-zinc-400 text-sm">Logging you in…</span>
    </div>
  )
}