import type { Session } from '../../types'
import { CopyButton } from '../ui/CopyButton'
import { StatusDot } from '../ui/StatusDot'
import { ttlLabel } from '../../lib/format'
import type { ConnectionStatus } from '../../types'

interface Props {
  session: Session
  status: ConnectionStatus
  onReset: () => void
}

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export function SessionBar({ session, status, onReset }: Props) {
  const inboxUrl = `${BASE_URL}/i/${session.id}`

  return (
    <div className="p-4 border-b border-zinc-800 space-y-3">
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold text-zinc-400 uppercase tracking-wider">
          Your endpoint
        </span>
        <StatusDot status={status} />
      </div>

      <div className="flex items-center gap-2 bg-zinc-900 rounded-lg px-3 py-2">
        <span className="flex-1 font-mono text-xs text-emerald-400 truncate">
          {inboxUrl}
        </span>
        <CopyButton text={inboxUrl} />
      </div>

      <div className="flex items-center justify-between text-xs text-zinc-500">
        <span>{ttlLabel(session.expires_at)}</span>
        <button
          onClick={onReset}
          className="hover:text-zinc-300 transition-colors"
        >
          New session
        </button>
      </div>
    </div>
  )
}