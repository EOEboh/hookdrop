export function PrivacyPage() {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-300">
      <div className="max-w-3xl mx-auto px-6 py-16 space-y-8">

        {/* Header */}
        <div className="space-y-2">
          <a
            href="/"
            className="text-sm text-zinc-600 hover:text-zinc-400 transition-colors"
          >
            ← Back to hookdrop
          </a>
          <h1 className="text-2xl font-semibold text-zinc-100 mt-4">
            Privacy Policy
          </h1>
          <p className="text-sm text-zinc-500">
            Last updated: July 2026
          </p>
        </div>

        <div className="space-y-8 text-sm leading-relaxed">

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              1. What we collect
            </h2>
            <p>
              hookdrop collects the minimum information needed to provide the
              service. This includes your email address (used for authentication),
              webhook request data you send to your endpoints, and basic usage
              analytics to improve the product.
            </p>
            <p>
              We do not collect payment card details — payments are processed
              directly by Paystack (Nigerian and African users) or Lemon Squeezy
              (international users). We never see your card number.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              2. Webhook payload data
            </h2>
            <p>
              Webhook payloads you receive through hookdrop may contain personal
              data belonging to your customers — for example, names, email
              addresses, or payment amounts included in Stripe or Paystack
              webhook events.
            </p>
            <p>
              hookdrop processes this data on your behalf as a data processor.
              You remain the data controller and are responsible for ensuring you
              have the appropriate rights to process this data. Payload data is
              stored for 7 days on the free plan and 90 days on the Pro plan,
              after which it is automatically and permanently deleted.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              3. How we store your data
            </h2>
            <p>
              All data is stored on servers located in Frankfurt, Germany
              (European Union), operated by Hetzner Online GmbH. Data is
              encrypted in transit using TLS 1.2 or higher. Daily encrypted
              backups are stored on Cloudflare R2 infrastructure.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              4. Third-party services
            </h2>
            <p>
              hookdrop uses the following third-party services to operate:
            </p>
            <ul className="list-disc list-inside space-y-1 text-zinc-400 ml-2">
              <li>Resend — transactional email (magic link login)</li>
              <li>Paystack — payment processing for Nigerian and African users</li>
              <li>Lemon Squeezy — payment processing for international users</li>
              <li>PostHog — product analytics (anonymised usage data)</li>
              <li>Hetzner — server infrastructure</li>
              <li>Cloudflare — CDN, DNS, and backup storage</li>
            </ul>
            <p>
              Each of these providers has their own privacy policy governing
              how they handle data passed to them.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              5. Your rights
            </h2>
            <p>
              You have the right to access, correct, or delete your personal
              data at any time. To request deletion of your account and all
              associated data, email{' '}
              <a
                href="mailto:support@hookdrop.app"
                className="text-emerald-400 hover:text-emerald-300"
              >
                support@hookdrop.app
              </a>
              . We will process your request within 30 days.
            </p>
            <p>
              If you are located in the European Union, you have additional
              rights under GDPR including the right to data portability and
              the right to lodge a complaint with your local supervisory authority.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              6. Cookies and analytics
            </h2>
            <p>
              hookdrop uses PostHog for product analytics. Analytics data is
              anonymised and used only to understand how the product is used
              and where it can be improved. We do not sell analytics data to
              third parties. We do not use advertising cookies.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              7. Contact
            </h2>
            <p>
              Questions about this policy:{' '}
              <a
                href="mailto:support@hookdrop.app"
                className="text-emerald-400 hover:text-emerald-300"
              >
                support@hookdrop.app
              </a>
            </p>
          </section>

        </div>
      </div>
    </div>
  )
}