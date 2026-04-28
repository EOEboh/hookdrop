const colours: Record<string, string> = {
  GET:    'bg-blue-500/10 text-blue-400 ring-blue-500/20',
  POST:   'bg-emerald-500/10 text-emerald-400 ring-emerald-500/20',
  PUT:    'bg-amber-500/10 text-amber-400 ring-amber-500/20',
  PATCH:  'bg-orange-500/10 text-orange-400 ring-orange-500/20',
  DELETE: 'bg-red-500/10 text-red-400 ring-red-500/20',
}

export function MethodBadge({ method }: { method: string }) {
  const cls = colours[method.toUpperCase()] ?? 'bg-zinc-500/10 text-zinc-400 ring-zinc-500/20'
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-mono font-medium ring-1 ${cls}`}>
      {method.toUpperCase()}
    </span>
  )
}