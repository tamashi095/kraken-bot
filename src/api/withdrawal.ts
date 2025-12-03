/**
 * Withdrawal API
 * 
 * Functions for withdrawing assets from Kraken to external wallets or payment methods.
 */

import type { KrakenClient } from "../client/KrakenClient.ts";
import type {
  WithdrawMethod,
  WithdrawParams,
  WithdrawResponse,
} from "../types/kraken.ts";

/**
 * Gets available withdrawal methods for an asset.
 *
 * @param client - KrakenClient instance
 * @param asset - Asset to get withdrawal methods for (optional, gets all if not specified)
 * @returns Promise resolving to array of withdrawal methods
 *
 * @example
 * ```typescript
 * // Get all withdrawal methods
 * const allMethods = await getWithdrawalMethods(client);
 * 
 * // Get withdrawal methods for a specific asset
 * const usdcMethods = await getWithdrawalMethods(client, "USDC");
 * ```
 */
export async function getWithdrawalMethods(
  client: KrakenClient,
  asset?: string
): Promise<WithdrawMethod[]> {
  const params = asset ? { asset } : {};
  return await client.request<WithdrawMethod[]>(
    "/0/private/WithdrawMethods",
    { body: params }
  );
}

/**
 * Withdraws funds from Kraken to a specified withdrawal key.
 *
 * @param client - KrakenClient instance
 * @param asset - Asset to withdraw (e.g., "USD", "USDC", "XBT")
 * @param key - Withdrawal key name configured in your Kraken account
 * @param amount - Amount to withdraw
 * @param options - Additional withdrawal options (address, max_fee, etc.)
 * @returns Promise resolving to withdrawal reference ID
 *
 * @example
 * ```typescript
 * const result = await withdraw(client, "USD", "MyBank", "1000.00");
 * console.log("Withdrawal reference:", result.refid);
 * ```
 */
export async function withdraw(
  client: KrakenClient,
  asset: string,
  key: string,
  amount: string,
  options: Omit<WithdrawParams, "asset" | "key" | "amount"> = {}
): Promise<WithdrawResponse> {
  const withdrawParams: WithdrawParams = {
    asset,
    key,
    amount,
    ...options,
  };

  return await client.request<WithdrawResponse>("/0/private/Withdraw", {
    body: withdrawParams,
  });
}

/**
 * Withdraws USD to a Mercury account.
 *
 * @param client - KrakenClient instance
 * @param amount - Amount of USD to withdraw
 * @param options - Additional withdrawal options
 * @returns Promise resolving to withdrawal reference ID
 *
 * @example
 * ```typescript
 * const result = await withdrawToMercury(client, "1000.00");
 * console.log("Withdrawal initiated:", result.refid);
 * ```
 */
export async function withdrawToMercury(
  client: KrakenClient,
  amount: string,
  options: Omit<WithdrawParams, "asset" | "key" | "amount"> = {}
): Promise<WithdrawResponse> {
  return await withdraw(client, "USD", "Mercury", amount, options);
}

