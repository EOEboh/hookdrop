import type { VerificationStatus } from '../../types'

const config = {
  verified: {
    label: 'Verified',
    className: 'bg-emerald-500/10 text-emerald-400 ring-emerald-500/20',
    icon: '✓',
  },
  failed: {
    label: 'Verification failed',
    className: 'bg-red-500/10 text-red-400 ring-red-500/20',
    icon: '✗',
  },
  unverified: {
    label: 'No secret',
    className: 'bg-zinc-500/10 text-zinc-500 ring-zinc-500/20',
    icon: '–',
  },
}

interface Props {
  status: VerificationStatus
  provider?: string
  showProvider?: boolean
}

export function VerificationBadge({ status, provider, showProvider = false }: Props) {
  const { label, className, icon } = config[status] ?? config.unverified

  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ring-1 ${className}`}>
      <span>{icon}</span>
      {showProvider && provider ? `${provider} — ${label}` : label}
    </span>
  )
}