export function Spinner({ size = 4 }: { size?: number }) {
  return (
    <span
      className={`inline-block w-${size} h-${size} rounded-full border-2 border-border-strong border-t-indigo-400 animate-spin`}
    />
  )
}
