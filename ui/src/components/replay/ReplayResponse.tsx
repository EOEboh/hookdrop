import type { ReplayResponse as ReplayResponseType } from '../../types'
import { tryPrettyPrint } from '../../lib/json'
import { CheckCircleIcon, AlertTriangleIcon } from '../ui/icons'

type Outcome = 'success' | 'redirect' | 'error'

function outcomeOf(code: number): Outcome {
  if (code < 300) return 'success'
  if (code < 400) return 'redirect'
  return 'error'
}

const OUTCOME_STYLES: Record<Outcome, {
  card: string
  status: string
  label: string
  Icon: typeof CheckCircleIcon | null
  iconClass: string
}> = {
  success: {
    card:  'bg-emerald-500/[0.06] border-emerald-500/25 shadow-[0_0_28px_-14px_rgba(16,185,129,0.5)]',
    status: 'text-emerald-400',
    label: 'text-emerald-400',
    Icon: CheckCircleIcon,
    iconClass: 'text-emerald-400',
  },
  redirect: {
    card:  'bg-surface border-border',
    status: 'text-amber-400',
    label: 'text-amber-400',
    Icon: null,
    iconClass: '',
  },
  error: {
    card:  'bg-red-500/[0.06] border-red-500/25',
    status: 'text-red-400',
    label: 'text-red-400',
    Icon: AlertTriangleIcon,
    iconClass: 'text-red-400',
  },
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
  const outcome = outcomeOf(response.status)
  const style = OUTCOME_STYLES[outcome]
  const Icon = style.Icon

  return (
    <div className="space-y-3">

      {/* Status + latency — outcome instantly legible before reading any text */}
      <div className={`flex items-center gap-3 border rounded-lg px-4 py-3 transition-colors duration-200 ease-(--ease-considered) ${style.card}`}>
        {Icon && <Icon className={`w-5 h-5 shrink-0 ${style.iconClass}`} />}
        <span className={`font-mono font-bold text-lg tabular-nums ${style.status}`}>
          {response.status}
        </span>
        <div className="w-px h-5 bg-border shrink-0" />
        <div className="flex items-baseline gap-1">
          <span className="text-sm font-medium text-ink tabular-nums">
            {response.latency_ms}
          </span>
          <span className="text-xs text-faint">ms</span>
        </div>
        <span className={`ml-auto text-xs font-medium ${style.label}`}>
          {statusLabel(response.status)}
        </span>
      </div>

      {/* Body — contained and scrollable */}
      {response.body && (
        <pre className="text-xs font-mono bg-surface border border-border rounded-lg px-4 py-3 overflow-y-auto max-h-52 text-ink whitespace-pre-wrap leading-relaxed">
          {pretty}
        </pre>
      )}

    </div>
  )
}
