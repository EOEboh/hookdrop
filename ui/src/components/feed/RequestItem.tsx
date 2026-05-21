import type { CapturedRequest } from '../../types'
import { MethodBadge } from '../ui/MethodBadge'
import { timeAgo, formatBytes } from '../../lib/format'
import { VerificationBadge } from '../ui/VerificationBadge'

interface Props {
  request: CapturedRequest
  selected: boolean
  onClick: () => void
}

export function RequestItem({ request, selected, onClick }: Props) {
  return (
    <button
      onClick={onClick}
      className={`relative w-full text-left px-4 py-3 border-b border-zinc-800/60 transition-colors group ${
        selected ? 'bg-emerald-500/[0.06]' : 'hover:bg-zinc-900'
      }`}
    >
      {/* Left accent bar — always present, no layout shift */}
      <span
        className={`absolute left-0 top-0 bottom-0 w-0.5 transition-colors ${
          selected ? 'bg-emerald-500' : 'bg-transparent group-hover:bg-zinc-700'
        }`}
      />

      <div className="flex items-start gap-3">
        <div className="pt-0.5 shrink-0">
          <MethodBadge method={request.method} size="sm" />
        </div>

        <div className="flex-1 min-w-0">
          {/* Path */}
          <p
            className={`text-xs font-mono truncate leading-tight transition-colors ${
              selected ? 'text-zinc-100' : 'text-zinc-400 group-hover:text-zinc-300'
            }`}
          >
            /i/{request.session_id}
          </p>

          {/* Meta row */}
          <div className="flex items-center gap-1.5 mt-1.5 flex-wrap">
            <span className="text-[11px] text-zinc-600 tabular-nums">
              {timeAgo(request.received_at)}
            </span>
            <span className="text-zinc-800 select-none text-[10px]">·</span>
            <span className="text-[11px] text-zinc-600">
              {formatBytes(request.body_size)}
            </span>
            {request.verified && request.verified !== 'unverified' && (
              <>
                <span className="text-zinc-800 select-none text-[10px]">·</span>
                <VerificationBadge status={request.verified} provider={request.provider} />
              </>
            )}
          </div>
        </div>
      </div>
    </button>
  )
}
