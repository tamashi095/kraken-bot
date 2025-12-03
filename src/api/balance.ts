/**
 * Balance API
 * 
 * Functions for retrieving account balance information from Kraken.
 */

import type { KrakenClient } from "../client/KrakenClient.ts";
import type { BalanceResponse } from "../types/kraken.ts";

/**
 * Retrieves the account balance for all assets.
 *
 * @param client - KrakenClient instance
 * @returns Promise resolving to balance data for all assets
 *
 * @example
 * ```typescript
 * const balance = await getBalance(client);
 * console.log("USDC:", balance.USDC);
 * console.log("USD:", balance.ZUSD);
 * ```
 */
export async function getBalance(
  client: KrakenClient
): Promise<BalanceResponse> {
  return await client.request<BalanceResponse>("/0/private/Balance");
}

/**
 * Gets the balance for a specific asset.
 *
 * @param client - KrakenClient instance
 * @param asset - Asset symbol (e.g., "USDC", "ZUSD", "XXBT")
 * @returns Balance string or "0" if not found
 *
 * @example
 * ```typescript
 * const usdcBalance = await getAssetBalance(client, "USDC");
 * console.log("USDC Balance:", usdcBalance);
 * ```
 */
export async function getAssetBalance(
  client: KrakenClient,
  asset: string
): Promise<string> {
  const balance = await getBalance(client);
  return balance[asset] ?? "0.00000000";
}

