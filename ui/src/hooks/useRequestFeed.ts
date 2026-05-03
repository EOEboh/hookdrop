import { useEffect, useRef, useState } from 'react'
import { api } from '../api/client'
import type { CapturedRequest, ConnectionStatus } from '../types'

export function useRequestFeed(sessionId: string | null) {
  const [requests, setRequests] = useState<CapturedRequest[]>([])
  const [status, setStatus] = useState<ConnectionStatus>('connecting')
  const esRef = useRef<EventSource | null>(null)

  useEffect(() => {
    if (!sessionId) return

    // Capture as a non-null string so TypeScript is happy inside the closure
    const id: string = sessionId
    let cancelled = false

    async function init() {
      try {
        const history = await api.getRequests(id)   // was sessionId
        if (!cancelled) {
          setRequests(history ?? [])
        }
      } catch {
        // non-fatal
      }

      if (cancelled) return

      const es = new EventSource(api.sseUrl(id))    // was sessionId
      esRef.current = es

      es.addEventListener('connected', () => {
        if (!cancelled) setStatus('live')
      })

      es.addEventListener('request', (e: MessageEvent) => {
        if (cancelled) return
        try {
          const incoming: CapturedRequest = JSON.parse(e.data)
          setRequests((prev) => [incoming, ...prev])
        } catch {
          // malformed event
        }
      })

      es.onerror = () => {
        if (!cancelled) setStatus('disconnected')
      }
    }

    init()

    return () => {
      cancelled = true
      esRef.current?.close()
      setStatus('connecting')
    }
  }, [sessionId])

  function clearRequests() {
    setRequests([])
  }

  return { requests, status, clearRequests }
}