import type { CapturedRequest } from '../../types'
import { RequestDetail } from '../detail/RequestDetail'
import { EmptyState } from '../ui/EmptyState'

export function MainPanel({ selected }: { selected: CapturedRequest | null }) {
  if (!selected) {
    return (
      <div className="flex-1 flex items-center justify-center bg-base-alt">
        <EmptyState
          variant="detail"
          size="lg"
          title="Nothing selected yet"
          description="Pick a request from the list to inspect its headers, body, and replay it against any URL."
        />
      </div>
    )
  }

  return (
    // key forces remount + fade-in animation on request change
    <main
      key={selected.id}
      className="flex-1 min-w-0 h-screen overflow-y-auto bg-base-alt animate-fade-in"
    >
      <RequestDetail request={selected} />
    </main>
  )
}
