import posthog from 'posthog-js'

// Only track in production 
const isProd = import.meta.env.PROD

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

export function identifyUser(userId: string, email: string) {
  if (!isProd) return
  posthog.identify(userId, { email })
}

export function resetUser() {
  if (!isProd) return
  posthog.reset()
}

export function trackPageView(path: string) {
  if (!isProd) return
  posthog.capture('$pageview', { $current_url: path })
}

export function trackEvent(
  event: AnalyticsEvent,
  properties?: Record<string, unknown>,
) {
  if (!isProd) return
  posthog.capture(event, properties)
}