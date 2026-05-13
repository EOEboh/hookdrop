import { useState } from 'react'
import type { CapturedRequest, ConnectionStatus, Endpoint, Session } from '../../types'
import { SessionBar } from '../session/SessionBar'
import { RequestList } from '../feed/RequestList'
import { EndpointList } from '../endpoints/EndpointList'
import { CreateEndpointModal } from '../endpoints/CreateEndpointModal'
import { useEndpoints } from '../../hooks/useEndpoints'

interface Props {
  session: Session
  status: ConnectionStatus
  requests: CapturedRequest[]          
  selectedId: string | null
  onSelect: (req: CapturedRequest) => void
  onReset: () => void
  onClear: () => void
  onLogout: () => void
  onSelectEndpoint: (ep: Endpoint) => void
  onBackToTemporary: () => void
  selectedEndpointId: string | null
  activeEndpoint: Endpoint | null      
}

type Tab = 'session' | 'endpoints'

export function Sidebar({
  session, status, requests, selectedId,
  onSelect, onReset, onClear, onLogout,
  onSelectEndpoint, selectedEndpointId, activeEndpoint, onBackToTemporary
}: Props) {
  const [tab, setTab]           = useState<Tab>('session')
  const [showModal, setShowModal] = useState(false)
  const { endpoints, createEndpoint, deleteEndpoint } = useEndpoints()

  return (
    <aside className="w-80 min-w-[280px] flex flex-col border-r border-zinc-800 bg-zinc-950 h-screen sticky top-0">

      {/* Logo */}
      <div className="px-4 py-4 border-b border-zinc-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-lg">⚡</span>
          <span className="font-semibold text-zinc-100 tracking-tight">hookdrop</span>
        </div>
        <button
          onClick={onLogout}
          className="text-xs text-zinc-600 hover:text-zinc-400 transition-colors"
        >
          Log out
        </button>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-zinc-800">
        {(['session', 'endpoints'] as Tab[]).map(t => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`flex-1 py-2.5 text-xs font-medium transition-colors capitalize ${
              tab === t
                ? 'text-emerald-400 border-b-2 border-emerald-400'
                : 'text-zinc-500 hover:text-zinc-300'
            }`}
          >
            {t === 'session' ? 'Temporary' : 'Named'}
          </button>
        ))}
      </div>

     {tab === 'session' ? (
  <>
    <SessionBar session={session} status={status} onReset={onReset} />
    {requests.length > 0 && (
      <div className="px-4 py-2 border-b border-zinc-800 flex items-center justify-between">
        <span className="text-xs text-zinc-600">
          {requests.length} request{requests.length !== 1 ? 's' : ''}
        </span>
        <button
          onClick={onClear}
          className="text-xs text-zinc-600 hover:text-zinc-400 transition-colors"
        >
          Clear
        </button>
      </div>
    )}
    <RequestList
      requests={requests}
      selectedId={selectedId}
      onSelect={onSelect}
    />
  </>
) : (
  <>
    <div className="p-3 border-b border-zinc-800">
      <button
        onClick={() => setShowModal(true)}
        className="w-full py-2 rounded-lg border border-dashed border-zinc-700 hover:border-emerald-500 text-xs text-zinc-500 hover:text-emerald-400 transition-colors"
      >
        + New named endpoint
      </button>
    </div>

    {/* Show requests for the active named endpoint */}
    {activeEndpoint ? (
      <>
        <div className="px-4 py-2 border-b border-zinc-800 flex items-center justify-between">
          <div className="min-w-0">
            <p className="text-xs font-medium text-zinc-300 truncate">{activeEndpoint.name}</p>
            <p className="text-xs font-mono text-emerald-500">/i/{activeEndpoint.slug}</p>
          </div>
          <button
            onClick={() => { onBackToTemporary(); setTab('session') }}
            className="text-xs text-zinc-600 hover:text-zinc-400 shrink-0 ml-2"
          >
            ← Back
          </button>
        </div>
        {requests.length > 0 && (
          <div className="px-4 py-2 border-b border-zinc-800">
            <span className="text-xs text-zinc-600">
              {requests.length} request{requests.length !== 1 ? 's' : ''}
            </span>
          </div>
        )}
        <RequestList
          requests={requests}
          selectedId={selectedId}
          onSelect={onSelect}
        />
      </>
    ) : (
      <div className="flex-1 overflow-y-auto">
        <EndpointList
          endpoints={endpoints}
          selectedId={selectedEndpointId}
          onSelect={(ep) => {
            onSelectEndpoint(ep)  // activates the feed
          }}
          onDelete={deleteEndpoint}
        />
      </div>
    )}
  </>
)}

      {showModal && (
        <CreateEndpointModal
          onClose={() => setShowModal(false)}
          onCreate={createEndpoint}
        />
      )}
    </aside>
  )
}