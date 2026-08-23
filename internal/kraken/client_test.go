package kraken

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGenerateSignatureMatchesKrakenVector(t *testing.T) {
	t.Parallel()

	secret, err := base64.StdEncoding.DecodeString("kQH5HW/8p1uGOVjbgWA7FunAmGO8lsSUXNsu3eow76sz84Q18fWxnyRzBHCd3pd5nE9qa99HAZtuZuj6F1huXg==")
	if err != nil {
		t.Fatal(err)
	}
	nonce := "1616492376594"
	payload := "nonce=1616492376594&ordertype=limit&pair=XBTUSD&price=37500&type=buy&volume=1.25"
	want := "4/dpxb3iT4tp/ZCVEwSnEsLxx0bqyhLpdfOpc6fn7OR8+UClSV5n9E6aSS8MPtnRfp32bAb0nmbRn6H8ndwLUQ=="

	if got := generateSignature(addOrderPath, nonce, payload, secret); got != want {
		t.Fatalf("generateSignature() = %q, want %q", got, want)
	}
}

func TestGenerateFundingSignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		signedPath string
		payload    string
		want       string
	}{
		{
			name:       "GET with signed query",
			signedPath: "/funding/v1/fees/d4ec4d52-b159-428e-ba64-f45455a978a1?amount=11.0000&fee_included=true",
			want:       "dzjdZToolusckkyWLHCTrBhWIS8E45AnM16Q2PkwQlS6g0WJUl5+FSg4SUhol3BG94fAu1brVDZN8TAulD8nmg==",
		},
		{
			name:       "POST with JSON body",
			signedPath: fundingWithdrawalsPath,
			payload:    `{"scope":{"method_id":"d4ec4d52-b159-428e-ba64-f45455a978a1"},"address_id":"ABR6SXP-SF6CY-VJMONY","amount":{"asset_amount":{"asset":{"class":"currency","name":"USD"},"amount":"11.0000"}},"fee":{"quoted_fee":{"token":"FEE-TOKEN"},"fee_included":true}}`,
			want:       "i9esIv8iMY9OQj1Kz021R6g6f99ibXWYW8wXznYXkWnM20OCCNK2KQCKF/hDpJd8+pCF11CUuuybZtDEwYqO4g==",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := generateFundingSignature(tt.signedPath, "1616492376594", []byte(tt.payload), []byte("test-secret"))
			if got != tt.want {
				t.Fatalf("generateFundingSignature() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientWorkflowRequests(t *testing.T) {
	t.Parallel()

	secret := []byte("test-secret")
	methodID := "d4ec4d52-b159-428e-ba64-f45455a978a1"
	addressID := "ABR6SXP-SF6CY-VJMONY"
	requestNumber := 0
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestNumber++
		var body []byte
		if request.Body != nil {
			var err error
			body, err = io.ReadAll(request.Body)
			if err != nil {
				return nil, err
			}
		}
		if request.Header.Get("API-Key") != "public" {
			t.Errorf("API-Key = %q", request.Header.Get("API-Key"))
		}

		var values url.Values
		var wantSignature string
		if strings.HasPrefix(request.URL.Path, "/funding/") {
			nonce := request.Header.Get("API-Nonce")
			signedPath := request.URL.EscapedPath()
			if request.URL.RawQuery != "" {
				signedPath += "?" + request.URL.RawQuery
			}
			wantSignature = generateFundingSignature(signedPath, nonce, body, secret)
		} else {
			var err error
			values, err = url.ParseQuery(string(body))
			if err != nil {
				return nil, err
			}
			wantSignature = generateSignature(request.URL.Path, values.Get("nonce"), string(body), secret)
		}
		if request.Header.Get("API-Sign") != wantSignature {
			t.Errorf("API-Sign = %q, want %q", request.Header.Get("API-Sign"), wantSignature)
		}

		var responseBody string
		switch request.URL.Path {
		case balancePath:
			responseBody = `{"error":[],"result":{"USDC":"12.50000000","ZUSD":"11.0000"}}`
		case addOrderPath:
			if values.Get("type") != "sell" || values.Get("ordertype") != "market" || values.Get("pair") != "USDCUSD" || values.Get("volume") != "12.50000000" {
				t.Errorf("AddOrder form = %v", values)
			}
			responseBody = `{"error":[],"result":{"descr":{"order":"sell 12.5 USDCUSD"},"txid":["ORDER-1"]}}`
		case fundingFeesBasePath + methodID:
			if request.Method != http.MethodGet || request.URL.Query().Get("amount") != "11.0000" || request.URL.Query().Get("fee_included") != "true" {
				t.Errorf("funding fee request = %s %s", request.Method, request.URL.String())
			}
			responseBody = `{"fee":{"asset":{"class":"currency","name":"USD"},"amount":"0.2500"},"gross_amount":{"asset":{"class":"currency","name":"USD"},"amount":"11.0000"},"net_amount":{"asset":{"class":"currency","name":"USD"},"amount":"10.7500"},"withdrawal_fee_token":"FEE-TOKEN"}`
		case fundingWithdrawalsPath:
			if request.Method != http.MethodPost || request.Header.Get("Content-Type") != "application/json" {
				t.Errorf("funding withdrawal request = %s, Content-Type %q", request.Method, request.Header.Get("Content-Type"))
			}
			var payload struct {
				Scope struct {
					MethodID string `json:"method_id"`
				} `json:"scope"`
				AddressID string `json:"address_id"`
				Amount    struct {
					AssetAmount FundingAssetAmount `json:"asset_amount"`
				} `json:"amount"`
				Fee struct {
					QuotedFee struct {
						Token string `json:"token"`
					} `json:"quoted_fee"`
					FeeIncluded bool `json:"fee_included"`
				} `json:"fee"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				return nil, err
			}
			if payload.Scope.MethodID != methodID || payload.AddressID != addressID || payload.Amount.AssetAmount.Asset.Class != "currency" || payload.Amount.AssetAmount.Asset.Name != "USD" || payload.Amount.AssetAmount.Amount != "11.0000" || payload.Fee.QuotedFee.Token != "FEE-TOKEN" || !payload.Fee.FeeIncluded {
				t.Errorf("funding withdrawal payload = %#v", payload)
			}
			responseBody = `{"withdrawal_id":"FTVZI01-1234567890123456789012","net_amount":{"asset_amount":{"asset":{"class":"currency","name":"USD"},"amount":"10.7500"}},"gross_amount":{"asset_amount":{"asset":{"class":"currency","name":"USD"},"amount":"11.0000"}},"fee":{"asset_amount":{"asset":{"class":"currency","name":"USD"},"amount":"0.2500"}}}`
		default:
			return testResponse(request, http.StatusNotFound, "not found"), nil
		}
		return testResponse(request, http.StatusOK, responseBody), nil
	})

	client, err := NewClient(ClientConfig{
		APIKey:     "public",
		Secret:     base64.StdEncoding.EncodeToString(secret),
		BaseURL:    "https://example.test",
		HTTPClient: &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatal(err)
	}
	client.now = func() time.Time { return time.UnixMilli(1_700_000_000_000) }

	balance, err := client.AssetBalance(context.Background(), "USDC")
	if err != nil || balance != "12.50000000" {
		t.Fatalf("AssetBalance() = %q, %v", balance, err)
	}
	order, err := client.SellUSDCToUSD(context.Background(), balance)
	if err != nil || len(order.TransactionIDs) != 1 {
		t.Fatalf("SellUSDCToUSD() = %#v, %v", order, err)
	}
	quote, err := client.QuoteFundingWithdrawalFee(context.Background(), methodID, "11.0000")
	if err != nil || quote.WithdrawalFeeToken != "FEE-TOKEN" || quote.NetAmount.Amount != "10.7500" {
		t.Fatalf("QuoteFundingWithdrawalFee() = %#v, %v", quote, err)
	}
	withdrawal, err := client.CreateFundingWithdrawal(context.Background(), FundingWithdrawalParams{
		MethodID:  methodID,
		AddressID: addressID,
		Asset:     "USD",
		Amount:    "11.0000",
		FeeToken:  quote.WithdrawalFeeToken,
	})
	if err != nil || withdrawal.WithdrawalID != "FTVZI01-1234567890123456789012" {
		t.Fatalf("CreateFundingWithdrawal() = %#v, %v", withdrawal, err)
	}
	if requestNumber != 4 {
		t.Fatalf("request count = %d, want 4", requestNumber)
	}
}

func TestListFundingMethodsAndAddressesFollowsPagination(t *testing.T) {
	t.Parallel()

	methodID := "d4ec4d52-b159-428e-ba64-f45455a978a1"
	requestNumber := 0
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestNumber++
		query := request.URL.Query()
		var responseBody string
		switch {
		case request.URL.Path == fundingMethodsBasePath+"withdraw" && query.Get("cursor") == "":
			if query.Get("limit") != "500" || len(query) != 1 {
				t.Errorf("first methods query = %v", query)
			}
			responseBody = `{"methods":[{"asset":{"class":"currency","name":"USD"},"method_id":"d4ec4d52-b159-428e-ba64-f45455a978a1","method_name":"ACH","minimum_amount":"1.00"}],"next_cursor":"methods-next"}`
		case request.URL.Path == fundingMethodsBasePath+"withdraw" && query.Get("cursor") == "methods-next":
			if len(query) != 1 {
				t.Errorf("next methods query = %v", query)
			}
			responseBody = `{"methods":[{"asset":{"class":"currency","name":"USD"},"method_id":"72ff4244-0c51-4e42-bf8d-3cf6900c61e1","method_name":"Wire","minimum_amount":"20.00"}]}`
		case request.URL.Path == fundingAddressesPath && query.Get("cursor") == "":
			if query.Get("limit") != "500" || query.Get("scope[method_id]") != methodID || len(query) != 2 {
				t.Errorf("first addresses query = %v", query)
			}
			responseBody = `{"addresses":[{"address_id":"ABR6SXP-SF6CY-VJMONY","scope":{"method_id":"d4ec4d52-b159-428e-ba64-f45455a978a1"},"name":"Example Bank","verified":true}],"next_cursor":"addresses-next"}`
		case request.URL.Path == fundingAddressesPath && query.Get("cursor") == "addresses-next":
			if len(query) != 1 {
				t.Errorf("next addresses query = %v", query)
			}
			responseBody = `{"addresses":[]}`
		default:
			return testResponse(request, http.StatusNotFound, "not found"), nil
		}
		return testResponse(request, http.StatusOK, responseBody), nil
	})

	client, err := NewClient(ClientConfig{
		APIKey:     "public",
		Secret:     base64.StdEncoding.EncodeToString([]byte("secret")),
		BaseURL:    "https://example.test",
		HTTPClient: &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatal(err)
	}

	methods, err := client.ListFundingMethods(context.Background(), "withdraw")
	if err != nil || len(methods) != 2 || methods[0].MethodName != "ACH" || methods[1].MethodName != "Wire" {
		t.Fatalf("ListFundingMethods() = %#v, %v", methods, err)
	}
	addresses, err := client.ListFundingAddresses(context.Background(), methodID)
	if err != nil || len(addresses) != 1 || addresses[0].AddressID != "ABR6SXP-SF6CY-VJMONY" || !addresses[0].Verified {
		t.Fatalf("ListFundingAddresses() = %#v, %v", addresses, err)
	}
	if requestNumber != 4 {
		t.Fatalf("request count = %d, want 4", requestNumber)
	}
}

func TestPrivateRequestReturnsKrakenError(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return testResponse(request, http.StatusOK, `{"error":["EAPI:Invalid key"]}`), nil
	})
	client, err := NewClient(ClientConfig{
		APIKey:     "public",
		Secret:     base64.StdEncoding.EncodeToString([]byte("secret")),
		BaseURL:    "https://example.test",
		HTTPClient: &http.Client{Transport: transport},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Balance(context.Background())
	if err == nil || !strings.Contains(err.Error(), "EAPI:Invalid key") {
		t.Fatalf("Balance() error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func testResponse(request *http.Request, statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}
}
