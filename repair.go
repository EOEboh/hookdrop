package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/EOEboh/hookdrop/internal/billing"
	"github.com/EOEboh/hookdrop/internal/store"
)

// repairPaystackSubscriptions reconciles rows that hold a transaction
// reference in provider_sub_id instead of a subscription code.
//
// Those rows predate subscription.create being applied correctly. They cannot
// be cancelled through the Paystack API, and their current_period_end was
// derived at signup and never refreshed by a renewal — so once access is gated
// on the period, they read as lapsed.
//
// Dry run by default: nothing is written unless apply is true.
func repairPaystackSubscriptions(
	st *store.Store,
	ps *billing.PaystackProvider,
	apply bool,
) error {
	rows, err := st.ListPaystackSubscriptionsNeedingRepair()
	if err != nil {
		return fmt.Errorf("list subscriptions: %w", err)
	}

	if len(rows) == 0 {
		log.Printf("repair: nothing to do — every Paystack row already holds a subscription code")
		return nil
	}

	mode := "DRY RUN — nothing will be written"
	if apply {
		mode = "APPLYING changes"
	}
	log.Printf("repair: %d Paystack row(s) need repair (%s)", len(rows), mode)

	var repaired, failed int
	for _, sub := range rows {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		found, err := ps.LookupCustomerSubscription(ctx, sub.ProviderCustomerID)
		cancel()

		if err != nil {
			// A customer with no subscription at Paystack is a result, not a
			// failure: they paid with an authorization that cannot be reused
			// (a bank transfer), so there is nothing to reconcile and the
			// period simply expires. Record that rather than reporting it as
			// unreconcilable.
			if strings.Contains(err.Error(), "no subscriptions") {
				log.Printf("repair:   user=%s customer=%s has no Paystack subscription",
					sub.UserID, sub.ProviderCustomerID)
				log.Printf("repair:     auto_renews        %t -> false (paid once, will not renew)",
					sub.AutoRenews)
				if apply {
					if err := st.MarkSubscriptionNonRecurring(sub.ID); err != nil {
						log.Printf("repair:     WRITE FAILED: %v", err)
						failed++
						continue
					}
					repaired++
				}
				continue
			}
			// Report and continue: one unreconcilable customer must not stop
			// the rest of the pass.
			log.Printf("repair:   user=%s customer=%s SKIPPED: %v",
				sub.UserID, sub.ProviderCustomerID, err)
			failed++
			continue
		}

		var periodEnd *time.Time
		if found.NextPaymentAt > 0 {
			t := time.Unix(found.NextPaymentAt, 0).UTC()
			periodEnd = &t
		}

		log.Printf("repair:   user=%s", sub.UserID)
		log.Printf("repair:     provider_sub_id    %q -> %q", sub.ProviderSubID, found.SubscriptionCode)
		log.Printf("repair:     status             %q -> %q", sub.Status, found.Status)
		log.Printf("repair:     current_period_end %v -> %v", fmtTime(sub.CurrentPeriodEnd), fmtTime(periodEnd))

		if !apply {
			continue
		}
		if err := st.RepairPaystackSubscription(
			sub.ID, found.SubscriptionCode, found.Status, periodEnd,
		); err != nil {
			log.Printf("repair:     WRITE FAILED: %v", err)
			failed++
			continue
		}
		repaired++
	}

	if apply {
		log.Printf("repair: done — %d repaired, %d failed", repaired, failed)
	} else {
		log.Printf("repair: dry run complete — %d would be repaired, %d could not be resolved", len(rows)-failed, failed)
		log.Printf("repair: re-run with -apply to write these changes")
	}
	if failed > 0 {
		return fmt.Errorf("%d subscription(s) could not be repaired", failed)
	}
	return nil
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return "<nil>"
	}
	return t.UTC().Format(time.RFC3339)
}
