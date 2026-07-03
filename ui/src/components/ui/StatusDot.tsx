import type { ConnectionStatus } from '../../types'

const config: Record<ConnectionStatus, { colour: string; pulse: boolean; label: string; textClass: string }> = {
  live:         { colour: 'bg-emerald-400', pulse: true,  label: 'Live',         textClass: 'text-emerald-400' },
  connecting:   { colour: 'bg-amber-400',   pulse: true,  label: 'Connecting',   textClass: 'text-amber-400'   },
  disconnected: { colour: 'bg-red-500',     pulse: false, label: 'Disconnected', textClass: 'text-red-400'     },
}

export function StatusDot({ status }: { status: ConnectionStatus }) {
  const { colour, pulse, label, textClass } = config[status]
  return (
    <span className={`flex items-center gap-1.5 text-[11px] font-medium ${textClass}`}>
      <span className="relative flex h-2 w-2 shrink-0">
        {pulse && (
          <span
            className={`animate-ping absolute inline-flex h-full w-full rounded-full ${colour} opacity-75`}
          />
        )}
        <span className={`relative inline-flex rounded-full h-2 w-2 ${colour}`} />
      </span>
      {label}
    </span>
  )
}
