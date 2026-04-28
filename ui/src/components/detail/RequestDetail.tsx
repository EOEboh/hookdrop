import type { CapturedRequest } from '../../types'
import { MetaBar } from './MetaBar'
import { HeadersTable } from './HeadersTable'
import { BodyViewer } from './BodyViewer'
import { ReplayPanel } from '../replay/ReplayPanel'

interface Props {
  request: CapturedRequest
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="px-6 py-2 bg-zinc-900/40 border-b border-zinc-800">
        <span className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">
          {title}
        </span>
      </div>
      {children}
    </div>
  )
}

export function RequestDetail({ request }: Props) {
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
    </div>
  )
}