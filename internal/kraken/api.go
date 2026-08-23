package kraken

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const (
	balancePath            = "/0/private/Balance"
	addOrderPath           = "/0/private/AddOrder"
	fundingFeesBasePath    = "/funding/v1/fees/"
	fundingWithdrawalsPath = "/funding/v1/withdrawals"
)

// Balance maps Kraken asset symbols to decimal balance strings.
type Balance map[string]string

// OrderDescription describes an accepted order.
type OrderDescription struct {
	Order string `json:"order"`
	Close string `json:"close,omitempty"`
}

// AddOrderResponse is returned by Kraken's AddOrder endpoint.
type AddOrderResponse struct {
	Description    OrderDescription `json:"descr"`
	TransactionIDs []string         `json:"txid,omitempty"`
}

// AddOrderParams contains the supported Spot order fields.
type AddOrderParams struct {
	Side                string
	OrderType           string
	Pair                string
	Volume              string
	Price               string
	SecondaryPrice      string
	Trigger             string
	Leverage            string
	ReduceOnly          bool
	SelfTradePrevention string
	OrderFlags          string
	TimeInForce         string
	StartTime           string
	ExpireTime          string
	UserReference       *int64
	ClientOrderID       string
	Validate            bool
	Deadline            string
}

// FundingAsset identifies an asset in the Funding API.
type FundingAsset struct {
	Class string `json:"class"`
	Name  string `json:"name"`
}

// FundingAssetAmount is an exact amount and its asset.
type FundingAssetAmount struct {
	Asset  FundingAsset `json:"asset"`
	Amount string       `json:"amount"`
}

// FundingAmount is the nested amount shape returned by withdrawal creation.
type FundingAmount struct {
	AssetAmount FundingAssetAmount `json:"asset_amount"`
}

// FundingFeeQuote pins the fee rate for a subsequent withdrawal.
type FundingFeeQuote struct {
	Fee                FundingAssetAmount `json:"fee"`
	GrossAmount        FundingAssetAmount `json:"gross_amount"`
	NetAmount          FundingAssetAmount `json:"net_amount"`
	WithdrawalFeeToken string             `json:"withdrawal_fee_token"`
}

// FundingWithdrawalParams selects a stable funding method and saved address.
// The amount is treated as the total account debit, including the quoted fee.
type FundingWithdrawalParams struct {
	MethodID  string
	AddressID string
	Asset     string
	Amount    string
	FeeToken  string
}

// FundingWithdrawalResponse describes an accepted Funding API withdrawal.
type FundingWithdrawalResponse struct {
	WithdrawalID      string        `json:"withdrawal_id"`
	NetAmount         FundingAmount `json:"net_amount"`
	GrossAmount       FundingAmount `json:"gross_amount"`
	Fee               FundingAmount `json:"fee"`
	ApprovalRequestID string        `json:"approval_request_id,omitempty"`
}

// Balance returns all balances for the authenticated account.
func (client *Client) Balance(ctx context.Context) (Balance, error) {
	var balance Balance
	if err := client.privateRequest(ctx, balancePath, nil, &balance); err != nil {
		return nil, err
	}
	return balance, nil
}

// AssetBalance returns one asset balance, or zero when Kraken omits the asset.
func (client *Client) AssetBalance(ctx context.Context, asset string) (string, error) {
	balances, err := client.Balance(ctx)
	if err != nil {
		return "", err
	}
	if balance, ok := balances[asset]; ok {
		return balance, nil
	}
	return "0.00000000", nil
}

