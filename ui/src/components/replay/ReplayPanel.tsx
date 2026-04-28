import { useState } from 'react'
import type { CapturedRequest } from '../../types'
import { useReplay } from '../../hooks/useReplay'
import { safeParseBody, tryPrettyPrint } from '../../lib/json'
import { ReplayResponse } from './ReplayResponse'
import { Spinner } from '../ui/Spinner'

export function ReplayPanel({ request }: { request: CapturedRequest }) {
  const rawBody = safeParseBody(request.body)
  const { pretty } = tryPrettyPrint(rawBody)

  const [targetUrl, setTargetUrl] = useState('http://localhost:3000')
  const [body, setBody] = useState(pretty)
  const { replay, loading, response, error, reset } = useReplay()

  async function handleReplay() {
    await replay(request, targetUrl, body)
  }

  return (
    <div className="px-6 py-4 space-y-4">
      {/* Target URL */}
      <div className="space-y-1.5">
        <label className="text-xs text-zinc-500">Target URL</label>
        <input
          type="text"
          value={targetUrl}
          onChange={(e) => { setTargetUrl(e.target.value); reset() }}
          className="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-sm font-mono text-zinc-200 focus:outline-none focus:border-emerald-500 transition-colors"
          placeholder="http://localhost:3000/webhook"
        />
      </div>

      {/* Editable body */}
      <div className="space-y-1.5">
        <label className="text-xs text-zinc-500">Body (editable)</label>
        <textarea
          value={body}
          onChange={(e) => { setBody(e.target.value); reset() }}
          rows={6}
          className="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-3 py-2 text-xs font-mono text-zinc-300 focus:outline-none focus:border-emerald-500 transition-colors resize-none"
        />
      </div>

      {/* Send button */}
      <button
        onClick={handleReplay}
        disabled={loading || !targetUrl}
        className="flex items-center gap-2 px-4 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm font-medium rounded-lg transition-colors"
      >
        {loading && <Spinner size={3} />}
        {loading ? 'Replaying...' : 'Replay request'}
      </button>

      {/* Error */}
      {error && (
        <p className="text-xs text-red-400 font-mono bg-red-500/10 px-3 py-2 rounded-lg">
          {error}
        </p>
      )}

      {/* Response */}
      {response && <ReplayResponse response={response} />}
    </div>
  )
}