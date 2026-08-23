// Package config loads and validates the bot's environment configuration.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"kraken-bot/internal/decimal"
)

const (
	defaultBaseURL           = "https://api.kraken.com"
	defaultMinimumWithdrawal = "10"
)

var (
	fundingMethodIDPattern  = regexp.MustCompile(`(?i)^[0-9a-f]{8}-?(?:[0-9a-f]{4}-?){3}[0-9a-f]{12}$`)
	fundingAddressIDPattern = regexp.MustCompile(`^AB[a-zA-Z0-9]{5}-[a-zA-Z0-9]{5}-[a-zA-Z0-9]{6}$`)
)

// Config contains validated runtime configuration.
type Config struct {
	APIKey                   string
	Secret                   string
	BaseURL                  string
	MinimumWithdrawal        *big.Int
	MinimumWithdrawalDisplay string
	USDFundingMethodID       string
	USDFundingAddressID      string
	SettlementDelay          time.Duration
}

// Load reads dotenvPath without overriding existing environment variables,
// then validates and returns the bot configuration.
func Load(dotenvPath string) (Config, error) {
	if dotenvPath != "" {
		if err := loadDotEnv(dotenvPath); err != nil {
			return Config{}, fmt.Errorf("load %s: %w", dotenvPath, err)
		}
	}

	apiKey, err := required("KRAKEN_PUBLIC_KEY")
	if err != nil {
		return Config{}, err
	}
	secret, err := required("KRAKEN_SECRET_KEY")
	if err != nil {
		return Config{}, err
	}
	fundingMethodID, err := required("USD_FUNDING_METHOD_ID")
	if err != nil {
		if os.Getenv("USD_WITHDRAWAL_KEY") != "" {
			return Config{}, fmt.Errorf("USD_WITHDRAWAL_KEY is only supported by Kraken's legacy funding API; set USD_FUNDING_METHOD_ID and USD_FUNDING_ADDRESS_ID")
		}
		return Config{}, err
	}
	if !fundingMethodIDPattern.MatchString(fundingMethodID) {
		return Config{}, fmt.Errorf("USD_FUNDING_METHOD_ID is not a valid Kraken funding method ID")
	}
	fundingAddressID, err := required("USD_FUNDING_ADDRESS_ID")
	if err != nil {
		return Config{}, err
	}
	if !fundingAddressIDPattern.MatchString(fundingAddressID) {
		return Config{}, fmt.Errorf("USD_FUNDING_ADDRESS_ID is not a valid Kraken funding address ID")
	}

	baseURL := valueOrDefault("KRAKEN_API_URL", defaultBaseURL)
	parsedURL, err := url.Parse(baseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return Config{}, fmt.Errorf("KRAKEN_API_URL must be an absolute HTTP(S) URL")
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return Config{}, fmt.Errorf("KRAKEN_API_URL must use http or https")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	minimumDisplay := valueOrDefault("MINIMUM_WITHDRAWAL_USD", defaultMinimumWithdrawal)
	minimum, err := decimal.ParseUnits(minimumDisplay, 4)
	if err != nil {
		return Config{}, fmt.Errorf("invalid MINIMUM_WITHDRAWAL_USD: %w", err)
	}
	if minimum.Sign() < 0 {
		return Config{}, fmt.Errorf("MINIMUM_WITHDRAWAL_USD must not be negative")
	}

	return Config{
		APIKey:                   apiKey,
		Secret:                   secret,
		BaseURL:                  baseURL,
		MinimumWithdrawal:        minimum,
		MinimumWithdrawalDisplay: minimumDisplay,
		USDFundingMethodID:       fundingMethodID,
		USDFundingAddressID:      fundingAddressID,
		SettlementDelay:          2 * time.Second,
	}, nil
}

func required(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s environment variable is not set", name)
	}
	return value, nil
}

func valueOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		name, rawValue, ok := strings.Cut(line, "=")
		name = strings.TrimSpace(name)
		if !ok || !validName(name) {
			return fmt.Errorf("line %d is not a valid KEY=value assignment", lineNumber)
		}
		value, err := parseDotEnvValue(strings.TrimSpace(rawValue))
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNumber, err)
		}
		if _, exists := os.LookupEnv(name); !exists {
			if err := os.Setenv(name, value); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func validName(name string) bool {
	if name == "" || !isNameStart(name[0]) {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !isNameStart(name[i]) && (name[i] < '0' || name[i] > '9') {
			return false
		}
	}
	return true
}

func isNameStart(character byte) bool {
	return character == '_' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z'
}

func parseDotEnvValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if value[0] == '\'' {
		if len(value) < 2 || value[len(value)-1] != '\'' {
			return "", fmt.Errorf("unterminated single-quoted value")
		}
		return value[1 : len(value)-1], nil
	}
	if value[0] == '"' {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("invalid double-quoted value: %w", err)
		}
		return unquoted, nil
	}
	if index := strings.Index(value, " #"); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value), nil
}
