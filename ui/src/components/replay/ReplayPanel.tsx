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
    <div className="px-6 py-5 space-y-4">

      {/* Target URL */}
      <div className="space-y-1.5">
        <label className="text-[11px] font-semibold text-zinc-500 uppercase tracking-widest block">
          Target URL
        </label>
        <input
          type="text"
          value={targetUrl}
          onChange={(e) => { setTargetUrl(e.target.value); reset() }}
          className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-sm font-mono text-zinc-200 focus:outline-none focus:border-zinc-600 transition-colors placeholder-zinc-600"
          placeholder="http://localhost:3000/webhook"
        />
      </div>

      {/* Editable body */}
      <div className="space-y-1.5">
        <label className="text-[11px] font-semibold text-zinc-500 uppercase tracking-widest block">
          Body
        </label>
        <textarea
          value={body}
          onChange={(e) => { setBody(e.target.value); reset() }}
          rows={6}
          className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs font-mono text-zinc-300 focus:outline-none focus:border-zinc-600 transition-colors resize-none leading-relaxed placeholder-zinc-600"
        />
      </div>

      {/* Send button */}
      <button
        onClick={handleReplay}
        disabled={loading || !targetUrl}
        className="flex items-center gap-2 px-4 py-2 bg-emerald-600 hover:bg-emerald-500 active:bg-emerald-700 disabled:opacity-40 disabled:cursor-not-allowed text-white text-sm font-medium rounded-lg transition-colors"
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
          <div className="border-t border-zinc-800/60" />
          <ReplayResponse response={response} />
        </div>
      )}

    </div>
  )
}