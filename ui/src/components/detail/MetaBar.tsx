import type { CapturedRequest } from '../../types'
import { MethodBadge } from '../ui/MethodBadge'
import { formatTime, formatBytes } from '../../lib/format'
import { VerificationBadge } from '../ui/VerificationBadge'

export function MetaBar({ request }: { request: CapturedRequest }) {
  return (
    <div className="px-6 py-4 border-b border-border bg-surface/50">
      <div className="flex items-center justify-between gap-4 flex-wrap">

        {/* Primary — dominates: method + verification */}
        <div className="flex items-center gap-3">
          <MethodBadge method={request.method} size="lg" />
          <div className="w-px h-5 bg-border shrink-0" />
          <VerificationBadge
            status={request.verified ?? 'unverified'}
            provider={request.provider}
            showProvider
          />
        </div>

        {/* Secondary — recedes: IP, time, size, ID */}
        <div className="flex items-center gap-2 font-mono text-xs text-muted flex-wrap">
          <span className="text-ink">{request.remote_ip}</span>
          <span className="text-border-strong">·</span>
          <span>{formatTime(request.received_at)}</span>
          <span className="text-border-strong">·</span>
          <span>{formatBytes(request.body_size)}</span>
          <span className="text-border-strong">·</span>
          <span className="text-faint">{request.id.slice(0, 8)}</span>
        </div>

      </div>
    </div>
  )
}
