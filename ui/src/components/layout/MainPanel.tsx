import type { CapturedRequest } from '../../types'
import { RequestDetail } from '../detail/RequestDetail'
import { MousePointerIcon } from '../ui/icons'

export function MainPanel({ selected }: { selected: CapturedRequest | null }) {
  if (!selected) {
    return (
      <div className="flex-1 flex items-center justify-center text-center p-12">
        <div className="space-y-4 max-w-xs">
          <div className="w-14 h-14 rounded-xl bg-zinc-900 border border-zinc-800 flex items-center justify-center mx-auto">
            <MousePointerIcon className="w-6 h-6 text-zinc-600" />
          </div>
          <div className="space-y-1.5">
            <p className="text-sm font-medium text-zinc-300">No request selected</p>
            <p className="text-xs text-zinc-600 leading-relaxed">
              Select a request from the sidebar to inspect its headers, body, and replay it
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    // key forces remount + fade-in animation on request change
    <main
      key={selected.id}
      className="flex-1 min-w-0 h-screen overflow-y-auto bg-zinc-950 animate-fade-in"
    >
      <RequestDetail request={selected} />
    </main>
  )
}
