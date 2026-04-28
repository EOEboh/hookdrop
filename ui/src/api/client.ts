import type { CapturedRequest, ReplayRequest, ReplayResponse, Session } from '../types'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

async function handle<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`${res.status}: ${text}`)
  }
  return res.json() as Promise<T>
}

export const api = {
  createSession(): Promise<Session> {
    return fetch(`${BASE_URL}/sessions`, { method: 'POST' }).then(handle<Session>)
  },

  getRequests(sessionId: string): Promise<CapturedRequest[]> {
    return fetch(`${BASE_URL}/requests/${sessionId}`).then(handle<CapturedRequest[]>)
  },

  replay(payload: ReplayRequest): Promise<ReplayResponse> {
    return fetch(`${BASE_URL}/replay`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }).then(handle<ReplayResponse>)
  },

  sseUrl(sessionId: string): string {
    return `${BASE_URL}/events/${sessionId}`
  },
}