// AddOrder submits an order to Kraken.
func (client *Client) AddOrder(ctx context.Context, params AddOrderParams) (AddOrderResponse, error) {
	if params.Side == "" || params.OrderType == "" || params.Pair == "" || params.Volume == "" {
		return AddOrderResponse{}, fmt.Errorf("side, order type, pair, and volume are required")
	}
	form := url.Values{
		"type":      {params.Side},
		"ordertype": {params.OrderType},
		"pair":      {params.Pair},
		"volume":    {params.Volume},
	}
	setOptional(form, "price", params.Price)
	setOptional(form, "price2", params.SecondaryPrice)
	setOptional(form, "trigger", params.Trigger)
	setOptional(form, "leverage", params.Leverage)
	setOptional(form, "stptype", params.SelfTradePrevention)
	setOptional(form, "oflags", params.OrderFlags)
	setOptional(form, "timeinforce", params.TimeInForce)
	setOptional(form, "starttm", params.StartTime)
	setOptional(form, "expiretm", params.ExpireTime)
	setOptional(form, "cl_ord_id", params.ClientOrderID)
	setOptional(form, "deadline", params.Deadline)
	if params.ReduceOnly {
		form.Set("reduce_only", "true")
	}
	if params.Validate {
		form.Set("validate", "true")
	}
	if params.UserReference != nil {
		form.Set("userref", strconv.FormatInt(*params.UserReference, 10))
	}

	var response AddOrderResponse
	if err := client.privateRequest(ctx, addOrderPath, form, &response); err != nil {
		return AddOrderResponse{}, err
	}
	return response, nil
}

// SellUSDCToUSD sells volume of USDC with a market order.
func (client *Client) SellUSDCToUSD(ctx context.Context, volume string) (AddOrderResponse, error) {
	return client.AddOrder(ctx, AddOrderParams{
		Side:      "sell",
		OrderType: "market",
		Pair:      "USDCUSD",
		Volume:    volume,
	})
}

// BuyUSDCWithUSD buys volume of USDC with a market order.
func (client *Client) BuyUSDCWithUSD(ctx context.Context, volume string) (AddOrderResponse, error) {
	return client.AddOrder(ctx, AddOrderParams{
		Side:      "buy",
		OrderType: "market",
		Pair:      "USDCUSD",
		Volume:    volume,
	})
}

// QuoteFundingWithdrawalFee calculates and pins the fee for a withdrawal whose
// amount is the total account debit, including fees.
func (client *Client) QuoteFundingWithdrawalFee(ctx context.Context, methodID, amount string) (FundingFeeQuote, error) {
	if methodID == "" || amount == "" {
		return FundingFeeQuote{}, fmt.Errorf("funding method ID and amount are required")
	}
	query := url.Values{
		"amount":       {amount},
		"fee_included": {"true"},
	}
	var quote FundingFeeQuote
	if err := client.fundingRequest(ctx, http.MethodGet, fundingFeesBasePath+url.PathEscape(methodID), query, nil, &quote); err != nil {
		return FundingFeeQuote{}, err
	}
	if quote.WithdrawalFeeToken == "" {
		return FundingFeeQuote{}, fmt.Errorf("Kraken Funding API fee quote did not include a withdrawal fee token")
	}
	return quote, nil
}

// CreateFundingWithdrawal withdraws through a stable Funding API method and
// address ID using a previously quoted fee token.
func (client *Client) CreateFundingWithdrawal(ctx context.Context, params FundingWithdrawalParams) (FundingWithdrawalResponse, error) {
	if params.MethodID == "" || params.AddressID == "" || params.Asset == "" || params.Amount == "" || params.FeeToken == "" {
		return FundingWithdrawalResponse{}, fmt.Errorf("funding method ID, address ID, asset, amount, and fee token are required")
	}
	request := struct {
		Scope struct {
			MethodID string `json:"method_id"`
		} `json:"scope"`
		AddressID string        `json:"address_id"`
		Amount    FundingAmount `json:"amount"`
		Fee       struct {
			QuotedFee struct {
				Token string `json:"token"`
			} `json:"quoted_fee"`
			FeeIncluded bool `json:"fee_included"`
		} `json:"fee"`
	}{}
	request.Scope.MethodID = params.MethodID
	request.AddressID = params.AddressID
	request.Amount.AssetAmount = FundingAssetAmount{
		Asset:  FundingAsset{Class: "currency", Name: params.Asset},
		Amount: params.Amount,
	}
	request.Fee.QuotedFee.Token = params.FeeToken
	request.Fee.FeeIncluded = true

	var response FundingWithdrawalResponse
	if err := client.fundingRequest(ctx, http.MethodPost, fundingWithdrawalsPath, nil, request, &response); err != nil {
		return FundingWithdrawalResponse{}, err
	}
	if response.WithdrawalID == "" {
		return FundingWithdrawalResponse{}, fmt.Errorf("Kraken Funding API response did not include a withdrawal ID")
	}
	return response, nil
}

func setOptional(values url.Values, key, value string) {
	if value != "" {
		values.Set(key, value)
	}
}
