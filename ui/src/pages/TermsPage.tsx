export function TermsPage() {
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
            Terms of Service
          </h1>
          <p className="text-sm text-zinc-500">
            Last updated: July 2026
          </p>
        </div>

        <div className="space-y-8 text-sm leading-relaxed">

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              1. Acceptance
            </h2>
            <p>
              By using hookdrop you agree to these terms. If you do not agree,
              do not use the service. These terms apply to all users including
              free and paid accounts.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              2. What hookdrop provides
            </h2>
            <p>
              hookdrop is a webhook inspection and debugging tool. It provides
              temporary and permanent URLs that capture incoming HTTP requests,
              displays them in real time, and allows you to replay them to any
              target URL. hookdrop is provided as-is for development and
              debugging purposes.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              3. Acceptable use
            </h2>
            <p>You may not use hookdrop to:</p>
            <ul className="list-disc list-inside space-y-1 text-zinc-400 ml-2">
              <li>Receive, store, or process data you do not have the right to handle</li>
              <li>Conduct illegal activity of any kind</li>
              <li>Attempt to disrupt or overload the service</li>
              <li>Circumvent rate limits or security measures</li>
              <li>Resell or redistribute the service without written permission</li>
            </ul>
            <p>
              We reserve the right to suspend or terminate accounts that violate
              these terms without notice.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              4. Webhook payload responsibility
            </h2>
            <p>
              You are solely responsible for the data sent to your hookdrop
              endpoints. If webhook payloads contain personal data belonging to
              third parties, you confirm you have the legal basis to process
              that data. hookdrop acts as a data processor on your behalf and
              processes payload data only as instructed by your configuration.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              5. Subscriptions and billing
            </h2>
            <p>
              The free plan is available indefinitely within its stated limits.
              Pro plan subscriptions are billed monthly or annually depending
              on your selection at checkout.
            </p>
            <p>
              Subscriptions auto-renew at the end of each billing period unless
              cancelled before the renewal date. You can cancel at any time from
              your billing settings — access continues until the end of the paid
              period with no partial refunds for unused time.
            </p>
            <p>
              If you are unhappy with hookdrop within 7 days of your first
              payment, email{' '}
              <a
                href="mailto:support@hookdrop.app"
                className="text-emerald-400 hover:text-emerald-300"
              >
                support@hookdrop.app
              </a>{' '}
              for a full refund. This does not apply to free trial periods.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              6. Data retention
            </h2>
            <p>
              Captured webhook requests are automatically deleted after 7 days
              on the free plan and 90 days on the Pro plan. hookdrop does not
              guarantee the preservation of any captured data beyond these
              periods. Do not use hookdrop as a permanent record of webhook
              events.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              7. Service availability
            </h2>
            <p>
              hookdrop aims for high availability but does not guarantee
              uninterrupted service. We are not liable for any losses resulting
              from service downtime, data loss, or webhook events missed during
              an outage. Check{' '}
              <a
                href="https://status.hookdrop.app"
                className="text-emerald-400 hover:text-emerald-300"
                target="_blank"
                rel="noopener noreferrer"
              >
                status.hookdrop.app
              </a>{' '}
              for current service status.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              8. Limitation of liability
            </h2>
            <p>
              hookdrop is provided without warranty of any kind. To the maximum
              extent permitted by law, hookdrop's total liability for any claim
              arising from use of the service is limited to the amount you paid
              in the 3 months preceding the claim, or $10 USD, whichever is
              greater.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              9. Changes to these terms
            </h2>
            <p>
              We may update these terms from time to time. Significant changes
              will be communicated by email to registered users at least 14 days
              before taking effect. Continued use of hookdrop after changes
              take effect constitutes acceptance of the new terms.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              10. Governing law
            </h2>
            <p>
              These terms are governed by the laws of the Federal Republic of
              Nigeria. Any disputes shall be resolved in Nigerian courts unless
              otherwise agreed in writing.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-base font-semibold text-zinc-200">
              11. Contact
            </h2>
            <p>
              Questions about these terms:{' '}
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