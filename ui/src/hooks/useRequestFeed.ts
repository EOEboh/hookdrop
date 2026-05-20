import { useEffect, useRef, useState, useCallback } from 'react'
import { api } from '../api/client'
import type { CapturedRequest, ConnectionStatus, RequestFilters } from '../types'

export function useRequestFeed(
  sessionId: string | null,
  filters: RequestFilters,
) {
  const [requests, setRequests] = useState<CapturedRequest[]>([])
  const [status, setStatus]     = useState<ConnectionStatus>('connecting')
  const esRef                   = useRef<EventSource | null>(null)

  // Re-fetch history whenever filters change
  const fetchHistory = useCallback(async () => {
    if (!sessionId) return
    try {
      const history = await api.getRequests(sessionId, filters)
      setRequests(history ?? [])
    } catch {
      // Non-fatal
    }
  }, [sessionId, filters])

  // SSE connection: only depends on sessionId, not filters
  // Live events arrive regardless of filter state
  useEffect(() => {
    if (!sessionId) return

    let cancelled = false

    fetchHistory()

    const es = new EventSource(api.sseUrl(sessionId))
    esRef.current = es

    es.addEventListener('connected', () => {
      if (!cancelled) setStatus('live')
    })

    es.addEventListener('request', (e: MessageEvent) => {
      if (cancelled) return
      try {
        const incoming: CapturedRequest = JSON.parse(e.data)
        // Only prepend if the incoming request matches active filters
        // This keeps the live feed consistent with what's shown
        setRequests(prev => {
          if (passesClientFilter(incoming, filters)) {
            return [incoming, ...prev]
          }
          return prev
        })
      } catch {
        // Malformed event
      }
    })

    es.onerror = () => {
      if (!cancelled) setStatus('disconnected')
    }

    return () => {
      cancelled = true
      esRef.current?.close()
      setStatus('connecting')
    }
  }, [sessionId]) // SSE reconnects only on session change

  // Re-fetch history when filters change without reconnecting SSE
  useEffect(() => {
    if (!sessionId) return
    fetchHistory()
  }, [fetchHistory])

  function clearRequests() {
    setRequests([])
  }

  return { requests, status, clearRequests }
}

// Client-side filter check for incoming SSE events
// Mirrors the server-side filter logic so live events are consistent
function passesClientFilter(req: CapturedRequest, filters: RequestFilters): boolean {
  if (filters.method && req.method !== filters.method) return false

  if (filters.verified) {
    const status = req.verified ?? 'unverified'
    if (status !== filters.verified) return false
  }

  if (filters.search) {
    const body = typeof req.body === 'string' ? req.body : ''
    if (!body.toLowerCase().includes(filters.search.toLowerCase())) return false
  }

  if (filters.range) {
    const ms = { '1h': 3600000, '24h': 86400000, '7d': 604800000 }[filters.range]
    if (ms && Date.now() - new Date(req.received_at).getTime() > ms) return false
  }

  return true
}