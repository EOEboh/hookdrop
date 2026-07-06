const colours: Record<string, string> = {
  GET:    'bg-sky-500/10 text-sky-400 ring-sky-500/20',
  POST:   'bg-indigo-500/10 text-indigo-400 ring-indigo-500/20',
  PUT:    'bg-amber-500/10 text-amber-400 ring-amber-500/20',
  PATCH:  'bg-orange-500/10 text-orange-400 ring-orange-500/20',
  DELETE: 'bg-rose-500/10 text-rose-400 ring-rose-500/20',
}

const sizeClasses = {
  sm: 'px-1.5 py-0.5 text-[10px]',
  md: 'px-2 py-0.5 text-xs',
  lg: 'px-3 py-1.5 text-sm',
}

interface Props {
  method: string
  size?: 'sm' | 'md' | 'lg'
}

export function MethodBadge({ method, size = 'md' }: Props) {
  const cls = colours[method.toUpperCase()] ?? 'bg-faint/10 text-faint ring-faint/20'
  return (
    <span
      className={`inline-flex items-center rounded font-mono font-semibold ring-1 tracking-wide ${cls} ${sizeClasses[size]}`}
    >
      {method.toUpperCase()}
    </span>
  )
}
