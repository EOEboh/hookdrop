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

export interface Subscription {
  id: string
  user_id: string
  plan: 'free' | 'pro'
  provider: 'stripe' | 'paystack' | ''
  status: 'active' | 'trialing' | 'past_due' | 'canceled'
  current_period_end: string | null
  trial_end: string | null
  cancel_at_period_end: boolean
  currency: 'usd' | 'ngn'
  interval: 'month' | 'year'
}

export interface PlanLimits {
  max_named_endpoints: number
  max_requests_per_month: number
  history_days: number
  max_secrets: number
  has_filtering: boolean
}

export interface BillingState {
  subscription: Subscription | null
  limits: PlanLimits | null
  is_active: boolean
  loading: boolean
}

export type VerificationStatus = 'verified' | 'failed' | 'unverified'

export type ConnectionStatus = 'connecting' | 'live' | 'disconnected'