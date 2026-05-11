import { useEffect, useRef } from 'react'
import { useAuth } from '../context/AuthContext'
import { Spinner } from '../components/ui/Spinner'

export function AuthCallbackPage() {
  const { login } = useAuth()
  const processed = useRef(false)

  useEffect(() => {
    // Guard against running twice in React StrictMode
    if (processed.current) return
    processed.current = true

    const hash = window.location.hash
    const params = new URLSearchParams(hash.slice(1))
    const token = params.get('token')

    if (token) {
      // Store the token first, then hard-navigate
    
      login(token)
      window.location.replace('/')
    } else {
      // No token in the URL — something went wrong
      // Redirect to login with an error hint
      window.location.replace('/?error=invalid_link')
    }
  }, [login])

  return (
    <div className="min-h-screen bg-zinc-950 flex flex-col items-center justify-center gap-4">
      <Spinner size={6} />
      <p className="text-zinc-400 text-sm">Logging you in…</p>
    </div>
  )
}