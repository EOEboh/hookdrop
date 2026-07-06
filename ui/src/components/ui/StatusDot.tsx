import type { ConnectionStatus } from '../../types'

const config: Record<ConnectionStatus, { colour: string; pulse: boolean; label: string; textClass: string }> = {
  live:         { colour: 'bg-indigo-400', pulse: true,  label: 'Live',         textClass: 'text-indigo-400' },
  connecting:   { colour: 'bg-amber-400',  pulse: true,  label: 'Connecting',   textClass: 'text-amber-400'  },
  disconnected: { colour: 'bg-red-500',    pulse: false, label: 'Disconnected', textClass: 'text-red-400'    },
}

export function StatusDot({ status }: { status: ConnectionStatus }) {
  const { colour, pulse, label, textClass } = config[status]
  return (
    <span className={`flex items-center gap-1.5 text-[11px] font-medium ${textClass}`}>
      <span className="relative flex h-2.5 w-2.5 shrink-0 items-center justify-center">
        {pulse && (
          <>
            <span
              className={`animate-ring-pulse absolute inline-flex h-2 w-2 rounded-full ${colour}`}
            />
            <span
              className={`animate-ring-pulse absolute inline-flex h-2 w-2 rounded-full ${colour}`}
              style={{ animationDelay: '1.1s' }}
            />
          </>
        )}
        <span className={`relative inline-flex rounded-full h-1.5 w-1.5 ${colour}`} />
      </span>
      {label}
    </span>
  )
}
