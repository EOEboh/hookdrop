import { useState } from 'react'
import { api } from '../api/client'
import type { CapturedRequest, ReplayResponse } from '../types'

export function useReplay() {
  const [loading, setLoading] = useState(false)
  const [response, setResponse] = useState<ReplayResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function replay(
    request: CapturedRequest,
    targetUrl: string,
    overrideBody?: string,
  ) {
    setLoading(true)
    setResponse(null)
    setError(null)

    try {
      const result = await api.replay({
        request_id: request.id,
        target_url: targetUrl,
        body: overrideBody,
      })
      setResponse(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Replay failed')
    } finally {
      setLoading(false)
    }
  }

  function reset() {
    setResponse(null)
    setError(null)
  }

  return { replay, loading, response, error, reset }
}