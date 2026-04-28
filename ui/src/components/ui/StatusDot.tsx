import type { ConnectionStatus } from '../../types'

const config = {
  live:         { colour: 'bg-emerald-400', pulse: true,  label: 'Live' },
  connecting:   { colour: 'bg-amber-400',   pulse: false, label: 'Connecting' },
  disconnected: { colour: 'bg-red-500',     pulse: false, label: 'Disconnected' },
}

export function StatusDot({ status }: { status: ConnectionStatus }) {
  const { colour, pulse, label } = config[status]
  return (
    <span className="flex items-center gap-1.5 text-xs text-zinc-400">
      <span className="relative flex h-2 w-2">
        {pulse && (
          <span className={`animate-ping absolute inline-flex h-full w-full rounded-full ${colour} opacity-75`} />
        )}
        <span className={`relative inline-flex rounded-full h-2 w-2 ${colour}`} />
      </span>
      {label}
    </span>
  )
}