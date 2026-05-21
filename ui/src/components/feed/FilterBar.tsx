import { useRef, useState } from 'react'
import type { RequestFilters } from '../../types'
import { SearchIcon, SlidersIcon, XIcon } from '../ui/icons'

const METHODS  = ['POST', 'GET', 'PUT', 'PATCH', 'DELETE']
const VERIFIED = [
  { value: 'verified',   label: 'Verified'   },
  { value: 'failed',     label: 'Failed'     },
  { value: 'unverified', label: 'No secret'  },
]
const RANGES = [
  { value: '1h',  label: 'Last 1h'   },
  { value: '24h', label: 'Last 24h'  },
  { value: '7d',  label: 'Last 7d'   },
]

const METHOD_CHIP_COLOURS: Record<string, string> = {
  GET:    'bg-blue-500/10 text-blue-400 border-blue-500/30',
  POST:   'bg-emerald-500/10 text-emerald-400 border-emerald-500/30',
  PUT:    'bg-amber-500/10 text-amber-400 border-amber-500/30',
  PATCH:  'bg-orange-500/10 text-orange-400 border-orange-500/30',
  DELETE: 'bg-red-500/10 text-red-400 border-red-500/30',
}

const METHOD_BTN_ACTIVE: Record<string, string> = {
  GET:    'bg-blue-500/10 text-blue-400 border-blue-500/30',
  POST:   'bg-emerald-500/10 text-emerald-400 border-emerald-500/30',
  PUT:    'bg-amber-500/10 text-amber-400 border-amber-500/30',
  PATCH:  'bg-orange-500/10 text-orange-400 border-orange-500/30',
  DELETE: 'bg-red-500/10 text-red-400 border-red-500/30',
}

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
  const hasActiveChips = filters.method || filters.verified || filters.range
  const activeChipCount = [filters.method, filters.verified, filters.range].filter(Boolean).length

  function set(key: keyof RequestFilters, value: string) {
    onChange({ ...filters, [key]: value === filters[key] ? '' : value })
  }

  function clearAll() {
    onChange({ search: '', method: '', verified: '', range: '' })
    searchRef.current?.focus()
  }

  const verifiedLabel = VERIFIED.find(v => v.value === filters.verified)?.label ?? ''
  const rangeLabel    = RANGES.find(r => r.value === filters.range)?.label ?? ''

  return (
    <div className="border-b border-zinc-800 bg-zinc-950">

      {/* Search row */}
      <div className="flex items-center gap-2 px-3 py-2">
        <div className="flex-1 flex items-center gap-2 bg-zinc-900 rounded-lg px-3 py-1.5 border border-transparent focus-within:border-zinc-700 transition-colors">
          <SearchIcon className="w-3.5 h-3.5 text-zinc-600 shrink-0" />
          <input
            ref={searchRef}
            type="text"
            value={filters.search}
            onChange={e => onChange({ ...filters, search: e.target.value })}
            placeholder="Search body..."
            className="flex-1 bg-transparent text-xs text-zinc-300 placeholder-zinc-600 focus:outline-none"
          />
          {filters.search && (
            <button
              onClick={() => onChange({ ...filters, search: '' })}
              className="text-zinc-600 hover:text-zinc-400 transition-colors"
            >
              <XIcon className="w-3 h-3" />
            </button>
          )}
        </div>

        {/* Filter toggle */}
        <button
          onClick={() => setExpanded(e => !e)}
          className={`shrink-0 flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-colors border ${
            expanded || hasActiveChips
              ? 'bg-emerald-600/15 text-emerald-400 border-emerald-500/20'
              : 'bg-zinc-900 border-zinc-800 text-zinc-500 hover:text-zinc-300 hover:border-zinc-700'
          }`}
        >
          <SlidersIcon className="w-3.5 h-3.5" />
          Filters
          {activeChipCount > 0 && (
            <span className="min-w-[18px] h-[18px] flex items-center justify-center bg-emerald-600 text-white rounded-full text-[10px] font-bold px-1">
              {activeChipCount}
            </span>
          )}
        </button>
      </div>

      {/* Active filter chips — always visible when filters applied (even when collapsed) */}
      {hasActiveChips && (
        <div className="flex flex-wrap gap-1.5 px-3 pb-2">
          {filters.method && (
            <button
              onClick={() => set('method', filters.method)}
              className={`inline-flex items-center gap-1 px-2 py-0.5 rounded border text-[11px] font-mono font-medium hover:opacity-70 transition-opacity ${
                METHOD_CHIP_COLOURS[filters.method] ?? 'bg-zinc-800 text-zinc-400 border-zinc-700'
              }`}
            >
              {filters.method}
              <XIcon className="w-2.5 h-2.5" />
            </button>
          )}
          {filters.verified && (
            <button
              onClick={() => set('verified', filters.verified)}
              className="inline-flex items-center gap-1 px-2 py-0.5 rounded border border-zinc-700 bg-zinc-800 text-[11px] text-zinc-400 font-medium hover:opacity-70 transition-opacity"
            >
              {verifiedLabel}
              <XIcon className="w-2.5 h-2.5" />
            </button>
          )}
          {filters.range && (
            <button
              onClick={() => set('range', filters.range)}
              className="inline-flex items-center gap-1 px-2 py-0.5 rounded border border-zinc-700 bg-zinc-800 text-[11px] text-zinc-400 font-medium hover:opacity-70 transition-opacity"
            >
              {rangeLabel}
              <XIcon className="w-2.5 h-2.5" />
            </button>
          )}
        </div>
      )}

      {/* Expanded filter panels */}
      {expanded && (
        <div className="border-t border-zinc-800/60 px-3 pt-3 pb-3 space-y-3">

          {/* Method */}
          <div className="space-y-1.5">
            <span className="text-[10px] font-semibold text-zinc-600 uppercase tracking-widest">
              Method
            </span>
            <div className="flex flex-wrap gap-1.5">
              {METHODS.map(m => (
                <button
                  key={m}
                  onClick={() => set('method', m)}
                  className={`px-2.5 py-1 rounded text-xs font-mono font-medium transition-colors border ${
                    filters.method === m
                      ? `${METHOD_BTN_ACTIVE[m] ?? 'bg-emerald-600/20 text-emerald-400 border-emerald-500/30'}`
                      : 'bg-zinc-900 border-zinc-800 text-zinc-400 hover:text-zinc-200 hover:border-zinc-700'
                  }`}
                >
                  {m}
                </button>
              ))}
            </div>
          </div>

          {/* Verification */}
          <div className="space-y-1.5">
            <span className="text-[10px] font-semibold text-zinc-600 uppercase tracking-widest">
              Verification
            </span>
            <div className="flex flex-wrap gap-1.5">
              {VERIFIED.map(v => (
                <button
                  key={v.value}
                  onClick={() => set('verified', v.value)}
                  className={`px-2.5 py-1 rounded text-xs font-medium transition-colors border ${
                    filters.verified === v.value
                      ? 'bg-emerald-600/20 text-emerald-400 border-emerald-500/30'
                      : 'bg-zinc-900 border-zinc-800 text-zinc-400 hover:text-zinc-200 hover:border-zinc-700'
                  }`}
                >
                  {v.label}
                </button>
              ))}
            </div>
          </div>

          {/* Date range */}
          <div className="space-y-1.5">
            <span className="text-[10px] font-semibold text-zinc-600 uppercase tracking-widest">
              Time range
            </span>
            <div className="flex flex-wrap gap-1.5">
              {RANGES.map(r => (
                <button
                  key={r.value}
                  onClick={() => set('range', r.value)}
                  className={`px-2.5 py-1 rounded text-xs font-medium transition-colors border ${
                    filters.range === r.value
                      ? 'bg-emerald-600/20 text-emerald-400 border-emerald-500/30'
                      : 'bg-zinc-900 border-zinc-800 text-zinc-400 hover:text-zinc-200 hover:border-zinc-700'
                  }`}
                >
                  {r.label}
                </button>
              ))}
            </div>
          </div>

        </div>
      )}

      {/* Results summary */}
      {isFiltered && (
        <div className="px-3 py-1.5 border-t border-zinc-800/60 flex items-center justify-between">
          <span className="text-[11px] text-zinc-600">
            {resultCount === totalCount
              ? `${resultCount} request${resultCount !== 1 ? 's' : ''}`
              : `${resultCount} of ${totalCount}`
            }
          </span>
          <button
            onClick={clearAll}
            className="text-[11px] text-zinc-600 hover:text-zinc-400 transition-colors"
          >
            Clear all
          </button>
        </div>
      )}

    </div>
  )
}
