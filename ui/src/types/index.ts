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
  body: string
  body_size: number
  remote_ip: string
  received_at: string
  verified: VerificationStatus
  provider: string
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

export interface WebhookSecret {
  id: string
  endpoint_id: string
  provider: 'stripe' | 'paystack' | 'github' | 'generic'
  created_at: string
  // secret value is never returned from the API
}

export interface RequestFilters {
  search: string
  method: string       // '' = all
  verified: string     // '' = all
  range: string        // '' | '1h' | '24h' | '7d'
}

export const DEFAULT_FILTERS: RequestFilters = {
  search:   '',
  method:   '',
  verified: '',
  range:    '',
}

export type VerificationStatus = 'verified' | 'failed' | 'unverified'

export type ConnectionStatus = 'connecting' | 'live' | 'disconnected'