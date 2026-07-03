import { useState } from 'react'
import type { Endpoint } from '../../types'
import { CopyButton } from '../ui/CopyButton'
import { TrashIcon, TerminalIcon } from '../ui/icons'
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
      <div className="flex flex-col items-center justify-center py-12 px-6 text-center space-y-3">
        <div className="w-10 h-10 rounded-lg bg-zinc-900 border border-zinc-800 flex items-center justify-center">
          <TerminalIcon className="w-4 h-4 text-zinc-600" />
        </div>
        <div>
          <p className="text-xs font-medium text-zinc-400 mb-0.5">No named endpoints</p>
          <p className="text-xs text-zinc-600">Create one above to get a permanent URL</p>
        </div>
      </div>
    )
  }

  return (
    <div className="p-2 space-y-1">
      {endpoints.map(ep => (
        <div
          key={ep.id}
          onClick={() => onSelect(ep)}
          className={`group rounded-lg cursor-pointer transition-all border ${
            selectedId === ep.id
              ? 'bg-zinc-900 border-zinc-700 ring-1 ring-emerald-500/15'
              : 'border-transparent hover:bg-zinc-900/60 hover:border-zinc-800'
          }`}
        >
          <div className="p-3">
            <div className="flex items-start justify-between gap-2">

              <div className="min-w-0 flex-1">
                <code className="text-xs font-mono font-medium text-emerald-400 truncate block">
                  /i/{ep.slug}
                </code>
                <p className="text-xs text-zinc-300 font-medium mt-0.5 truncate">{ep.name}</p>
                {ep.description && (
                  <p className="text-[11px] text-zinc-600 mt-0.5 truncate">{ep.description}</p>
                )}
              </div>

              <div
                className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                onClick={e => e.stopPropagation()}
              >
                <CopyButton text={`${BASE_URL}/i/${ep.slug}`} iconOnly />
                {confirmDelete === ep.id ? (
                  <div className="flex gap-1">
                    <button
                      onClick={() => handleDelete(ep.id)}  
                      className="text-[11px] px-2 py-1 rounded bg-red-600 hover:bg-red-500 text-white transition-colors"
                    >
                      Delete
                    </button>
                    <button
                      onClick={() => setConfirmDelete(null)}
                      className="text-[11px] px-2 py-1 rounded bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition-colors"
                    >
                      Cancel
                    </button>
                  </div>
                ) : (
                  <button
                    onClick={() => setConfirmDelete(ep.id)}
                    className="p-1.5 rounded-md text-zinc-600 hover:text-red-400 hover:bg-red-500/10 transition-colors"
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