import type { CapturedRequest } from '../../types'
import { RequestDetail } from '../detail/RequestDetail'

export function MainPanel({ selected }: { selected: CapturedRequest | null }) {
  if (!selected) {
    return (
      <div className="flex-1 flex items-center justify-center text-center p-12">
        <div className="space-y-2">
          <p className="text-zinc-400 text-sm">Select a request to inspect it</p>
          <p className="text-zinc-700 text-xs">Captured requests appear in the sidebar</p>
        </div>
      </div>
    )
  }

  return (
    <main className="flex-1 min-w-0 h-screen overflow-y-auto bg-zinc-950">
      <RequestDetail request={selected} />
    </main>
  )
}