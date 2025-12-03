/**
 * Trading API
 * 
 * Functions for placing and managing orders on Kraken.
 */

import type { KrakenClient } from "../client/KrakenClient.ts";
import type { AddOrderParams, AddOrderResponse } from "../types/kraken.ts";

/**
 * Places an order on the Kraken exchange.
 *
 * @param client - KrakenClient instance
 * @param params - Order parameters
 * @returns Promise resolving to order response with transaction IDs
 *
 * @example
 * ```typescript
 * const order = await placeOrder(client, {
 *   type: "buy",
 *   ordertype: "market",
 *   pair: "XBTUSD",
 *   volume: "0.01"
 * });
 * ```
 */
export async function placeOrder(
  client: KrakenClient,
  params: AddOrderParams
): Promise<AddOrderResponse> {
  return await client.request<AddOrderResponse>("/0/private/AddOrder", {
    body: params,
  });
}

/**
 * Places a market order to sell USDC for USD.
 *
 * @param client - KrakenClient instance
 * @param volume - Amount of USDC to sell
 * @param options - Additional order options (validate, userref, etc.)
 * @returns Promise resolving to order response with transaction IDs
 *
 * @example
 * ```typescript
 * // Sell 100 USDC for USD
 * const result = await sellUSDCToUSD(client, "100.00");
 * console.log("Transaction IDs:", result.txid);
 * 
 * // Validate order without executing
 * const validation = await sellUSDCToUSD(client, "100.00", { validate: true });
 * ```
 */
export async function sellUSDCToUSD(
  client: KrakenClient,
  volume: string,
  options: { validate?: boolean; userref?: number } = {}
): Promise<AddOrderResponse> {
  return await placeOrder(client, {
    type: "sell",
    ordertype: "market",
    pair: "USDCUSD",
    volume,
    ...options,
  });
}

/**
 * Places a market order to buy USDC with USD.
 *
 * @param client - KrakenClient instance
 * @param volume - Amount of USDC to buy
 * @param options - Additional order options (validate, userref, etc.)
 * @returns Promise resolving to order response with transaction IDs
 */
export async function buyUSDCWithUSD(
  client: KrakenClient,
  volume: string,
  options: { validate?: boolean; userref?: number } = {}
): Promise<AddOrderResponse> {
  return await placeOrder(client, {
    type: "buy",
    ordertype: "market",
    pair: "USDCUSD",
    volume,
    ...options,
  });
}

