import { useState } from 'react'
import { useAuth } from './context/AuthContext'
import { useSession } from './hooks/useSession'
import { useRequestFeed } from './hooks/useRequestFeed'
import { Sidebar } from './components/layout/Sidebar'
import { MainPanel } from './components/layout/MainPanel'
import { LoginPage } from './pages/LoginPage'
import { AuthCallbackPage } from './pages/AuthCallbackPage'
import { PricingPage } from './components/billing/PricingPage'
import { Spinner } from './components/ui/Spinner'
import { DEFAULT_FILTERS, type CapturedRequest, type Endpoint, type RequestFilters } from './types'

export default function App() {
  const { user, loading: authLoading, logout } = useAuth()

  // Special routes handled before auth check 
  if (window.location.pathname === '/auth/callback') {
    return <AuthCallbackPage />
  }

  if (authLoading) {
    return (
      <div className="min-h-screen bg-zinc-950 flex items-center justify-center">
        <Spinner size={5} />
      </div>
    )
  }

  const urlError = new URLSearchParams(window.location.search).get('error')
  if (!user) return <LoginPage errorHint={urlError} />

  //  Authenticated routes 
  if (window.location.pathname === '/settings/billing') {
    return <BillingPageShell onLogout={logout} />
  }

  return <AuthenticatedApp onLogout={logout} />
}

// Thin shell that wraps PricingPage with a back button and consistent chrome
function BillingPageShell({ onLogout }: { onLogout: () => void }) {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">

      {/* Top bar */}
      <div className="border-b border-zinc-800 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button
            onClick={() => window.history.back()}
            className="text-zinc-500 hover:text-zinc-300 text-sm transition-colors flex items-center gap-1.5"
          >
            ← Back
          </button>
          <div className="flex items-center gap-2">
            <span className="text-lg">⚡</span>
            <span className="font-semibold text-zinc-100 tracking-tight">hookdrop</span>
          </div>
        </div>
        <button
          onClick={onLogout}
          className="text-xs text-zinc-600 hover:text-zinc-400 transition-colors"
        >
          Log out
        </button>
      </div>

      {/* Page content */}
      <PricingPage />
    </div>
  )
}

function AuthenticatedApp({ onLogout }: { onLogout: () => void }) {
  const { session, loading, error, resetSession } = useSession()

  const [activeEndpoint, setActiveEndpoint]   = useState<Endpoint | null>(null)
  const [selectedRequest, setSelectedRequest] = useState<CapturedRequest | null>(null)
  const [filters, setFilters]                 = useState<RequestFilters>(DEFAULT_FILTERS)

  const activeFeedId = activeEndpoint ? activeEndpoint.id : session?.id ?? null
  const { requests, status, clearRequests, totalCount } = useRequestFeed(activeFeedId, filters)

  function handleSelectEndpoint(ep: Endpoint) {
    setActiveEndpoint(ep)
    setSelectedRequest(null)
    setFilters(DEFAULT_FILTERS)
  }

  function handleSelectTemporary() {
    setActiveEndpoint(null)
    setSelectedRequest(null)
    setFilters(DEFAULT_FILTERS)
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-zinc-950 flex items-center justify-center gap-3">
        <Spinner size={5} />
        <span className="text-zinc-400 text-sm">Starting session…</span>
      </div>
    )
  }

  if (error || !session) {
    return (
      <div className="min-h-screen bg-zinc-950 flex items-center justify-center">
        <div className="text-center space-y-3">
          <p className="text-red-400 text-sm">{error ?? 'Failed to start session'}</p>
          <button
            onClick={resetSession}
            className="text-xs text-zinc-400 hover:text-zinc-200 underline"
          >
            Try again
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen bg-zinc-950 text-zinc-100">
      <Sidebar
        session={session}
        status={status}
        requests={requests}
        selectedId={selectedRequest?.id ?? null}
        onSelect={setSelectedRequest}
        onReset={() => { resetSession(); setSelectedRequest(null) }}
        onClear={clearRequests}
        onLogout={onLogout}
        onSelectEndpoint={handleSelectEndpoint}
        onBackToTemporary={handleSelectTemporary}
        selectedEndpointId={activeEndpoint?.id ?? null}
        activeEndpoint={activeEndpoint}
        filters={filters}
        onFilterChange={setFilters}
        totalRequestCount={totalCount}
      />
      <MainPanel selected={selectedRequest} />
    </div>
  )
}