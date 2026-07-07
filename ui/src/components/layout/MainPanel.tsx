import type { CapturedRequest, Endpoint } from '../../types'
import { RequestDetail } from '../detail/RequestDetail'
import { EndpointDetail } from '../endpoints/EndpointDetail'
import { EmptyState } from '../ui/EmptyState'

interface Props {
  selected: CapturedRequest | null
  activeEndpoint?: Endpoint | null
}

export function MainPanel({ selected, activeEndpoint = null }: Props) {
  if (!selected) {
    // No request selected yet, but a named endpoint is active — let the user
    // configure verification secrets before any webhook has ever arrived.
    if (activeEndpoint) {
      return (
        <main className="flex-1 min-w-0 h-screen overflow-y-auto bg-base-alt animate-fade-in">
          <EndpointDetail endpoint={activeEndpoint} />
        </main>
      )
    }

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
