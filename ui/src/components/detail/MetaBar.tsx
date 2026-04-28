import type { CapturedRequest } from '../../types'
import { MethodBadge } from '../ui/MethodBadge'
import { formatTime, formatBytes } from '../../lib/format'

export function MetaBar({ request }: { request: CapturedRequest }) {
  return (
    <div className="flex items-center gap-4 px-6 py-3 border-b border-zinc-800 bg-zinc-900/60 flex-wrap">
      <MethodBadge method={request.method} />
      <span className="text-xs text-zinc-400 font-mono">{request.remote_ip}</span>
      <span className="text-xs text-zinc-600">·</span>
      <span className="text-xs text-zinc-400">{formatTime(request.received_at)}</span>
      <span className="text-xs text-zinc-600">·</span>
      <span className="text-xs text-zinc-400">{formatBytes(request.body_size)}</span>
      <span className="text-xs text-zinc-600">·</span>
      <span className="text-xs font-mono text-zinc-500">id: {request.id.slice(0, 8)}</span>
    </div>
  )
}