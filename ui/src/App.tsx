import { useEffect, useState } from 'react'
import { useAuth } from './context/AuthContext'
import { useSession } from './hooks/useSession'
import { useRequestFeed } from './hooks/useRequestFeed'
import { Sidebar } from './components/layout/Sidebar'
import { MainPanel } from './components/layout/MainPanel'
import { MobileTabBar, type MobileTab } from './components/layout/MobileTabBar'
import { LandingPage } from './pages/LandingPage'
import { AuthCallbackPage } from './pages/AuthCallbackPage'
import { PricingPage } from './components/billing/PricingPage'
import { Spinner } from './components/ui/Spinner'
import { Logo, LogoMark } from './components/ui/Logo'
import { DEFAULT_FILTERS, type CapturedRequest, type Endpoint, type RequestFilters } from './types'
import { usePostHog } from '@posthog/react'
import { PrivacyPage } from './pages/PrivacyPage'
import { TermsPage } from './pages/TermsPage'
import { TourProvider } from './components/onboarding/TourProvider'

export default function App() {
    if (window.location.pathname === '/privacy') return <PrivacyPage />
  if (window.location.pathname === '/terms')   return <TermsPage />

  if (window.location.pathname === '/auth/callback') return <AuthCallbackPage />

  const { user, loading: authLoading, logout } = useAuth()


  // Special routes handled before auth check 
  if (window.location.pathname === '/auth/callback') {
    return <AuthCallbackPage />
  }

  if (authLoading) {
    return (
      <div className="min-h-screen bg-base flex items-center justify-center">
        <Spinner size={5} />
      </div>
    )
  }

  const urlError = new URLSearchParams(window.location.search).get('error')
  if (!user) return <LandingPage errorHint={urlError} />

  //  Authenticated routes 
  if (window.location.pathname === '/settings/billing') {
    return <BillingPageShell onLogout={logout} />
  }

  return <AuthenticatedApp onLogout={logout} />
}

// Thin shell that wraps PricingPage with a back button and consistent chrome
function BillingPageShell({ onLogout }: { onLogout: () => void }) {
  return (
    <div className="min-h-screen bg-base text-ink">

      {/* Top bar */}
      <div className="border-b border-border px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button
            onClick={() => window.history.back()}
            className="text-muted hover:text-ink text-sm transition-colors duration-200 ease-(--ease-considered) flex items-center gap-1.5"
          >
            ← Back
          </button>
          <Logo size="sm" />
        </div>
        <button
          onClick={onLogout}
          className="text-xs text-faint hover:text-muted transition-colors duration-200 ease-(--ease-considered)"
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
  const posthog = usePostHog()
  const { session, loading, error, resetSession } = useSession()
  const [activeEndpoint, setActiveEndpoint]   = useState<Endpoint | null>(null)
  const [selectedRequest, setSelectedRequest] = useState<CapturedRequest | null>(null)
  const [filters, setFilters]                 = useState<RequestFilters>(DEFAULT_FILTERS)
  const [mobileTab, setMobileTab]             = useState<MobileTab>('requests')
  const [showUpgradeBanner, setShowUpgradeBanner] = useState(
    new URLSearchParams(window.location.search).get('upgraded') === 'true'
  )

  useEffect(() => {
    if (showUpgradeBanner) {
      // Clean the URL without reloading
      window.history.replaceState(null, '', '/')
      const t = setTimeout(() => setShowUpgradeBanner(false), 8000)
      return () => clearTimeout(t)
    }
  }, [showUpgradeBanner])

  const activeFeedId = activeEndpoint ? activeEndpoint.id : session?.id ?? null
  const { requests, status, clearRequests, newIds } = useRequestFeed(activeFeedId, filters)

  if (loading) {
    return (
      <div className="min-h-screen bg-base flex items-center justify-center gap-3">
        <Spinner size={5} />
        <span className="text-muted text-sm">Starting session…</span>
      </div>
    )
  }

  if (error || !session) {
    return (
      <div className="min-h-screen bg-base flex items-center justify-center">
        <div className="text-center space-y-3">
          <p className="text-red-400 text-sm">{error ?? 'Failed to start session'}</p>
          <button onClick={resetSession} className="text-xs text-muted hover:text-ink underline transition-colors duration-200 ease-(--ease-considered)">
            Try again
          </button>
        </div>
      </div>
    )
  }

   function handleSelectRequest(req: CapturedRequest) {
    setSelectedRequest(req)
    setMobileTab('detail')
    posthog?.capture('request_inspected', {             
      method:   req.method,
      verified: req.verified,
      provider: req.provider,
    })
  }

  return (
    <TourProvider>
    <div className="flex flex-col min-h-screen bg-base text-ink">

      {/* Upgrade banner: sits above everything, dismissible */}
      {showUpgradeBanner && (
        <div className="flex items-center justify-between gap-4 px-5 py-2.5 bg-indigo-600 text-white text-sm shrink-0">
          <div className="flex items-center gap-3">
            <span className="font-semibold flex items-center gap-1.5">
              <LogoMark size="sm" /> You're on hookdrop Pro
            </span>
            <span className="text-indigo-100 text-xs hidden sm:block">
              Unlimited named endpoints, 50k requests/month, and 90 day history are now active.
            </span>
          </div>
          <button
            onClick={() => setShowUpgradeBanner(false)}
            className="text-indigo-100 hover:text-white transition-colors duration-200 ease-(--ease-considered) shrink-0 text-lg leading-none"
            aria-label="Dismiss"
          >
            ×
          </button>
        </div>
      )}

      <div className="flex flex-1 min-h-0 pb-[52px] lg:pb-0">
        <div className={`${mobileTab === 'detail' ? 'hidden' : 'flex'} lg:flex w-full lg:w-auto`}>
          <Sidebar
            session={session}
            status={status}
            requests={requests}
            newIds={newIds}
            selectedId={selectedRequest?.id ?? null}
            onSelect={handleSelectRequest}
            onReset={() => { resetSession(); setSelectedRequest(null) }}
            onClear={clearRequests}
            onLogout={onLogout}
            onSelectEndpoint={(ep) => {
              setActiveEndpoint(ep)
              setSelectedRequest(null)
              setFilters(DEFAULT_FILTERS)
            }}
            onBackToTemporary={() => {
              setActiveEndpoint(null)
              setSelectedRequest(null)
              setFilters(DEFAULT_FILTERS)
            }}
            selectedEndpointId={activeEndpoint?.id ?? null}
            activeEndpoint={activeEndpoint}
            filters={filters}
            onFilterChange={setFilters}
            totalRequestCount={requests.length}
            hideTabSwitcher
            controlledTab={mobileTab === 'endpoints' ? 'endpoints' : 'session'}
            onTabChange={(t) => setMobileTab(t === 'endpoints' ? 'endpoints' : 'requests')}
          />
        </div>
        <div data-tour="detail-panel" className={`${mobileTab === 'detail' ? 'flex' : 'hidden'} lg:flex flex-1 min-w-0`}>
          <MainPanel selected={selectedRequest} activeEndpoint={activeEndpoint} />
        </div>
      </div>

      <MobileTabBar
        active={mobileTab}
        onChange={setMobileTab}
        hasSelection={!!selectedRequest || !!activeEndpoint}
      />
    </div>
    </TourProvider>
  )
}