package app

import (
	"context"
	"fmt"

	"github.com/durgakar/reward-system-users/internal/rewards"
)

// TestVoucherifyConnection verifies Voucherify API credentials from config.
func (a *Application) TestVoucherifyConnection(ctx context.Context) error {
	p := rewards.NewVoucherifyProvider(a.Config.Voucherify)
	if a.Config.Voucherify.LoyaltyID == "" {
		return fmt.Errorf("voucherify.loyalty_id is required — find it in Voucherify Dashboard → Loyalty → your campaign → Campaign ID (camp_...)")
	}
	if err := p.TestConnection(ctx); err != nil {
		return fmt.Errorf("voucherify connection failed: %w", err)
	}
	return nil
}
