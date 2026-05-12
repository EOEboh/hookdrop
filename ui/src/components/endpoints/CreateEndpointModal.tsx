import { useState } from 'react'
import type { Endpoint } from '../../types'

interface Props {
  onClose: () => void
  onCreate: (data: { slug: string; name: string; description?: string }) => Promise<Endpoint>
}

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export function CreateEndpointModal({ onClose, onCreate }: Props) {
  const [slug, setSlug]               = useState('')
  const [name, setName]               = useState('')
  const [description, setDescription] = useState('')
  const [loading, setLoading]         = useState(false)
  const [error, setError]             = useState<string | null>(null)
  const [slugError, setSlugError]     = useState<string | null>(null)

  function validateSlugLocally(val: string): string | null {
    if (val.length < 3) return 'At least 3 characters'
    if (val.length > 48) return 'Max 48 characters'
    if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(val)) return 'Lowercase letters, numbers, hyphens only'
    return null
  }

  function handleSlugChange(val: string) {
    const normalised = val.toLowerCase().replace(/\s+/g, '-')
    setSlug(normalised)
    setSlugError(validateSlugLocally(normalised))
  }

  async function handleSubmit() {
    const localError = validateSlugLocally(slug)
    if (localError) { setSlugError(localError); return }
    if (!name.trim()) { setError('Name is required'); return }

    setLoading(true)
    setError(null)
    try {
      await onCreate({ slug, name: name.trim(), description: description.trim() || undefined })
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create endpoint')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4"
      onClick={e => e.target === e.currentTarget && onClose()}
    >
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl w-full max-w-md p-6 space-y-5">
        <div className="flex items-center justify-between">
          <h2 className="text-zinc-100 font-semibold">New named endpoint</h2>
          <button onClick={onClose} className="text-zinc-500 hover:text-zinc-300 text-xl leading-none">×</button>
        </div>


        <div className="space-y-1.5">
          <label className="text-xs text-zinc-500">Name</label>
          <input
            type="text"
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="Stripe production"
            className="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-emerald-500 transition-colors"
          />
        </div>


        <div className="space-y-1.5">
          <label className="text-xs text-zinc-500">Slug</label>
          <div className="flex items-center bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-2 focus-within:border-emerald-500 transition-colors">
            <span className="text-zinc-600 text-sm font-mono">/i/</span>
            <input
              type="text"
              value={slug}
              onChange={e => handleSlugChange(e.target.value)}
              placeholder="stripe-production"
              className="flex-1 bg-transparent text-sm font-mono text-zinc-200 placeholder-zinc-600 focus:outline-none"
            />
          </div>
          {slugError
            ? <p className="text-xs text-red-400">{slugError}</p>
            : slug.length >= 3 && (
              <p className="text-xs text-zinc-600 font-mono">
                {BASE_URL}/i/{slug}
              </p>
            )
          }
        </div>


        <div className="space-y-1.5">
          <label className="text-xs text-zinc-500">Description <span className="text-zinc-700">(optional)</span></label>
          <input
            type="text"
            value={description}
            onChange={e => setDescription(e.target.value)}
            placeholder="Live Stripe payment events"
            className="w-full bg-zinc-950 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-emerald-500 transition-colors"
          />
        </div>

        {error && (
          <p className="text-xs text-red-400 bg-red-500/10 px-3 py-2 rounded-lg">{error}</p>
        )}

        <div className="flex gap-3 pt-1">
          <button
            onClick={onClose}
            className="flex-1 px-4 py-2 rounded-lg border border-zinc-700 text-sm text-zinc-400 hover:text-zinc-200 hover:border-zinc-600 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading || !!slugError || !slug || !name}
            className="flex-1 px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm font-medium transition-colors"
          >
            {loading ? 'Creating…' : 'Create endpoint'}
          </button>
        </div>
      </div>
    </div>
  )
}