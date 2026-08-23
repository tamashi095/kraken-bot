// Package kraken implements the authenticated Kraken Spot REST operations used by the bot.
package kraken

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultBaseURL = "https://api.kraken.com"

// ClientConfig configures a Client.
type ClientConfig struct {
	APIKey     string
	Secret     string
	BaseURL    string
	HTTPClient *http.Client
}

// Client is safe for concurrent use. A client must not share its API key with
// another process unless Kraken's nonce window is configured appropriately.
type Client struct {
	apiKey     string
	secret     []byte
	baseURL    string
	httpClient *http.Client

	nonceMu sync.Mutex
	nonce   uint64
	now     func() time.Time
}

// NewClient validates cfg and creates a Kraken client.
func NewClient(cfg ClientConfig) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if strings.TrimSpace(cfg.Secret) == "" {
		return nil, fmt.Errorf("secret key is required")
	}
	secret, err := base64.StdEncoding.DecodeString(cfg.Secret)
	if err != nil {
		return nil, fmt.Errorf("secret key is not valid base64: %w", err)
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("base URL must be an absolute URL")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	now := time.Now
	return &Client{
		apiKey:     cfg.APIKey,
		secret:     secret,
		baseURL:    baseURL,
		httpClient: httpClient,
		nonce:      uint64(now().UnixMilli()),
		now:        now,
	}, nil
}

func (client *Client) nextNonce() uint64 {
	client.nonceMu.Lock()
	defer client.nonceMu.Unlock()

	now := uint64(client.now().UnixMilli())
	if now > client.nonce {
		client.nonce = now
	} else {
		client.nonce++
	}
	return client.nonce
}

func (client *Client) privateRequest(ctx context.Context, path string, values url.Values, result any) error {
	form := cloneValues(values)
	nonce := strconv.FormatUint(client.nextNonce(), 10)
	form.Set("nonce", nonce)
	payload := form.Encode()
	signature := generateSignature(path, nonce, payload, client.secret)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+path, strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create Kraken request: %w", err)
	}
	request.Header.Set("API-Key", client.apiKey)
	request.Header.Set("API-Sign", signature)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send Kraken request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return fmt.Errorf("read Kraken response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Kraken API HTTP error: %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	var envelope struct {
		Errors []string        `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decode Kraken response: %w", err)
	}
	if len(envelope.Errors) > 0 {
		return fmt.Errorf("Kraken API error: %s", strings.Join(envelope.Errors, ", "))
	}
	if result == nil {
		return nil
	}
	if len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return fmt.Errorf("Kraken API response did not contain a result")
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("decode Kraken result: %w", err)
	}
	return nil
}

func (client *Client) fundingRequest(ctx context.Context, method, path string, query url.Values, requestBody, result any) error {
	var payload []byte
	var err error
	if requestBody != nil {
		payload, err = json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode Kraken Funding request: %w", err)
		}
	}

	signedPath := path
	if encodedQuery := query.Encode(); encodedQuery != "" {
		signedPath += "?" + encodedQuery
	}
	nonce := strconv.FormatUint(client.nextNonce(), 10)
	signature := generateFundingSignature(signedPath, nonce, payload, client.secret)

	var body io.Reader
	if requestBody != nil {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+signedPath, body)
	if err != nil {
		return fmt.Errorf("create Kraken Funding request: %w", err)
	}
	request.Header.Set("API-Key", client.apiKey)
	request.Header.Set("API-Sign", signature)
	request.Header.Set("API-Nonce", nonce)
	request.Header.Set("Accept", "application/json")
	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send Kraken Funding request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return fmt.Errorf("read Kraken Funding response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Kraken Funding API HTTP error: %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}
	if result == nil {
		return nil
	}
	if len(responseBody) == 0 {
		return fmt.Errorf("Kraken Funding API response was empty")
	}
	if err := json.Unmarshal(responseBody, result); err != nil {
		return fmt.Errorf("decode Kraken Funding response: %w", err)
	}
	return nil
}

func cloneValues(values url.Values) url.Values {
	clone := make(url.Values, len(values)+1)
	for key, entries := range values {
		clone[key] = append([]string(nil), entries...)
	}
	return clone
}

func generateSignature(path, nonce, payload string, secret []byte) string {
	hash := sha256.Sum256([]byte(nonce + payload))
	message := append([]byte(path), hash[:]...)
	mac := hmac.New(sha512.New, secret)
	_, _ = mac.Write(message)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func generateFundingSignature(signedPath, nonce string, payload, secret []byte) string {
	hashInput := append([]byte(nonce), payload...)
	hash := sha256.Sum256(hashInput)
	message := append([]byte(signedPath), hash[:]...)
	mac := hmac.New(sha512.New, secret)
	_, _ = mac.Write(message)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
