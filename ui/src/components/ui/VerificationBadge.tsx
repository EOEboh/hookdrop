import type { VerificationStatus } from '../../types'

const config = {
  verified: {
    label: 'Verified',
    className: 'bg-emerald-500/10 text-emerald-400 ring-emerald-500/20',
    dotClass: 'bg-emerald-400',
  },
  failed: {
    label: 'Failed',
    className: 'bg-red-500/10 text-red-400 ring-red-500/20',
    dotClass: 'bg-red-400',
  },
  unverified: {
    label: 'Unverified',
    className: 'bg-zinc-500/10 text-zinc-500 ring-zinc-500/20',
    dotClass: 'bg-zinc-600',
  },
}

interface Props {
  status: VerificationStatus
  provider?: string
  showProvider?: boolean
}

export function VerificationBadge({ status, provider, showProvider = false }: Props) {
  const { label, className, dotClass } = config[status] ?? config.unverified
  const displayLabel =
    showProvider && provider && status !== 'unverified'
      ? `${provider} — ${label}`
      : label

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-medium ring-1 ${className}`}
    >
      <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${dotClass}`} />
      {displayLabel}
    </span>
  )
}
