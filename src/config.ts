/**
 * Configuration Management
 * 
 * This module handles application configuration and environment variables.
 */

import type { KrakenClientConfig } from "./types/kraken.ts";

/**
 * Application configuration interface
 */
export interface AppConfig {
  kraken: KrakenClientConfig;
  minimumWithdrawalUSD: number;
  usdWithdrawalKey: string;
}

/**
 * Loads and validates configuration from environment variables.
 * 
 * @throws {Error} If required environment variables are missing
 * @returns Application configuration object
 */
export function loadConfig(): AppConfig {
  const apiKey = Bun.env.KRAKEN_PUBLIC_KEY;
  const secret = Bun.env.KRAKEN_SECRET_KEY;

  if (!apiKey) {
    throw new Error(
      "KRAKEN_PUBLIC_KEY environment variable is not set. " +
      "Please set it in your .env file or environment."
    );
  }

  if (!secret) {
    throw new Error(
      "KRAKEN_SECRET_KEY environment variable is not set. " +
      "Please set it in your .env file or environment."
    );
  }

  const usdWithdrawalKey = Bun.env.USD_WITHDRAWAL_KEY;
  if (!usdWithdrawalKey) {
    throw new Error(
      "USD_WITHDRAWAL_KEY environment variable is not set. " +
      "Please set it in your .env file or environment."
    );
  }

  return {
    kraken: {
      apiKey,
      secret,
      baseUrl: Bun.env.KRAKEN_API_URL || "https://api.kraken.com",
    },
    minimumWithdrawalUSD: Number(Bun.env.MINIMUM_WITHDRAWAL_USD || "10"),
    usdWithdrawalKey,
  };
}

/**
 * Validates the configuration object.
 * 
 * @param config - Configuration object to validate
 * @throws {Error} If configuration is invalid
 */
export function validateConfig(config: AppConfig): void {
  if (!config.kraken.apiKey || !config.kraken.secret) {
    throw new Error("Kraken API credentials are required");
  }

  if (config.minimumWithdrawalUSD < 0) {
    throw new Error("Minimum withdrawal amount must be positive");
  }

  if (!config.usdWithdrawalKey) {
    throw new Error("USD withdrawal key is required");
  }
}

