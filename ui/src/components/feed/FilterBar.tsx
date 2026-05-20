import { useRef, useState } from 'react'
import type { RequestFilters } from '../../types'

const METHODS  = ['POST', 'GET', 'PUT', 'PATCH', 'DELETE']
const VERIFIED = [
  { value: 'verified',   label: '✓ Verified' },
  { value: 'failed',     label: '✗ Failed' },
  { value: 'unverified', label: '– No secret' },
]
const RANGES = [
  { value: '1h',  label: 'Last hour' },
  { value: '24h', label: 'Last 24h' },
  { value: '7d',  label: 'Last 7 days' },
]

interface Props {
  filters: RequestFilters
  onChange: (filters: RequestFilters) => void
  resultCount: number
  totalCount: number
}

export function FilterBar({ filters, onChange, resultCount, totalCount }: Props) {
  const [expanded, setExpanded] = useState(false)
  const searchRef = useRef<HTMLInputElement>(null)

  const isFiltered = filters.search || filters.method || filters.verified || filters.range
  const hasResults  = resultCount === totalCount

  function set(key: keyof RequestFilters, value: string) {
    onChange({ ...filters, [key]: value === filters[key] ? '' : value })
  }

  function clearAll() {
    onChange({ search: '', method: '', verified: '', range: '' })
    searchRef.current?.focus()
  }

  return (
    <div className="border-b border-zinc-800 bg-zinc-950">
      {/* Search row */}
      <div className="flex items-center gap-2 px-3 py-2">
        <div className="flex-1 flex items-center gap-2 bg-zinc-900 rounded-lg px-3 py-1.5">
          <svg className="w-3 h-3 text-zinc-500 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
          </svg>
          <input
            ref={searchRef}
            type="text"
            value={filters.search}
            onChange={e => onChange({ ...filters, search: e.target.value })}
            placeholder="Search body..."
            className="flex-1 bg-transparent text-xs text-zinc-300 placeholder-zinc-600 focus:outline-none"
          />
          {filters.search && (
            <button onClick={() => onChange({ ...filters, search: '' })} className="text-zinc-600 hover:text-zinc-400">
              ×
            </button>
          )}
        </div>

        {/* Filter toggle */}
        <button
          onClick={() => setExpanded(e => !e)}
          className={`shrink-0 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-colors ${
            expanded || (isFiltered && !filters.search)
              ? 'bg-emerald-600/20 text-emerald-400'
              : 'bg-zinc-800 text-zinc-500 hover:text-zinc-300'
          }`}
        >
          Filters
          {(filters.method || filters.verified || filters.range) && (
            <span className="ml-1.5 bg-emerald-600 text-white rounded-full px-1.5 py-0.5 text-[10px]">
              {[filters.method, filters.verified, filters.range].filter(Boolean).length}
            </span>
          )}
        </button>
      </div>

      {/* Expanded filter panels */}
      {expanded && (
        <div className="px-3 pb-3 space-y-3">

          {/* Method */}
          <div className="space-y-1.5">
            <span className="text-[10px] font-semibold text-zinc-600 uppercase tracking-wider">Method</span>
            <div className="flex flex-wrap gap-1.5">
              {METHODS.map(m => (
                <button
                  key={m}
                  onClick={() => set('method', m)}
                  className={`px-2.5 py-1 rounded text-xs font-mono font-medium transition-colors ${
                    filters.method === m
                      ? 'bg-emerald-600 text-white'
                      : 'bg-zinc-800 text-zinc-400 hover:text-zinc-200'
                  }`}
                >
                  {m}
                </button>
              ))}
            </div>
          </div>

          {/* Verification status */}
          <div className="space-y-1.5">
            <span className="text-[10px] font-semibold text-zinc-600 uppercase tracking-wider">Verification</span>
            <div className="flex flex-wrap gap-1.5">
              {VERIFIED.map(v => (
                <button
                  key={v.value}
                  onClick={() => set('verified', v.value)}
                  className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                    filters.verified === v.value
                      ? 'bg-emerald-600 text-white'
                      : 'bg-zinc-800 text-zinc-400 hover:text-zinc-200'
                  }`}
                >
                  {v.label}
                </button>
              ))}
            </div>
          </div>

          {/* Date range */}
          <div className="space-y-1.5">
            <span className="text-[10px] font-semibold text-zinc-600 uppercase tracking-wider">Time range</span>
            <div className="flex flex-wrap gap-1.5">
              {RANGES.map(r => (
                <button
                  key={r.value}
                  onClick={() => set('range', r.value)}
                  className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                    filters.range === r.value
                      ? 'bg-emerald-600 text-white'
                      : 'bg-zinc-800 text-zinc-400 hover:text-zinc-200'
                  }`}
                >
                  {r.label}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Results count + clear */}
      {isFiltered && (
        <div className="px-3 py-1.5 border-t border-zinc-800/60 flex items-center justify-between">
          <span className="text-[11px] text-zinc-500">
            {hasResults
              ? `${resultCount} request${resultCount !== 1 ? 's' : ''}`
              : `${resultCount} of ${totalCount} request${totalCount !== 1 ? 's' : ''}`
            }
          </span>
          <button
            onClick={clearAll}
            className="text-[11px] text-zinc-600 hover:text-zinc-400 transition-colors"
          >
            Clear filters
          </button>
        </div>
      )}
    </div>
  )
}