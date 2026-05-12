import { useState } from 'react'
import type { Endpoint } from '../../types'
import { CopyButton } from '../ui/CopyButton'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

interface Props {
  endpoints: Endpoint[]
  selectedId: string | null
  onSelect: (ep: Endpoint) => void
  onDelete: (id: string) => void
}

export function EndpointList({ endpoints, selectedId, onSelect, onDelete }: Props) {
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)

  if (endpoints.length === 0) {
    return (
      <div className="px-4 py-6 text-center">
        <p className="text-zinc-600 text-xs">No named endpoints yet</p>
      </div>
    )
  }

  return (
    <div className="space-y-1 p-2">
      {endpoints.map(ep => (
        <div
          key={ep.id}
          onClick={() => onSelect(ep)}
          className={`group rounded-lg px-3 py-2.5 cursor-pointer transition-colors ${
            selectedId === ep.id
              ? 'bg-zinc-800 border border-zinc-700'
              : 'hover:bg-zinc-800/50 border border-transparent'
          }`}
        >
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0 flex-1">
              <p className="text-sm text-zinc-200 font-medium truncate">{ep.name}</p>
              <p className="text-xs font-mono text-emerald-500 truncate mt-0.5">/i/{ep.slug}</p>
              {ep.description && (
                <p className="text-xs text-zinc-600 truncate mt-0.5">{ep.description}</p>
              )}
            </div>
            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
              <CopyButton text={`${BASE_URL}/i/${ep.slug}`} label="Copy" />
              {confirmDelete === ep.id ? (
                <div className="flex gap-1">
                  <button
                    onClick={e => { e.stopPropagation(); onDelete(ep.id); setConfirmDelete(null) }}
                    className="text-xs px-2 py-1 rounded bg-red-600 hover:bg-red-500 text-white transition-colors"
                  >
                    Confirm
                  </button>
                  <button
                    onClick={e => { e.stopPropagation(); setConfirmDelete(null) }}
                    className="text-xs px-2 py-1 rounded bg-zinc-700 text-zinc-300 transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              ) : (
                <button
                  onClick={e => { e.stopPropagation(); setConfirmDelete(ep.id) }}
                  className="text-xs px-2 py-1 rounded bg-zinc-800 hover:bg-red-500/20 hover:text-red-400 text-zinc-500 transition-colors"
                >
                  Delete
                </button>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}