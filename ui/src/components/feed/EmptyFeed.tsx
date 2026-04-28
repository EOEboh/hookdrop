export function EmptyFeed() {
  return (
    <div className="flex flex-col items-center justify-center h-full py-16 px-6 text-center">
      <div className="w-10 h-10 rounded-full bg-zinc-800 flex items-center justify-center mb-4">
        <span className="text-lg">⏳</span>
      </div>
      <p className="text-zinc-300 text-sm font-medium mb-1">Waiting for requests</p>
      <p className="text-zinc-600 text-xs">
        Copy your endpoint above and send a webhook to it
      </p>
    </div>
  )
}