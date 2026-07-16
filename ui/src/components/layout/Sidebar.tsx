import { useState } from 'react'
import type { CapturedRequest, ConnectionStatus, Endpoint, RequestFilters, Session } from '../../types'
import { SessionBar } from '../session/SessionBar'
import { RequestList } from '../feed/RequestList'
import { EndpointList } from '../endpoints/EndpointList'
import { CreateEndpointModal } from '../endpoints/CreateEndpointModal'
import { useEndpoints } from '../../hooks/useEndpoints'
import { PlusIcon } from '../ui/icons'
import { useBilling } from '../../context/BillingContext'
import { usePostHog } from '@posthog/react'
import { Logo } from '../ui/Logo'
import { TourHelpButton } from '../onboarding/TourHelpButton'

interface Props {
  session: Session
  status: ConnectionStatus
  requests: CapturedRequest[]
  newIds: Set<string>
  selectedId: string | null
  onSelect: (req: CapturedRequest) => void
  onReset: () => void
  onClear: () => void
  onLogout: () => void
  onSelectEndpoint: (ep: Endpoint) => void
  onBackToTemporary: () => void
  selectedEndpointId: string | null
  activeEndpoint: Endpoint | null
  filters: RequestFilters
  onFilterChange: (f: RequestFilters) => void
  totalRequestCount: number
  controlledTab?: Tab
  onTabChange?: (t: Tab) => void
  hideTabSwitcher?: boolean
}

export type Tab = 'session' | 'endpoints'

export function Sidebar({
  session, status, requests, newIds, selectedId,
  onSelect, onReset, onClear, onLogout,
  onSelectEndpoint, selectedEndpointId, activeEndpoint, onBackToTemporary, filters, onFilterChange, totalRequestCount,
  controlledTab, onTabChange, hideTabSwitcher = false,
}: Props) {
  const posthog = usePostHog()
  const [internalTab, setInternalTab] = useState<Tab>('session')
  const tab = controlledTab ?? internalTab
  function setTab(t: Tab) {
    if (onTabChange) onTabChange(t)
    else setInternalTab(t)
  }
  const [showModal, setShowModal] = useState(false)
  const { endpoints, createEndpoint, deleteEndpoint } = useEndpoints()
  const { isPro } = useBilling()


   function handleLogout() {
    posthog?.capture('user_logged_out')
    onLogout()
  }

  return (
    <aside className="w-full lg:w-80 lg:min-w-[280px] flex flex-col border-r border-border bg-base h-full lg:h-screen lg:sticky lg:top-0">

{/* Logo */}
      <div className="px-4 py-4 border-b border-border flex items-center justify-between">
        <Logo size="sm" />
        <div className="flex items-center gap-3">
          <a
           href="/settings/billing"
            className={`text-xs transition-colors duration-200 ease-(--ease-considered) ${
              isPro
                ? 'text-indigo-400 hover:text-indigo-300'
                : 'text-faint hover:text-muted'
            }`}
          >
            {isPro ? '⚡ Pro' : 'Upgrade'}
          </a>
          <a
           href="/settings/tokens"
            className="text-xs text-faint hover:text-muted transition-colors duration-200 ease-(--ease-considered)"
          >
            API tokens
          </a>
          <a
           href="https://status.hookdrop.app"
      target="_blank"
      rel="noopener noreferrer"
      className="text-xs text-faint hover:text-muted transition-colors duration-200 ease-(--ease-considered)"
    >
      Status
    </a>
          <button
            onClick={handleLogout}
            className="text-xs text-faint hover:text-muted transition-colors duration-200 ease-(--ease-considered)"
          >
            Log out
          </button>
          <TourHelpButton />
        </div>
      </div>
      

      {/* Tabs — hidden on mobile where MobileTabBar takes over this role */}
      <div className={`${hideTabSwitcher ? 'hidden lg:flex' : 'flex'} border-b border-border`}>
        {(['session', 'endpoints'] as Tab[]).map(t => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`flex-1 py-2.5 text-xs font-medium transition-colors duration-200 ease-(--ease-considered) capitalize ${
              tab === t
                ? 'text-indigo-400 border-b-2 border-indigo-400'
                : 'text-muted hover:text-ink'
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
      <div className="px-4 py-2 border-b border-border flex items-center justify-between">
        <span className="text-xs text-faint">
          {requests.length} request{requests.length !== 1 ? 's' : ''}
        </span>
        <button
          onClick={onClear}
          className="text-xs text-faint hover:text-muted transition-colors duration-200 ease-(--ease-considered)"
        >
          Clear
        </button>
      </div>
    )}

    <RequestList
      requests={requests}
      newIds={newIds}
      allCount={totalRequestCount}
      selectedId={selectedId}
      onSelect={onSelect}
      filters={filters}
      onFilterChange={onFilterChange}
    />
  </>
) : (
  <>
    <div className="p-3 border-b border-border">
      <button
        onClick={() => setShowModal(true)}
        className="w-full py-2.5 rounded-lg border border-dashed border-border-strong hover:border-indigo-500/50 hover:bg-indigo-500/[0.04] active:scale-[0.99] text-xs text-muted hover:text-indigo-400 transition-all duration-200 ease-(--ease-considered) flex items-center justify-center gap-2"
      >
        <PlusIcon className="w-3.5 h-3.5" />
        New named endpoint
      </button>
    </div>

    {/* Show requests for the active named endpoint */}
    {activeEndpoint ? (
      <>
        <div className="px-4 py-2 border-b border-border flex items-center justify-between">
          <div className="min-w-0">
            <p className="text-xs font-medium text-ink truncate">{activeEndpoint.name}</p>
            <p className="text-xs font-mono text-indigo-400">/i/{activeEndpoint.slug}</p>
          </div>
          <button
            onClick={() => { onBackToTemporary(); setTab('session') }}
            className="text-xs text-faint hover:text-muted transition-colors duration-200 ease-(--ease-considered) shrink-0 ml-2"
          >
            ← Back
          </button>
        </div>
        {requests.length > 0 && (
          <div className="px-4 py-2 border-b border-border">
            <span className="text-xs text-faint">
              {requests.length} request{requests.length !== 1 ? 's' : ''}
            </span>
          </div>
        )}

      <RequestList
        requests={requests}
        newIds={newIds}
        allCount={totalRequestCount}
        selectedId={selectedId}
        onSelect={onSelect}
        filters={filters}
        onFilterChange={onFilterChange}
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
