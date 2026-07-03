import type { CapturedRequest } from '../../types'
import { MethodBadge } from '../ui/MethodBadge'
import { formatTime, formatBytes } from '../../lib/format'
import { VerificationBadge } from '../ui/VerificationBadge'

export function MetaBar({ request }: { request: CapturedRequest }) {
  return (
    <div className="px-6 py-4 border-b border-zinc-800 bg-zinc-900/50">
      <div className="flex items-center justify-between gap-4 flex-wrap">

        {/* Primary — dominates: method + verification */}
        <div className="flex items-center gap-3">
          <MethodBadge method={request.method} size="lg" />
          <div className="w-px h-5 bg-zinc-800 shrink-0" />
          <VerificationBadge
            status={request.verified ?? 'unverified'}
            provider={request.provider}
            showProvider
          />
        </div>

        {/* Secondary — recedes: IP, time, size, ID */}
        <div className="flex items-center gap-2 font-mono text-xs text-zinc-500 flex-wrap">
          <span className="text-zinc-400">{request.remote_ip}</span>
          <span className="text-zinc-700">·</span>
          <span>{formatTime(request.received_at)}</span>
          <span className="text-zinc-700">·</span>
          <span>{formatBytes(request.body_size)}</span>
          <span className="text-zinc-700">·</span>
          <span className="text-zinc-600">{request.id.slice(0, 8)}</span>
        </div>

      </div>
    </div>
  )
}
