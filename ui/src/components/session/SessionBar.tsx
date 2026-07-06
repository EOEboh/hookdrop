import type { Session, ConnectionStatus } from '../../types'
import { CopyButton } from '../ui/CopyButton'
import { StatusDot } from '../ui/StatusDot'
import { OnboardingHint, markOnboardingSeen } from './OnboardingHint'
import { ClockIcon, RefreshCwIcon } from '../ui/icons'
import { ttlLabel } from '../../lib/format'
import { usePostHog } from '@posthog/react'

interface Props {
  session: Session
  status: ConnectionStatus
  onReset: () => void
}

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export function SessionBar({ session, status, onReset }: Props) {
  const inboxUrl = `${BASE_URL}/i/${session.id}`
  const posthog = usePostHog()

  return (
    <div className="px-4 py-3 border-b border-border space-y-3">

      {/* Header row */}
      <div className="flex items-center justify-between">
        <span className="text-[11px] font-semibold text-muted uppercase tracking-widest">
          Endpoint
        </span>
        <StatusDot status={status} />
      </div>

      {/* URL block — primary affordance */}
      <div className="bg-surface border border-border rounded-lg px-3 py-2.5 flex items-center gap-2 hover:border-border-strong transition-colors duration-200 ease-(--ease-considered)">
        <code className="flex-1 font-mono text-xs text-indigo-400 truncate select-all leading-relaxed">
          {inboxUrl}
        </code>
        <CopyButton
          text={inboxUrl}
          iconOnly
          onCopy={() => { posthog?.capture('inbox_url_copied'); markOnboardingSeen() }}
        />
      </div>

      {/* First-time nudge — points at the copy button, dismissed after first copy */}
      <OnboardingHint />

      {/* TTL + reset */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5 text-[11px] text-faint">
          <ClockIcon className="w-3 h-3 shrink-0" />
          <span className="tabular-nums">{ttlLabel(session.expires_at)}</span>
        </div>
        <button
          onClick={onReset}
          className="flex items-center gap-1 text-[11px] text-faint hover:text-muted transition-colors duration-200 ease-(--ease-considered)"
        >
          <RefreshCwIcon className="w-3 h-3" />
          New session
        </button>
      </div>

    </div>
  )
}
