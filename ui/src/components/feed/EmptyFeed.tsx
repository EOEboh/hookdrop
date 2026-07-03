import { TerminalIcon } from '../ui/icons'

export function EmptyFeed() {
  return (
    <div className="flex flex-col items-center justify-center py-12 px-5 text-center space-y-4">

      {/* Icon */}
      <div className="w-12 h-12 rounded-xl bg-zinc-900 border border-zinc-800 flex items-center justify-center">
        <TerminalIcon className="w-5 h-5 text-zinc-500" />
      </div>

      <div className="space-y-1">
        <p className="text-sm font-medium text-zinc-300">Waiting for requests</p>
        <p className="text-xs text-zinc-600 leading-relaxed">
          Send a webhook to your endpoint to get started
        </p>
      </div>

      {/* Quick test snippet */}
      <div className="w-full bg-zinc-900 border border-zinc-800 rounded-lg p-3 text-left space-y-1.5">
        <p className="text-[10px] font-semibold text-zinc-600 uppercase tracking-widest">
          Quick test
        </p>
        <code className="block text-[11px] font-mono text-zinc-500 leading-relaxed whitespace-pre">
          {`curl -X POST \\
  -H "Content-Type: application/json" \\
  -d '{"event":"test"}' \\
  [your endpoint]`}
        </code>
      </div>

    </div>
  )
}
