import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { Session } from '../types'

export function useSession() {
  const [session, setSession] = useState<Session | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function init() {
      try {
        // Check if we have a stored session that hasn't expired
        const stored = localStorage.getItem('hookdrop_session')
        if (stored) {
          const parsed: Session = JSON.parse(stored)
          if (new Date(parsed.expires_at) > new Date()) {
            setSession(parsed)
            setLoading(false)
            return
          }
        }

        // Create a fresh session
        const fresh = await api.createSession()
        localStorage.setItem('hookdrop_session', JSON.stringify(fresh))
        setSession(fresh)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to create session')
      } finally {
        setLoading(false)
      }
    }

    init()
  }, [])

  function resetSession() {
    localStorage.removeItem('hookdrop_session')
    setLoading(true)
    setSession(null)
    setError(null)
    api.createSession().then((s) => {
      localStorage.setItem('hookdrop_session', JSON.stringify(s))
      setSession(s)
    }).finally(() => setLoading(false))
  }

  return { session, loading, error, resetSession }
}
