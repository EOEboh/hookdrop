import { EmptyState } from '../ui/EmptyState'

export function EmptyFeed() {
  return (
    <EmptyState
      variant="feed"
      title="Waiting for your first webhook"
      description="Point Stripe, Paystack, or any provider at your endpoint above — requests will land here in real time."
    >
      {/* Quick test snippet */}
      <div className="w-full bg-surface border border-border rounded-lg p-3 text-left space-y-1.5">
        <p className="text-[10px] font-semibold text-faint uppercase tracking-widest">
          Quick test
        </p>
        <code className="block text-[11px] font-mono text-muted leading-relaxed whitespace-pre">
          {`curl -X POST \\
  -H "Content-Type: application/json" \\
  -d '{"event":"test"}' \\
  [your endpoint]`}
        </code>
      </div>
    </EmptyState>
  )
}
