import { useEffect, useRef, useState, useCallback } from 'react'
import { api } from '../api/client'
import type { CapturedRequest, ConnectionStatus, RequestFilters } from '../types'
import { trackEvent } from '../lib/analytics'

export function useRequestFeed(
  sessionId: string | null,
  filters: RequestFilters,
) {
  const [requests, setRequests]   = useState<CapturedRequest[]>([])
  const [status, setStatus]       = useState<ConnectionStatus>('connecting')
  const [totalCount, setTotalCount] = useState(0)
  const [newIds, setNewIds]       = useState<Set<string>>(new Set())
  const esRef = useRef<EventSource | null>(null)

  const fetchHistory = useCallback(async () => {
    if (!sessionId) return
    try {
      const history = await api.getRequests(sessionId, filters)
      setRequests(history ?? [])

      const isFiltered = filters.search || filters.method || filters.verified || filters.range
      if (isFiltered) {
        const all = await api.getRequests(sessionId, {
          search: '', method: '', verified: '', range: '',
        })
        setTotalCount(all?.length ?? 0)
      } else {
        setTotalCount(history?.length ?? 0)
      }
    } catch {
      // Non-fatal
    }
  }, [sessionId, filters])

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

        setTotalCount(prev => prev + 1)

        setRequests(prev => {
          if (prev.length === 0) {
          trackEvent('first_webhook_received', {           
            method:   incoming.method,
            verified: incoming.verified,
            has_body: incoming.body_size > 0,
          })
        }

          if (passesClientFilter(incoming, filters)) {
            // Mark as freshly-arrived so the list can play an entrance
            // animation, then clear the flag once it has had time to run.
            setNewIds(ids => new Set(ids).add(incoming.id))
            setTimeout(() => {
              setNewIds(ids => {
                if (!ids.has(incoming.id)) return ids
                const next = new Set(ids)
                next.delete(incoming.id)
                return next
              })
            }, 600)
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
  }, [sessionId])

  useEffect(() => {
    if (!sessionId) return
    fetchHistory()
  }, [fetchHistory])

  function clearRequests() {
    setRequests([])
    setTotalCount(0)
    setNewIds(new Set())
  }

  return { requests, status, clearRequests, totalCount, newIds }
}

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