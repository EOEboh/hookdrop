import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { Endpoint } from '../types'

export function useEndpoints() {
  const [endpoints, setEndpoints] = useState<Endpoint[]>([])
  const [loading, setLoading]     = useState(true)
  const [error, setError]         = useState<string | null>(null)

  const fetch = useCallback(async () => {
    try {
      const data = await api.getEndpoints()
      setEndpoints(data ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load endpoints')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetch() }, [fetch])

  async function createEndpoint(data: {
    slug: string
    name: string
    description?: string
  }) {
    const ep = await api.createEndpoint(data)
    setEndpoints(prev => [ep, ...prev])
    return ep
  }

  async function deleteEndpoint(id: string) {
    await api.deleteEndpoint(id)
    setEndpoints(prev => prev.filter(ep => ep.id !== id))
  }

  return { endpoints, loading, error, createEndpoint, deleteEndpoint, refetch: fetch }
}