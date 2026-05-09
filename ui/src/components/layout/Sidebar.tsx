import type { CapturedRequest, ConnectionStatus, Session } from '../../types'
import { SessionBar } from '../session/SessionBar'
import { RequestList } from '../feed/RequestList'

interface Props {
  session: Session
  status: ConnectionStatus
  requests: CapturedRequest[]
  selectedId: string | null
  onSelect: (req: CapturedRequest) => void
  onReset: () => void
  onClear: () => void
  onLogout: () => void
}

export function Sidebar({ session, status, requests, selectedId, onSelect, onReset, onClear, onLogout }: Props) {
  return (
    <aside className="w-80 min-w-[280px] flex flex-col border-r border-zinc-800 bg-zinc-950 h-screen sticky top-0">
      {/* Logo */}
      <div className="px-4 py-4 border-b border-zinc-800 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-lg">⚡</span>
          <span className="font-semibold text-zinc-100 tracking-tight">hookdrop</span>
        </div>
        {requests.length > 0 && (
          <button
            onClick={onClear}
            className="text-xs text-zinc-600 hover:text-zinc-400 transition-colors"
          >
            Clear
          </button>
        )}
        // In Sidebar.tsx — add to the logo div
<button
  onClick={onLogout}
  className="text-xs text-zinc-600 hover:text-zinc-400 transition-colors"
>
  Log out
</button>
      </div>

      <SessionBar session={session} status={status} onReset={onReset} />

      {/* Request count */}
      {requests.length > 0 && (
        <div className="px-4 py-2 border-b border-zinc-800">
          <span className="text-xs text-zinc-600">
            {requests.length} request{requests.length !== 1 ? 's' : ''} captured
          </span>
        </div>
      )}

      <RequestList requests={requests} selectedId={selectedId} onSelect={onSelect} />
    </aside>
  )
}