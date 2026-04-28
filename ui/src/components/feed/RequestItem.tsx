import type { CapturedRequest } from '../../types'
import { MethodBadge } from '../ui/MethodBadge'
import { timeAgo, formatBytes } from '../../lib/format'

interface Props {
  request: CapturedRequest
  selected: boolean
  onClick: () => void
}

export function RequestItem({ request, selected, onClick }: Props) {
  return (
    <button
      onClick={onClick}
      className={`w-full text-left px-4 py-3 border-b border-zinc-800/60 hover:bg-zinc-800/40 transition-colors flex items-start gap-3 ${
        selected ? 'bg-zinc-800/60 border-l-2 border-l-emerald-500' : ''
      }`}
    >
      <div className="pt-0.5">
        <MethodBadge method={request.method} />
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-xs font-mono text-zinc-300 truncate">
          /i/{request.session_id}
        </p>
        <div className="flex items-center gap-2 mt-1">
          <span className="text-xs text-zinc-600">{timeAgo(request.received_at)}</span>
          <span className="text-xs text-zinc-700">·</span>
          <span className="text-xs text-zinc-600">{formatBytes(request.body_size)}</span>
        </div>
      </div>
    </button>
  )
}