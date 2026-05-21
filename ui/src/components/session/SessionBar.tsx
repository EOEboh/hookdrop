import type { Session, ConnectionStatus } from '../../types'
import { CopyButton } from '../ui/CopyButton'
import { StatusDot } from '../ui/StatusDot'
import { ClockIcon, RefreshCwIcon } from '../ui/icons'
import { ttlLabel } from '../../lib/format'

interface Props {
  session: Session
  status: ConnectionStatus
  onReset: () => void
}

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export function SessionBar({ session, status, onReset }: Props) {
  const inboxUrl = `${BASE_URL}/i/${session.id}`

  return (
    <div className="px-4 py-3 border-b border-zinc-800 space-y-3">

      {/* Header row */}
      <div className="flex items-center justify-between">
        <span className="text-[11px] font-semibold text-zinc-500 uppercase tracking-widest">
          Endpoint
        </span>
        <StatusDot status={status} />
      </div>

      {/* URL block — primary affordance */}
      <div className="bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2.5 flex items-center gap-2 hover:border-zinc-700 transition-colors">
        <code className="flex-1 font-mono text-xs text-emerald-400 truncate select-all leading-relaxed">
          {inboxUrl}
        </code>
        <CopyButton text={inboxUrl} iconOnly />
      </div>

      {/* TTL + reset */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5 text-[11px] text-zinc-600">
          <ClockIcon className="w-3 h-3 shrink-0" />
          <span className="tabular-nums">{ttlLabel(session.expires_at)}</span>
        </div>
        <button
          onClick={onReset}
          className="flex items-center gap-1 text-[11px] text-zinc-600 hover:text-zinc-400 transition-colors"
        >
          <RefreshCwIcon className="w-3 h-3" />
          New session
        </button>
      </div>

    </div>
  )
}
