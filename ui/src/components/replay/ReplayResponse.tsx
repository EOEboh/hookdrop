import type { ReplayResponse as ReplayResponseType } from '../../types'
import { tryPrettyPrint } from '../../lib/json'

function statusColor(code: number) {
  if (code < 300) return 'text-emerald-400'
  if (code < 400) return 'text-amber-400'
  return 'text-red-400'
}

export function ReplayResponse({ response }: { response: ReplayResponseType }) {
  const { pretty } = tryPrettyPrint(response.body)

  return (
    <div className="mt-4 space-y-3">
      <div className="flex items-center gap-4">
        <span className={`font-mono font-medium text-sm ${statusColor(response.status)}`}>
          {response.status}
        </span>
        <span className="text-xs text-zinc-500">{response.latency_ms}ms</span>
      </div>
      {response.body && (
        <pre className="text-xs font-mono bg-zinc-900 rounded-lg p-4 overflow-x-auto text-zinc-300 whitespace-pre-wrap max-h-48">
          {pretty}
        </pre>
      )}
    </div>
  )
}