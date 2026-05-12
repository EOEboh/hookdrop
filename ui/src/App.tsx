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
  const [selected, setSelected] = useState<CapturedRequest | null>(null)



  // Handle the magic link callback route
  if (window.location.pathname === '/auth/callback') {
    return <AuthCallbackPage />
  }

  const urlError = new URLSearchParams(window.location.search).get('error')

  if (authLoading) {
  return (
    <div className="min-h-screen bg-zinc-950 flex items-center justify-center gap-3">
      <Spinner size={5} />
    </div>
  )
}

if (!user) {
  return <LoginPage errorHint={urlError} />
}

  return <AuthenticatedApp selected={selected} setSelected={setSelected} onLogout={logout} />
}

function AuthenticatedApp({
  selected,
  setSelected,
  onLogout,
}: {
  selected: CapturedRequest | null
  setSelected: (r: CapturedRequest | null) => void
  onLogout: () => void
}) {
  const { session, loading, error, resetSession } = useSession()
  const { requests, status, clearRequests } = useRequestFeed(session?.id ?? null)

    const [selectedEndpoint, setSelectedEndpoint] = useState<Endpoint | null>(null)
// const { requests: endpointRequests, status: endpointStatus } = useRequestFeed(
//   selectedEndpoint?.id ?? null
// )

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
          <button onClick={resetSession} className="text-xs text-zinc-400 hover:text-zinc-200 underline">
            Try again
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen bg-zinc-950 text-zinc-100">
      <Sidebar
      onSelectEndpoint={setSelectedEndpoint}
selectedEndpointId={selectedEndpoint?.id ?? null}
        session={session}
        status={status}
        requests={requests}
        selectedId={selected?.id ?? null}
        onSelect={setSelected}
        onReset={() => { resetSession(); setSelected(null) }}
        onClear={clearRequests}
        onLogout={onLogout}
      />
      <MainPanel selected={selected} />
    </div>
  )
}