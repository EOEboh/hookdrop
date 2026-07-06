import type { ReactNode } from 'react'

type EmptyStateVariant = 'feed' | 'detail' | 'endpoints'

interface Props {
  variant: EmptyStateVariant
  title: string
  description: string
  size?: 'sm' | 'lg'
  children?: ReactNode
}

// Abstract, geometric illustrations built from simple shapes tinted with the
// indigo/violet accent — no stock art, no emoji, matches the new palette.

function FeedIllustration() {
  return (
    <svg viewBox="0 0 64 64" fill="none" className="w-full h-full">
      <circle cx="32" cy="32" r="31" className="fill-indigo-500/[0.05]" />
      <rect x="13" y="19" width="30" height="7" rx="3.5" className="fill-indigo-400/15" />
      <rect x="13" y="30.5" width="38" height="7" rx="3.5" className="fill-indigo-500/25" />
      <rect x="13" y="42" width="22" height="7" rx="3.5" className="fill-indigo-400/10" />
      <circle cx="48" cy="20" r="2.5" className="fill-indigo-300/60" />
      <circle cx="53" cy="20" r="1.5" className="fill-indigo-300/30" />
    </svg>
  )
}

function DetailIllustration() {
  return (
    <svg viewBox="0 0 64 64" fill="none" className="w-full h-full">
      <rect x="10" y="12" width="34" height="40" rx="5" className="fill-indigo-500/[0.06] stroke-indigo-400/20" strokeWidth="1" />
      <rect x="16" y="20" width="20" height="4" rx="2" className="fill-indigo-400/25" />
      <rect x="16" y="28" width="14" height="4" rx="2" className="fill-indigo-400/15" />
      <rect x="16" y="36" width="22" height="4" rx="2" className="fill-indigo-400/15" />
      <circle cx="46" cy="46" r="10" className="fill-indigo-500/15 stroke-indigo-400/40" strokeWidth="1.5" />
      <path d="M53 53 L58 58" className="stroke-indigo-400/40" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
}

function EndpointsIllustration() {
  return (
    <svg viewBox="0 0 64 64" fill="none" className="w-full h-full">
      <circle cx="18" cy="18" r="5" className="fill-indigo-400/25" />
      <circle cx="46" cy="16" r="4" className="fill-indigo-400/15" />
      <circle cx="48" cy="46" r="6" className="fill-indigo-500/25" />
      <circle cx="16" cy="46" r="4" className="fill-indigo-400/15" />
      <path d="M18 18 L46 16 M18 18 L48 46 M18 18 L16 46 M46 16 L48 46" className="stroke-indigo-400/20" strokeWidth="1.5" />
    </svg>
  )
}

const ILLUSTRATIONS: Record<EmptyStateVariant, () => ReactNode> = {
  feed:      FeedIllustration,
  detail:    DetailIllustration,
  endpoints: EndpointsIllustration,
}

export function EmptyState({ variant, title, description, size = 'sm', children }: Props) {
  const Illustration = ILLUSTRATIONS[variant]
  const boxSize = size === 'lg' ? 'w-20 h-20' : 'w-14 h-14'

  return (
    <div className="flex flex-col items-center justify-center text-center px-6 py-10 space-y-4">
      <div className={`${boxSize} rounded-2xl bg-surface border border-border flex items-center justify-center p-3`}>
        <Illustration />
      </div>

      <div className="space-y-1.5 max-w-xs">
        <p className="text-sm font-medium text-ink">{title}</p>
        <p className="text-xs text-muted leading-relaxed">{description}</p>
      </div>

      {children}
    </div>
  )
}
