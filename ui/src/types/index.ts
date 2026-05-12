export interface AuthUser {
  id: string
  email: string
}

export interface AuthState {
  user: AuthUser | null
  token: string | null
  loading: boolean
}

export interface Session {
  id: string
  created_at: string
  expires_at: string
}

export interface CapturedRequest {
  id: string
  session_id: string
  method: string
  headers: Record<string, string>
  body: string        // base64 when binary, UTF-8 string otherwise
  body_size: number
  remote_ip: string
  received_at: string
}

export interface ReplayRequest {
  request_id: string
  target_url: string
  headers?: Record<string, string>  
  body?: string                     
}

export interface ReplayResponse {
  status: number
  headers: Record<string, string>
  body: string
  latency_ms: number
}

export interface Endpoint {
  id: string
  user_id: string
  slug: string
  name: string
  description?: string
  created_at: string
}

export type ConnectionStatus = 'connecting' | 'live' | 'disconnected'