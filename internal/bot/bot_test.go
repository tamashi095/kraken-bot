package bot

import (
	"bytes"
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"kraken-bot/internal/kraken"
)

type fakeAPI struct {
	balances       map[string]string
	sold           string
	quotedMethodID string
	quotedAmount   string
	withdrawal     kraken.FundingWithdrawalParams
}

func (api *fakeAPI) AssetBalance(_ context.Context, asset string) (string, error) {
	return api.balances[asset], nil
}

func (api *fakeAPI) SellUSDCToUSD(_ context.Context, volume string) (kraken.AddOrderResponse, error) {
	api.sold = volume
	return kraken.AddOrderResponse{
		Description:    kraken.OrderDescription{Order: "sell USDCUSD"},
		TransactionIDs: []string{"ORDER-1"},
	}, nil
}

func (api *fakeAPI) QuoteFundingWithdrawalFee(_ context.Context, methodID, amount string) (kraken.FundingFeeQuote, error) {
	api.quotedMethodID = methodID
	api.quotedAmount = amount
	usd := kraken.FundingAsset{Class: "currency", Name: "USD"}
	return kraken.FundingFeeQuote{
		Fee:                kraken.FundingAssetAmount{Asset: usd, Amount: "0.2500"},
		GrossAmount:        kraken.FundingAssetAmount{Asset: usd, Amount: amount},
		NetAmount:          kraken.FundingAssetAmount{Asset: usd, Amount: "11.0000"},
		WithdrawalFeeToken: "FEE-TOKEN",
	}, nil
}

func (api *fakeAPI) CreateFundingWithdrawal(_ context.Context, params kraken.FundingWithdrawalParams) (kraken.FundingWithdrawalResponse, error) {
	api.withdrawal = params
	return kraken.FundingWithdrawalResponse{WithdrawalID: "FTVZI01-1234567890123456789012"}, nil
}

func TestRunnerSellsAndWithdraws(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{balances: map[string]string{"USDC": "12.50000000", "ZUSD": "11.2500"}}
	var output bytes.Buffer
	runner := Runner{
		API:                      api,
		Output:                   &output,
		MinimumWithdrawal:        big.NewInt(100000),
		MinimumWithdrawalDisplay: "10",
		USDFundingMethodID:       "d4ec4d52-b159-428e-ba64-f45455a978a1",
		USDFundingAddressID:      "ABR6SXP-SF6CY-VJMONY",
		SettlementDelay:          2 * time.Second,
		Sleep:                    func(context.Context, time.Duration) error { return nil },
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if api.sold != "12.50000000" {
		t.Errorf("sold = %q", api.sold)
	}
	if api.quotedMethodID != "d4ec4d52-b159-428e-ba64-f45455a978a1" || api.quotedAmount != "11.2500" {
		t.Errorf("quote = %q, %q", api.quotedMethodID, api.quotedAmount)
	}
	if api.withdrawal.MethodID != "d4ec4d52-b159-428e-ba64-f45455a978a1" || api.withdrawal.AddressID != "ABR6SXP-SF6CY-VJMONY" || api.withdrawal.Asset != "USD" || api.withdrawal.Amount != "11.2500" || api.withdrawal.FeeToken != "FEE-TOKEN" {
		t.Errorf("withdrawal = %#v", api.withdrawal)
	}
	for _, expected := range []string{"ORDER-1", "$0.2500", "FTVZI01-1234567890123456789012", "completed successfully"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("output does not contain %q", expected)
		}
	}
}

func TestRunnerSkipsBalancesBelowLimits(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{balances: map[string]string{"USDC": "0.00000000", "ZUSD": "9.9999"}}
	var output bytes.Buffer
	runner := Runner{
		API:                      api,
		Output:                   &output,
		MinimumWithdrawal:        big.NewInt(100000),
		MinimumWithdrawalDisplay: "10",
		USDFundingMethodID:       "d4ec4d52-b159-428e-ba64-f45455a978a1",
		USDFundingAddressID:      "ABR6SXP-SF6CY-VJMONY",
		Sleep:                    func(context.Context, time.Duration) error { return nil },
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if api.sold != "" || api.quotedAmount != "" || api.withdrawal.Amount != "" {
		t.Fatalf("unexpected action: sold %q, quoted %q, withdrew %q", api.sold, api.quotedAmount, api.withdrawal.Amount)
	}
	if !strings.Contains(output.String(), "below minimum withdrawal threshold") {
		t.Errorf("output = %q", output.String())
	}
}
