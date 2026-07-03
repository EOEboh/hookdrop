import type { ReplayResponse as ReplayResponseType } from '../../types'
import { tryPrettyPrint } from '../../lib/json'

function statusColor(code: number): string {
  if (code < 300) return 'text-emerald-400'
  if (code < 400) return 'text-amber-400'
  return 'text-red-400'
}

function statusLabel(code: number): string {
  if (code >= 200 && code < 300) return 'Success'
  if (code >= 300 && code < 400) return 'Redirect'
  if (code >= 400 && code < 500) return 'Client error'
  if (code >= 500) return 'Server error'
  return 'Response'
}

export function ReplayResponse({ response }: { response: ReplayResponseType }) {
  const { pretty } = tryPrettyPrint(response.body)

  return (
    <div className="space-y-3">

      {/* Status + latency — prominent */}
      <div className="flex items-center gap-3 bg-zinc-900 border border-zinc-800 rounded-lg px-4 py-3">
        <span className={`font-mono font-bold text-lg tabular-nums ${statusColor(response.status)}`}>
          {response.status}
        </span>
        <div className="w-px h-5 bg-zinc-800 shrink-0" />
        <div className="flex items-baseline gap-1">
          <span className="text-sm font-medium text-zinc-300 tabular-nums">
            {response.latency_ms}
          </span>
          <span className="text-xs text-zinc-600">ms</span>
        </div>
        <span className="ml-auto text-xs text-zinc-600">
          {statusLabel(response.status)}
        </span>
      </div>

      {/* Body — contained and scrollable */}
      {response.body && (
        <pre className="text-xs font-mono bg-zinc-900 border border-zinc-800 rounded-lg px-4 py-3 overflow-y-auto max-h-52 text-zinc-300 whitespace-pre-wrap leading-relaxed">
          {pretty}
        </pre>
      )}

    </div>
  )
}
