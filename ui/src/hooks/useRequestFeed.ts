import { useEffect, useRef, useState } from 'react'
import { api } from '../api/client'
import type { CapturedRequest, ConnectionStatus } from '../types'

export function useRequestFeed(sessionId: string | null) {
  const [requests, setRequests] = useState<CapturedRequest[]>([])
  const [status, setStatus] = useState<ConnectionStatus>('connecting')
  const esRef = useRef<EventSource | null>(null)

  useEffect(() => {
    if (!sessionId) return

    let cancelled = false

    async function init() {
      // 1. Fetch request history first
      try {
        const history = await api.getRequests(sessionId)
        if (!cancelled) {
          setRequests(history ?? [])
        }
      } catch {
        // History fetch failing is non-fatal — SSE will still work
      }

      if (cancelled) return

      // 2. Open SSE connection
      const es = new EventSource(api.sseUrl(sessionId))
      esRef.current = es

      es.addEventListener('connected', () => {
        if (!cancelled) setStatus('live')
      })

      es.addEventListener('request', (e: MessageEvent) => {
        if (cancelled) return
        try {
          const incoming: CapturedRequest = JSON.parse(e.data)
          setRequests((prev) => [incoming, ...prev]) // newest first
        } catch {
          // malformed event — ignore
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