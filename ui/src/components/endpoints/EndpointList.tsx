import { useState } from 'react'
import type { Endpoint } from '../../types'
import { CopyButton } from '../ui/CopyButton'
import { EmptyState } from '../ui/EmptyState'
import { TrashIcon, SettingsIcon } from '../ui/icons'
import { usePostHog } from '@posthog/react'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

interface Props {
  endpoints: Endpoint[]
  selectedId: string | null
  onSelect: (ep: Endpoint) => void
  onDelete: (id: string) => void
}

export function EndpointList({ endpoints, selectedId, onSelect, onDelete }: Props) {
  const posthog = usePostHog()
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)

  function handleDelete(id: string) {
    posthog?.capture('named_endpoint_deleted')
    onDelete(id)
    setConfirmDelete(null)
  }

  if (endpoints.length === 0) {
    return (
      <EmptyState
        variant="endpoints"
        title="No named endpoints yet"
        description="Create one above to get a permanent URL you can point real infrastructure at."
      />
    )
  }

  return (
    <div className="p-2 space-y-1">
      {endpoints.map(ep => (
        <div
          key={ep.id}
          onClick={() => onSelect(ep)}
          className={`group rounded-lg cursor-pointer transition-all duration-200 ease-(--ease-considered) border ${
            selectedId === ep.id
              ? 'bg-surface border-border-strong ring-1 ring-indigo-500/20'
              : 'border-transparent hover:bg-surface/70 hover:border-border'
          }`}
        >
          <div className="p-3">
            <div className="flex items-start justify-between gap-2">

              {/* Slug is the dominant, real-infrastructure element */}
              <div className="min-w-0 flex-1">
                <code className="text-sm font-mono font-semibold text-indigo-400 truncate block leading-tight">
                  /i/{ep.slug}
                </code>
                <p className="text-xs text-muted mt-1 truncate">{ep.name}</p>
                {ep.description && (
                  <p className="text-[11px] text-faint mt-0.5 truncate">{ep.description}</p>
                )}
              </div>

              <div
                className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity duration-200 ease-(--ease-considered) shrink-0"
                onClick={e => e.stopPropagation()}
              >
                <CopyButton text={`${BASE_URL}/i/${ep.slug}`} iconOnly />
                <button
                  onClick={() => onSelect(ep)}
                  title="Configure secrets & verification"
                  className="p-1.5 rounded-md text-faint hover:text-indigo-400 hover:bg-indigo-500/10 transition-colors duration-200 ease-(--ease-considered)"
                >
                  <SettingsIcon className="w-3.5 h-3.5" />
                </button>
                {confirmDelete === ep.id ? (
                  <div className="flex gap-1">
                    <button
                      onClick={() => handleDelete(ep.id)}  
                      className="text-[11px] px-2 py-1 rounded bg-red-500 hover:bg-red-400 text-white transition-colors duration-200 ease-(--ease-considered)"
                    >
                      Delete
                    </button>
                    <button
                      onClick={() => setConfirmDelete(null)}
                      className="text-[11px] px-2 py-1 rounded bg-surface-hover text-muted hover:text-ink transition-colors duration-200 ease-(--ease-considered)"
                    >
                      Cancel
                    </button>
                  </div>
                ) : (
                  <button
                    onClick={() => setConfirmDelete(ep.id)}
                    className="p-1.5 rounded-md text-faint hover:text-red-400 hover:bg-red-500/10 transition-colors duration-200 ease-(--ease-considered)"
                  >
                    <TrashIcon className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>

            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
