import posthog from 'posthog-js'

// Typed event names 
export type AnalyticsEvent =
  | 'magic_link_requested'
  | 'magic_link_verified'
  | 'user_logged_out'
  | 'inbox_url_copied'
  | 'first_webhook_received'
  | 'request_inspected'
  | 'replay_fired'
  | 'replay_succeeded'
  | 'named_endpoint_created'
  | 'named_endpoint_deleted'
  | 'endpoint_limit_hit'
  | 'webhook_secret_added'
  | 'webhook_verified'
  | 'webhook_verification_failed'
  | 'filter_applied'
  | 'pricing_page_viewed'
  | 'upgrade_prompt_seen'
  | 'checkout_started'
  | 'checkout_completed'
  | 'trial_started'
  | 'subscription_cancelled'

// Use this ONLY outside React components (auth callbacks, context, etc.)
// Inside components use the usePostHog() hook from @posthog/react instead
export function identifyUser(userId: string, email: string) {
  posthog.identify(userId, { email })
}

export function resetUser() {
  posthog.reset()
}

export function trackPageView(path: string) {
  posthog.capture('$pageview', { $current_url: path })
}

export function trackEvent(
  event: AnalyticsEvent,
  properties?: Record<string, unknown>,
) {
  posthog.capture(event, properties)
}