export function Spinner({ size = 4 }: { size?: number }) {
  return (
    <span
      className={`inline-block w-${size} h-${size} border-2 border-zinc-600 border-t-emerald-400 rounded-full animate-spin`}
    />
  )
}