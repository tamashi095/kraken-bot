// Package bot orchestrates the USDC-to-USD conversion and withdrawal workflow.
package bot

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"time"

	"kraken-bot/internal/decimal"
	"kraken-bot/internal/kraken"
)

// API is the subset of Kraken operations needed by Runner.
type API interface {
	AssetBalance(context.Context, string) (string, error)
	SellUSDCToUSD(context.Context, string) (kraken.AddOrderResponse, error)
	QuoteFundingWithdrawalFee(context.Context, string, string) (kraken.FundingFeeQuote, error)
	CreateFundingWithdrawal(context.Context, kraken.FundingWithdrawalParams) (kraken.FundingWithdrawalResponse, error)
}

// Runner executes one bot cycle.
type Runner struct {
	API                      API
	Output                   io.Writer
	MinimumWithdrawal        *big.Int
	MinimumWithdrawalDisplay string
	USDFundingMethodID       string
	USDFundingAddressID      string
	SettlementDelay          time.Duration
	Sleep                    func(context.Context, time.Duration) error
}

// Run converts all available USDC to USD and withdraws the USD balance when it
// meets the configured threshold.
func (runner Runner) Run(ctx context.Context) error {
	if runner.API == nil {
		return fmt.Errorf("Kraken API client is required")
	}
	if runner.Output == nil {
		return fmt.Errorf("output writer is required")
	}
	if runner.MinimumWithdrawal == nil {
		return fmt.Errorf("minimum withdrawal is required")
	}
	if runner.USDFundingMethodID == "" || runner.USDFundingAddressID == "" {
		return fmt.Errorf("USD funding method and address IDs are required")
	}
	if runner.Sleep == nil {
		runner.Sleep = sleepContext
	}

	fmt.Fprintln(runner.Output, "🤖 Kraken Bot Started")
	fmt.Fprintln(runner.Output)
	fmt.Fprintln(runner.Output, "📊 Checking USDC balance...")
	usdcBalance, err := runner.API.AssetBalance(ctx, "USDC")
	if err != nil {
		return fmt.Errorf("get USDC balance: %w", err)
	}
	usdcUnits, err := decimal.ParseUnits(usdcBalance, 8)
	if err != nil {
		return fmt.Errorf("parse USDC balance returned by Kraken: %w", err)
	}
	fmt.Fprintf(runner.Output, "   USDC Balance: %s\n", usdcBalance)

	if usdcUnits.Sign() > 0 {
		fmt.Fprintf(runner.Output, "\n💱 Selling %s USDC to USD...\n", usdcBalance)
		order, err := runner.API.SellUSDCToUSD(ctx, usdcBalance)
		if err != nil {
			return fmt.Errorf("sell USDC to USD: %w", err)
		}
		if len(order.TransactionIDs) > 0 {
			fmt.Fprintln(runner.Output, "   ✅ Order placed successfully")
			fmt.Fprintf(runner.Output, "   Transaction ID: %s\n", join(order.TransactionIDs))
			fmt.Fprintf(runner.Output, "   Order: %s\n", order.Description.Order)
		}
	} else {
		fmt.Fprintln(runner.Output, "   ℹ️  No USDC balance to sell")
	}

	fmt.Fprintf(runner.Output, "   ℹ️  Waiting %s before checking USD balance...\n", runner.SettlementDelay)
	if err := runner.Sleep(ctx, runner.SettlementDelay); err != nil {
		return err
	}

	fmt.Fprintln(runner.Output)
	fmt.Fprintln(runner.Output, "💵 Checking USD balance...")
	usdBalance, err := runner.API.AssetBalance(ctx, "ZUSD")
	if err != nil {
		return fmt.Errorf("get USD balance: %w", err)
	}
	usdUnits, err := decimal.ParseUnits(usdBalance, 4)
	if err != nil {
		return fmt.Errorf("parse USD balance returned by Kraken: %w", err)
	}
	fmt.Fprintf(runner.Output, "   USD Balance: $%s\n", usdBalance)

	if usdUnits.Cmp(runner.MinimumWithdrawal) >= 0 {
		fmt.Fprintf(runner.Output, "\n🧾 Quoting the USD withdrawal fee...\n")
		quote, err := runner.API.QuoteFundingWithdrawalFee(ctx, runner.USDFundingMethodID, usdBalance)
		if err != nil {
			return fmt.Errorf("quote USD withdrawal fee: %w", err)
		}
		for name, quotedAmount := range map[string]kraken.FundingAssetAmount{
			"fee": quote.Fee, "gross amount": quote.GrossAmount, "net amount": quote.NetAmount,
		} {
			if quotedAmount.Asset.Class != "currency" || quotedAmount.Asset.Name != "USD" {
				return fmt.Errorf("USD funding method returned a %s in %s:%s", name, quotedAmount.Asset.Class, quotedAmount.Asset.Name)
			}
		}
		if quote.Fee.Amount == "" || quote.NetAmount.Amount == "" {
			return fmt.Errorf("USD funding fee quote did not include fee and net amounts")
		}
		fmt.Fprintf(runner.Output, "   Fee: $%s; destination receives: $%s\n", quote.Fee.Amount, quote.NetAmount.Amount)

		fmt.Fprintf(runner.Output, "\n🏦 Withdrawing $%s including the quoted fee...\n", usdBalance)
		withdrawal, err := runner.API.CreateFundingWithdrawal(ctx, kraken.FundingWithdrawalParams{
			MethodID:  runner.USDFundingMethodID,
			AddressID: runner.USDFundingAddressID,
			Asset:     "USD",
			Amount:    usdBalance,
			FeeToken:  quote.WithdrawalFeeToken,
		})
		if err != nil {
			return fmt.Errorf("withdraw USD: %w", err)
		}
		fmt.Fprintln(runner.Output, "   ✅ Withdrawal initiated")
		fmt.Fprintf(runner.Output, "   Withdrawal ID: %s\n", withdrawal.WithdrawalID)
		if withdrawal.ApprovalRequestID != "" {
			fmt.Fprintf(runner.Output, "   Approval required: %s\n", withdrawal.ApprovalRequestID)
		}
	} else {
		fmt.Fprintf(runner.Output, "   ℹ️  USD balance below minimum withdrawal threshold ($%s)\n", runner.MinimumWithdrawalDisplay)
	}

	fmt.Fprintln(runner.Output)
	fmt.Fprintln(runner.Output, "✨ Bot completed successfully!")
	return nil
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func join(values []string) string {
	if len(values) == 0 {
		return ""
	}
	result := values[0]
	for _, value := range values[1:] {
		result += ", " + value
	}
	return result
}
