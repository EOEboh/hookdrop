export function Spinner({ size = 4 }: { size?: number }) {
  return (
    <span
      className={`inline-block w-${size} h-${size} rounded-full border-2 border-zinc-700 border-t-emerald-400 animate-spin`}
    />
  )
}
