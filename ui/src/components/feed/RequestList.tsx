import type { CapturedRequest } from '../../types'
import { RequestItem } from './RequestItem'
import { EmptyFeed } from './EmptyFeed'

interface Props {
  requests: CapturedRequest[]
  selectedId: string | null
  onSelect: (req: CapturedRequest) => void
}

export function RequestList({ requests, selectedId, onSelect }: Props) {
  if (requests.length === 0) return <EmptyFeed />

  return (
    <div className="flex-1 overflow-y-auto">
      {requests.map((req) => (
        <RequestItem
          key={req.id}
          request={req}
          selected={req.id === selectedId}
          onClick={() => onSelect(req)}
        />
      ))}
    </div>
  )
}