import { useState } from 'react'
import type { Endpoint } from '../../types'
import { Portal } from '../ui/Portal'
import { UpgradePrompt } from '../billing/UpgradePrompt'
import { usePostHog } from '@posthog/react'

interface Props {
  onClose: () => void
  onCreate: (data: { slug: string; name: string; description?: string }) => Promise<Endpoint>
}

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export function CreateEndpointModal({ onClose, onCreate }: Props) {
  const posthog = usePostHog()

  const [slug, setSlug]               = useState('')
  const [name, setName]               = useState('')
  const [description, setDescription] = useState('')
  const [loading, setLoading]         = useState(false)
  const [error, setError]             = useState<string | null>(null)
  const [slugError, setSlugError]     = useState<string | null>(null)
  const [limitReached, setLimitReached] = useState(false)

  

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
    setLimitReached(false)

    try {
      await onCreate({ slug, name: name.trim(), description: description.trim() || undefined })
      posthog?.capture('named_endpoint_created', {      
        has_description: !!description.trim(),
      })
      onClose()
    } catch (err) {
      const message = err instanceof Error ? err.message : ''

      // Backend returns "402: ..." when the endpoint limit is reached
      if (message.startsWith('402') || message.includes('endpoint_limit_reached')) {
        setLimitReached(true)
      } else {
        setError(message || 'Failed to create endpoint')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <Portal>
      <div
        className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4 animate-fade-in"
        onClick={e => e.target === e.currentTarget && onClose()}
      >
        <div className="bg-surface border border-border rounded-xl w-full max-w-md p-6 space-y-5">

          <div className="flex items-center justify-between">
            <h2 className="text-ink font-semibold">New named endpoint</h2>
            <button
              onClick={onClose}
              className="text-muted hover:text-ink text-xl leading-none transition-colors duration-200 ease-(--ease-considered)"
            >
              ×
            </button>
          </div>

          {/* Upgrade prompt replaces the form when limit is hit */}
          {limitReached ? (
            <div className="space-y-4">
              <UpgradePrompt
                feature="Named endpoint limit reached"
                description="Free accounts include 1 named endpoint. Upgrade to Pro for unlimited named endpoints, 50k requests/month, and 90 day history."
              />
              <button
                onClick={onClose}
                className="w-full px-4 py-2 rounded-lg border border-border-strong text-sm text-muted hover:text-ink hover:border-faint transition-colors duration-200 ease-(--ease-considered)"
              >
                Maybe later
              </button>
            </div>
          ) : (
            <>
              {/* Name */}
              <div className="space-y-1.5">
                <label className="text-xs text-muted">Name</label>
                <input
                  type="text"
                  value={name}
                  onChange={e => setName(e.target.value)}
                  placeholder="Stripe production"
                  className="w-full bg-base border border-border-strong rounded-lg px-3 py-2 text-sm text-ink placeholder-faint focus:outline-none focus:border-indigo-500 transition-colors duration-200 ease-(--ease-considered)"
                />
              </div>

              {/* Slug */}
              <div className="space-y-1.5">
                <label className="text-xs text-muted">Slug</label>
                <div className="flex items-center bg-base border border-border-strong rounded-lg px-3 py-2 focus-within:border-indigo-500 transition-colors duration-200 ease-(--ease-considered)">
                  <span className="text-faint text-sm font-mono">/i/</span>
                  <input
                    type="text"
                    value={slug}
                    onChange={e => handleSlugChange(e.target.value)}
                    placeholder="stripe-production"
                    className="flex-1 bg-transparent text-sm font-mono text-ink placeholder-faint focus:outline-none"
                  />
                </div>
                {slugError ? (
                  <p className="text-xs text-red-400">{slugError}</p>
                ) : slug.length >= 3 && (
                  <p className="text-xs text-faint font-mono">
                    {BASE_URL}/i/{slug}
                  </p>
                )}
              </div>

              {/* Description */}
              <div className="space-y-1.5">
                <label className="text-xs text-muted">
                  Description{' '}
                  <span className="text-faint">(optional)</span>
                </label>
                <input
                  type="text"
                  value={description}
                  onChange={e => setDescription(e.target.value)}
                  placeholder="Live Stripe payment events"
                  className="w-full bg-base border border-border-strong rounded-lg px-3 py-2 text-sm text-ink placeholder-faint focus:outline-none focus:border-indigo-500 transition-colors duration-200 ease-(--ease-considered)"
                />
              </div>

              {error && (
                <p className="text-xs text-red-400 bg-red-500/10 px-3 py-2 rounded-lg">
                  {error}
                </p>
              )}

              <div className="flex gap-3 pt-1">
                <button
                  onClick={onClose}
                  className="flex-1 px-4 py-2 rounded-lg border border-border-strong text-sm text-muted hover:text-ink hover:border-faint transition-colors duration-200 ease-(--ease-considered)"
                >
                  Cancel
                </button>
                <button
                  onClick={handleSubmit}
                  disabled={loading || !!slugError || !slug || !name}
                  className="flex-1 px-4 py-2 rounded-lg bg-indigo-500 hover:bg-indigo-400 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm font-medium transition-all duration-200 ease-(--ease-considered)"
                >
                  {loading ? 'Creating…' : 'Create endpoint'}
                </button>
              </div>
            </>
          )}

        </div>
      </div>
    </Portal>
  )
}
