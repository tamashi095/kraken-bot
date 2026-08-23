package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFromDotEnv(t *testing.T) {
	clearConfigEnvironment(t)

	path := filepath.Join(t.TempDir(), ".env")
	contents := "KRAKEN_PUBLIC_KEY=public\nKRAKEN_SECRET_KEY='c2VjcmV0'\nUSD_FUNDING_METHOD_ID=d4ec4d52-b159-428e-ba64-f45455a978a1\nUSD_FUNDING_ADDRESS_ID=ABR6SXP-SF6CY-VJMONY\nMINIMUM_WITHDRAWAL_USD=12.3456\nKRAKEN_API_URL=https://example.test/\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIKey != "public" || cfg.Secret != "c2VjcmV0" {
		t.Fatalf("Load() credentials = %#v", cfg)
	}
	if cfg.USDFundingMethodID != "d4ec4d52-b159-428e-ba64-f45455a978a1" || cfg.USDFundingAddressID != "ABR6SXP-SF6CY-VJMONY" {
		t.Fatalf("Load() funding IDs = %#v", cfg)
	}
	if cfg.BaseURL != "https://example.test" {
		t.Errorf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.MinimumWithdrawal.String() != "123456" {
		t.Errorf("MinimumWithdrawal = %s", cfg.MinimumWithdrawal)
	}
	if cfg.SettlementDelay != 2*time.Second {
		t.Errorf("SettlementDelay = %s", cfg.SettlementDelay)
	}
}

func TestEnvironmentTakesPrecedenceOverDotEnv(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("KRAKEN_PUBLIC_KEY", "from-environment")
	t.Setenv("KRAKEN_SECRET_KEY", "c2VjcmV0")
	t.Setenv("USD_FUNDING_METHOD_ID", "d4ec4d52-b159-428e-ba64-f45455a978a1")
	t.Setenv("USD_FUNDING_ADDRESS_ID", "ABR6SXP-SF6CY-VJMONY")

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("KRAKEN_PUBLIC_KEY=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIKey != "from-environment" {
		t.Errorf("APIKey = %q", cfg.APIKey)
	}
}

func TestLoadKrakenDoesNotRequireFundingIDs(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("KRAKEN_PUBLIC_KEY", "public")
	t.Setenv("KRAKEN_SECRET_KEY", "c2VjcmV0")
	t.Setenv("KRAKEN_API_URL", "https://example.test/")

	cfg, err := LoadKraken("")
	if err != nil {
		t.Fatalf("LoadKraken() error = %v", err)
	}
	if cfg.APIKey != "public" || cfg.Secret != "c2VjcmV0" || cfg.BaseURL != "https://example.test" {
		t.Fatalf("LoadKraken() = %#v", cfg)
	}
}

func TestLoadRejectsInvalidMinimum(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("KRAKEN_PUBLIC_KEY", "public")
	t.Setenv("KRAKEN_SECRET_KEY", "c2VjcmV0")
	t.Setenv("USD_FUNDING_METHOD_ID", "d4ec4d52-b159-428e-ba64-f45455a978a1")
	t.Setenv("USD_FUNDING_ADDRESS_ID", "ABR6SXP-SF6CY-VJMONY")
	t.Setenv("MINIMUM_WITHDRAWAL_USD", "10.00001")

	if _, err := Load(""); err == nil {
		t.Fatal("Load() unexpectedly succeeded")
	}
}

func clearConfigEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"KRAKEN_PUBLIC_KEY",
		"KRAKEN_SECRET_KEY",
		"USD_WITHDRAWAL_KEY",
		"USD_FUNDING_METHOD_ID",
		"USD_FUNDING_ADDRESS_ID",
		"MINIMUM_WITHDRAWAL_USD",
		"KRAKEN_API_URL",
	} {
		value, exists := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if exists {
				_ = os.Setenv(name, value)
			} else {
				_ = os.Unsetenv(name)
			}
		})
	}
}
