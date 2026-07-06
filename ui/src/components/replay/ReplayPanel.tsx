import { useState, useEffect } from 'react'
import type { CapturedRequest } from '../../types'
import { useReplay } from '../../hooks/useReplay'
import { safeParseBody, tryPrettyPrint } from '../../lib/json'
import { ReplayResponse } from './ReplayResponse'
import { Spinner } from '../ui/Spinner'
import { ArrowRightIcon } from '../ui/icons'
import { usePostHog } from '@posthog/react'

export function ReplayPanel({ request }: { request: CapturedRequest }) {
  const rawBody = safeParseBody(request.body)
  const { pretty } = tryPrettyPrint(rawBody)
  const posthog = usePostHog()

  const [targetUrl, setTargetUrl] = useState('http://localhost:3000')
  const [body, setBody] = useState(pretty)
  const { replay, loading, response, error, reset } = useReplay()


  useEffect(() => {
    if (response) {
      posthog?.capture('replay_succeeded', {
        status:     response.status,
        latency_ms: response.latency_ms,
      })
    }
  }, [response])

  async function handleReplay() {
    posthog?.capture('replay_fired', {
      has_body_override:   body !== pretty,
      target_is_localhost: targetUrl.includes('localhost'),
    })
    await replay(request, targetUrl, body)
  }

  return (
    <div data-tour="replay-panel" className="px-6 py-5 space-y-4">

      {/* Target URL */}
      <div className="space-y-1.5">
        <label className="text-[11px] font-semibold text-muted uppercase tracking-widest block">
          Target URL
        </label>
        <input
          type="text"
          value={targetUrl}
          onChange={(e) => { setTargetUrl(e.target.value); reset() }}
          className="w-full bg-surface border border-border rounded-lg px-3 py-2 text-sm font-mono text-ink focus:outline-none focus:border-border-strong transition-colors duration-200 ease-(--ease-considered) placeholder-faint"
          placeholder="http://localhost:3000/webhook"
        />
      </div>

      {/* Editable body */}
      <div className="space-y-1.5">
        <label className="text-[11px] font-semibold text-muted uppercase tracking-widest block">
          Body
        </label>
        <textarea
          value={body}
          onChange={(e) => { setBody(e.target.value); reset() }}
          rows={6}
          className="w-full bg-surface border border-border rounded-lg px-3 py-2 text-xs font-mono text-ink focus:outline-none focus:border-border-strong transition-colors duration-200 ease-(--ease-considered) resize-none leading-relaxed placeholder-faint"
        />
      </div>

      {/* Send button */}
      <button
        onClick={handleReplay}
        disabled={loading || !targetUrl}
        className="flex items-center gap-2 px-4 py-2 bg-indigo-500 hover:bg-indigo-400 active:bg-indigo-600 active:scale-[0.98] disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-medium rounded-lg transition-all duration-200 ease-(--ease-considered)"
      >
        {loading ? (
          <>
            <Spinner size={4} />
            <span>Replaying…</span>
          </>
        ) : (
          <>
            <ArrowRightIcon className="w-4 h-4" />
            <span>Replay request</span>
          </>
        )}
      </button>

      {/* Error */}
      {error && (
        <div className="flex items-start gap-2 font-mono text-xs text-red-400 bg-red-500/10 border border-red-500/20 px-3 py-2.5 rounded-lg">
          <span className="select-none shrink-0">!</span>
          <span className="break-all">{error}</span>
        </div>
      )}

      {/* Response */}
      {response && (
        <div className="space-y-3">
          <div className="border-t border-border/60" />
          <ReplayResponse response={response} />
        </div>
      )}

    </div>
  )
}
