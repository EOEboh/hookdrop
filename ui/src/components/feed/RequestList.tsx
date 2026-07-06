import type { CapturedRequest, RequestFilters } from '../../types'
import { RequestItem } from './RequestItem'
import { EmptyFeed } from './EmptyFeed'
import { FilterBar } from './FilterBar'

interface Props {
  requests: CapturedRequest[]
  newIds: Set<string>
  allCount: number
  selectedId: string | null
  onSelect: (req: CapturedRequest) => void
  filters: RequestFilters
  onFilterChange: (f: RequestFilters) => void
}

export function RequestList({
  requests, newIds, allCount, selectedId, onSelect, filters, onFilterChange,
}: Props) {
  const isFiltered = Object.values(filters).some(Boolean)

  return (
    <div data-tour="request-feed" className="flex flex-col flex-1 min-h-0">
      <FilterBar
        filters={filters}
        onChange={onFilterChange}
        resultCount={requests.length}
        totalCount={allCount}
      />

      <div className="flex-1 overflow-y-auto">
        {requests.length === 0 ? (
          isFiltered ? (
            <div className="flex flex-col items-center justify-center py-12 px-6 text-center">
              <p className="text-ink text-sm font-medium mb-1">No matching requests</p>
              <p className="text-faint text-xs">Try adjusting your filters</p>
            </div>
          ) : (
            <EmptyFeed />
          )
        ) : (
          requests.map(req => (
            <RequestItem
              key={req.id}
              request={req}
              selected={req.id === selectedId}
              isNew={newIds.has(req.id)}
              onClick={() => onSelect(req)}
            />
          ))
        )}
      </div>
    </div>
  )
}
