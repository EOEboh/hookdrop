import type { CapturedRequest } from '../../types'
import { MetaBar } from './MetaBar'
import { HeadersTable } from './HeadersTable'
import { BodyViewer } from './BodyViewer'
import { ReplayPanel } from '../replay/ReplayPanel'
import { SecretManager } from '../endpoints/SecretManager'

interface SectionProps {
  title: string
  children: React.ReactNode
}

function Section({ title, children }: SectionProps) {
  return (
    <div>
      <div className="px-6 py-2.5 bg-surface/30 border-y border-border/60 flex items-center">
        <span className="text-[11px] font-semibold text-muted uppercase tracking-widest">
          {title}
        </span>
      </div>
      {children}
    </div>
  )
}

export function RequestDetail({ request }: { request: CapturedRequest }) {
  return (
    <div className="flex flex-col h-full overflow-y-auto">
      <MetaBar request={request} />
      <Section title="Headers">
        <HeadersTable headers={request.headers} />
      </Section>
      <Section title="Body">
        <BodyViewer body={request.body} />
      </Section>
      <Section title="Replay">
        <ReplayPanel request={request} />
      </Section>
      {request.session_id && (
        <Section title="Verification">
          <div className="px-6 py-4">
            <SecretManager endpointId={request.session_id} />
          </div>
        </Section>
      )}
    </div>
  )
}
