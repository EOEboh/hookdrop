import type {
  CapturedRequest, ReplayRequest,
  ReplayResponse, Session
} from '../types'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

function getToken(): string | null {
  return localStorage.getItem('hookdrop_token')
}

function authHeaders(): Record<string, string> {
  const token = getToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

async function handle<T>(res: Response): Promise<T> {
  if (res.status === 401) {
    // Token expired — clear it and reload to trigger login
    localStorage.removeItem('hookdrop_token')
    window.location.href = '/'
  }
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`${res.status}: ${text}`)
  }
  return res.json() as Promise<T>
}

export const api = {
  requestMagicLink(email: string): Promise<{ message: string }> {
    return fetch(`${BASE_URL}/auth/request`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email }),
    }).then(handle<{ message: string }>)
  },

  createSession(): Promise<Session> {
    return fetch(`${BASE_URL}/sessions`, {
      method: 'POST',
      headers: { ...authHeaders() },
    }).then(handle<Session>)
  },

  getRequests(sessionId: string): Promise<CapturedRequest[]> {
    return fetch(`${BASE_URL}/requests/${sessionId}`, {
      headers: { ...authHeaders() },
    }).then(handle<CapturedRequest[]>)
  },

  replay(payload: ReplayRequest): Promise<ReplayResponse> {
    return fetch(`${BASE_URL}/replay`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...authHeaders(),
      },
      body: JSON.stringify(payload),
    }).then(handle<ReplayResponse>)
  },

  sseUrl(sessionId: string): string {
    const token = getToken()
    // Pass token as query param for SSE since EventSource doesn't support headers
    return `${BASE_URL}/events/${sessionId}?token=${token}`
  },
}