import { useState } from 'react'
import { useSession } from './hooks/useSession'
import { useRequestFeed } from './hooks/useRequestFeed'
import { Sidebar } from './components/layout/Sidebar'
import { MainPanel } from './components/layout/MainPanel'
import { Spinner } from './components/ui/Spinner'
import type { CapturedRequest } from './types'

export default function App() {
  const { session, loading, error, resetSession } = useSession()
  const { requests, status, clearRequests } = useRequestFeed(session?.id ?? null)
  const [selected, setSelected] = useState<CapturedRequest | null>(null)

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
        selectedId={selected?.id ?? null}
        onSelect={setSelected}
        onReset={() => { resetSession(); setSelected(null) }}
        onClear={clearRequests}
      />
      <MainPanel selected={selected} />
    </div>
  )
}