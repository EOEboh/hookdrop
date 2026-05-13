import { useState } from 'react'
import { useAuth } from './context/AuthContext'
import { useSession } from './hooks/useSession'
import { useRequestFeed } from './hooks/useRequestFeed'
import { Sidebar } from './components/layout/Sidebar'
import { MainPanel } from './components/layout/MainPanel'
import { LoginPage } from './pages/LoginPage'
import { AuthCallbackPage } from './pages/AuthCallbackPage'
import { Spinner } from './components/ui/Spinner'
import type { CapturedRequest, Endpoint } from './types'

export default function App() {
  const { user, loading: authLoading, logout } = useAuth()

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

  return <AuthenticatedApp onLogout={logout} />
}

function AuthenticatedApp({ onLogout }: { onLogout: () => void }) {
  const { session, loading, error, resetSession } = useSession()

  // Active feed source — null means use the temporary session
  const [activeEndpoint, setActiveEndpoint] = useState<Endpoint | null>(null)
  const [selectedRequest, setSelectedRequest] = useState<CapturedRequest | null>(null)

  // Single feed — switches between temp session and named endpoint
  const activeFeedId = activeEndpoint ? activeEndpoint.id : session?.id ?? null
  const { requests, status, clearRequests } = useRequestFeed(activeFeedId)

  // When switching feed source, clear the selected request
  function handleSelectEndpoint(ep: Endpoint) {
    setActiveEndpoint(ep)
    setSelectedRequest(null)
  }

  function handleSelectTemporary() {
    setActiveEndpoint(null)
    setSelectedRequest(null)
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
      />
      <MainPanel selected={selectedRequest} />
    </div>
  )
}