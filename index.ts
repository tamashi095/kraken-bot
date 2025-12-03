import { createHash, createHmac } from "crypto";
import { stringify } from "querystring";

const KRAKEN_PUBLIC_KEY = Bun.env.KRAKEN_PUBLIC_KEY;
const KRAKEN_SECRET_KEY = Bun.env.KRAKEN_SECRET_KEY;

if (!KRAKEN_PUBLIC_KEY) throw new Error("KRAKEN_PUBLIC_KEY is not set");
if (!KRAKEN_SECRET_KEY) throw new Error("KRAKEN_SECRET_KEY is not set");

interface KrakenRequestData {
  nonce: number | string;
  [key: string]: unknown;
}

type KrakenDataInput = string | KrakenRequestData;

/**
 * Generates a Kraken API signature for authenticated requests.
 *
 * @param urlPath - The API endpoint path (e.g., "/0/private/Balance")
 * @param data - Request data as either a JSON string or an object with a nonce property
 * @param secret - Base64-encoded API secret key
 * @returns Base64-encoded HMAC-SHA512 signature
 * @throws {Error} If data type is invalid or nonce is missing
 */
export function getKrakenSignature(
  urlPath: string,
  data: KrakenDataInput,
  secret: string
): string {
  let encoded: string;

  if (typeof data === "string") {
    const jsonData = JSON.parse(data) as KrakenRequestData;

    if (jsonData.nonce === undefined) {
      throw new Error("Nonce is required in data");
    }

    encoded = String(jsonData.nonce) + data;
  } else if (typeof data === "object" && data !== null) {
    if (data.nonce === undefined) {
      throw new Error("Nonce is required in data object");
    }

    const dataStr = stringify(data as Record<string, string | number>);
    encoded = String(data.nonce) + dataStr;
  } else {
    throw new Error("Invalid data type: expected string or object");
  }

  const sha256Hash = createHash("sha256").update(encoded).digest();
  const message = urlPath + sha256Hash.toString("binary");
  const secretBuffer = Buffer.from(secret, "base64");
  const hmac = createHmac("sha512", secretBuffer);
  hmac.update(message, "binary");

  return hmac.digest("base64");
}

/**
 * Kraken API client that handles authentication, nonce generation, and request signing.
 */
export class KrakenClient {
  private readonly apiKey: string;
  private readonly secret: string;
  private readonly baseUrl: string;
  private nonceCounter: number;

  /**
   * Creates a new KrakenClient instance.
   *
   * @param apiKey - Public API key from Kraken API key-pair
   * @param secret - Base64-encoded secret key from Kraken API key-pair
   * @param baseUrl - Kraken API base URL (defaults to https://api.kraken.com)
   */
  constructor(
    apiKey: string,
    secret: string,
    baseUrl: string = "https://api.kraken.com"
  ) {
    if (!apiKey) throw new Error("API key is required");
    if (!secret) throw new Error("Secret key is required");

    this.apiKey = apiKey;
    this.secret = secret;
    this.baseUrl = baseUrl;
    // Initialize nonce with current timestamp in milliseconds
    this.nonceCounter = Date.now();
  }

  /**
   * Generates a unique, always-increasing nonce value.
   * Uses timestamp + counter to ensure uniqueness even for rapid requests.
   *
   * @returns Unsigned 64-bit integer nonce
   */
  private generateNonce(): number {
    // Use timestamp in milliseconds and increment counter
    // This ensures nonce is always increasing even for rapid requests
    const timestamp = Date.now();
    if (timestamp > this.nonceCounter) {
      this.nonceCounter = timestamp;
    } else {
      this.nonceCounter += 1;
    }
    return this.nonceCounter;
  }

  /**
   * Makes an authenticated request to the Kraken API.
   *
   * @param endpoint - API endpoint path (e.g., "/0/private/Balance")
   * @param options - Request options including method, body, and optional OTP
   * @returns Promise resolving to the API response
   */
  async request<T = unknown>(
    endpoint: string,
    options: {
      method?: "GET" | "POST";
      body?: Record<string, unknown>;
      otp?: string;
    } = {}
  ): Promise<T> {
    const { method = "POST", body = {}, otp } = options;

    // Generate nonce and prepare payload
    const nonce = this.generateNonce();
    const payload: KrakenRequestData = {
      ...body,
      nonce,
    };

    // Add OTP if provided
    if (otp) {
      payload.otp = otp;
    }

    // Generate signature
    const signature = getKrakenSignature(endpoint, payload, this.secret);

    // Prepare headers
    const headers: Record<string, string> = {
      "API-Key": this.apiKey,
      "API-Sign": signature,
      "Content-Type": "application/x-www-form-urlencoded",
    };

    // Prepare request body
    const formData = stringify(payload as Record<string, string | number>);

    // Make the request
    const url = `${this.baseUrl}${endpoint}`;
    const response = await fetch(url, {
      method,
      headers,
      body: method === "POST" ? formData : undefined,
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(
        `Kraken API error: ${response.status} ${response.statusText} - ${errorText}`
      );
    }

    const data = (await response.json()) as { error: string[]; result?: T };

    // Check for API-level errors
    if (data.error && data.error.length > 0) {
      throw new Error(`Kraken API error: ${data.error.join(", ")}`);
    }

    return data.result as T;
  }

  /**
   * Convenience method for public endpoints (no authentication required).
   *
   * @param endpoint - API endpoint path
   * @param params - Query parameters
   * @returns Promise resolving to the API response
   */
  async publicRequest<T = unknown>(
    endpoint: string,
    params?: Record<string, string | number>
  ): Promise<T> {
    const url = new URL(`${this.baseUrl}${endpoint}`);
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        url.searchParams.append(key, String(value));
      });
    }

    const response = await fetch(url.toString(), {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(
        `Kraken API error: ${response.status} ${response.statusText} - ${errorText}`
      );
    }

    const data = (await response.json()) as { error: string[]; result?: T };

    if (data.error && data.error.length > 0) {
      throw new Error(`Kraken API error: ${data.error.join(", ")}`);
    }

    return data.result as T;
  }
}

// Create a default client instance using environment variables
export const krakenClient = new KrakenClient(
  KRAKEN_PUBLIC_KEY,
  KRAKEN_SECRET_KEY
);

interface AddOrderParams {
  /** Order direction (buy/sell) */
  type: "buy" | "sell";
  /** The execution model of the order */
  ordertype:
    | "market"
    | "limit"
    | "iceberg"
    | "stop-loss"
    | "take-profit"
    | "stop-loss-limit"
    | "take-profit-limit"
    | "trailing-stop"
    | "trailing-stop-limit"
    | "settle-position";
  /** Asset pair id or altname */
  pair: string;
  /** Order quantity in terms of the base asset */
  volume: string;
  /** Limit price for limit orders */
  price?: string;
  /** Secondary price for stop orders */
  price2?: string;
  /** Price signal used to trigger stop orders */
  trigger?: "index" | "last";
  /** Amount of leverage desired */
  leverage?: string;
  /** If true, order will only reduce a currently open position */
  reduce_only?: boolean;
  /** Self Trade Prevention mode */
  stptype?: "cancel-newest" | "cancel-oldest" | "cancel-both";
  /** Comma delimited list of order flags */
  oflags?: string;
  /** Time-in-force of the order */
  timeinforce?: "GTC" | "IOC" | "GTD";
  /** Scheduled start time */
  starttm?: string;
  /** Expiry time */
  expiretm?: string;
  /** User reference id */
  userref?: number;
  /** Client order identifier */
  cl_ord_id?: string;
  /** Validate order only (don't execute) */
  validate?: boolean;
  /** Deadline for order execution */
  deadline?: string;
  [key: string]: unknown;
}

interface AddOrderResponse {
  descr: {
    order: string;
    close?: string;
  };
  txid?: string[];
}

/**
 * Places a market order to sell USDC to USD.
 *
 * @param volume - Amount of USDC to sell
 * @param options - Additional order options (validate, userref, etc.)
 * @returns Promise resolving to the order response with transaction IDs
 */
export async function sellUSDCToUSD(
  volume: string,
  options: { validate?: boolean; userref?: number } = {}
): Promise<AddOrderResponse> {
  const orderParams: AddOrderParams = {
    type: "sell",
    ordertype: "market",
    pair: "USDCUSD",
    volume,
    ...options,
  };

  return await krakenClient.request<AddOrderResponse>("/0/private/AddOrder", {
    body: orderParams,
  });
}

interface WithdrawMethod {
  asset: string;
  method: string;
  network?: string;
  minimum?: string;
  [key: string]: unknown;
}

/**
 * Gets available withdrawal methods for an asset.
 *
 * @param asset - Asset to get withdrawal methods for (optional, gets all if not specified)
 * @returns Promise resolving to withdrawal methods
 */
export async function getWithdrawMethods(
  asset?: string
): Promise<WithdrawMethod[]> {
  const params = asset ? { asset } : {};
  return await krakenClient.request<WithdrawMethod[]>(
    "/0/private/WithdrawMethods",
    { body: params }
  );
}

interface WithdrawParams {
  /** Asset being withdrawn */
  asset: string;
  /** Withdrawal key name, as set up on your account */
  key: string;
  /** Amount to be withdrawn */
  amount: string;
  /** Crypto address that can be used to confirm address matches key */
  address?: string;
  /** Asset class of the asset being withdrawn */
  aclass?: "currency" | "tokenized_asset";
  /** If the processed withdrawal fee is higher than max_fee, withdrawal will fail */
  max_fee?: string;
  /** Optional parameter for viewing xstocks data */
  rebase_multiplier?: "rebased" | "base";
  [key: string]: unknown;
}

interface WithdrawResponse {
  refid: string;
}

/**
 * Withdraws funds from Kraken to a specified withdrawal key.
 *
 * @param asset - Asset to withdraw (e.g., "USD", "USDC", "XBT")
 * @param key - Withdrawal key name configured in your Kraken account
 * @param amount - Amount to withdraw
 * @param options - Additional withdrawal options (address, max_fee, etc.)
 * @returns Promise resolving to withdrawal reference ID
 */
export async function withdraw(
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

  return await krakenClient.request<WithdrawResponse>("/0/private/Withdraw", {
    body: withdrawParams,
  });
}

/**
 * Withdraws USD to Mercury.
 *
 * @param amount - Amount of USD to withdraw
 * @param options - Additional withdrawal options
 * @returns Promise resolving to withdrawal reference ID
 */
export async function withdrawToMercury(
  amount: string,
  options: Omit<WithdrawParams, "asset" | "key" | "amount"> = {}
): Promise<WithdrawResponse> {
  return await withdraw("USD", "Mercury", amount, options);
}

/**
 * Parses a decimal string to BigInt by converting to base units.
 *
 * @param value - Decimal string (e.g., "366.14886400")
 * @param decimals - Number of decimal places (default: 8 for USDC)
 * @returns BigInt representation in base units
 *
 * @example
 * parseDecimalToBigInt("366.14886400", 8) // 36614886400n
 * parseDecimalToBigInt("0.00000000", 8)   // 0n
 */
function parseDecimalToBigInt(value: string, decimals: number = 8): bigint {
  const [whole = "0", fraction = ""] = value.split(".");
  const paddedFraction = fraction.padEnd(decimals, "0").slice(0, decimals);
  return BigInt(whole + paddedFraction);
}

async function main() {
  try {
    const usdcBalanceResponse = await krakenClient.request<
      Record<string, string>
    >("/0/private/Balance");
    const usdcBalance = usdcBalanceResponse.USDC ?? "0.00000000";
    const usdcBalanceBigInt = parseDecimalToBigInt(usdcBalance, 8);

    // Sell USDC to USD if balance is greater than 0
    if (usdcBalanceBigInt > 0n) {
      console.log(`\nSelling ${usdcBalance} USDC...`);
      const orderResult = await sellUSDCToUSD(usdcBalance);
      console.log("Order result:", orderResult);
    } else {
      console.log("No USDC balance to sell.");
    }

    const usdBalanceResponse = await krakenClient.request<
      Record<string, string>
    >("/0/private/Balance");
    const usdBalance = usdBalanceResponse.ZUSD ?? "0.00000000";
    const usdBalanceBigInt = parseDecimalToBigInt(usdBalance, 4);

    // Withdraw all USD to Mercury
    // Only withdraw if USD balance is at least $10
    if (usdBalanceBigInt >= 100000n) {
      console.log(`Withdrawing ${usdBalance} USD to Mercury...`);
      const withdrawResult = await withdrawToMercury(usdBalance);
      console.log("Withdrawal reference ID:", withdrawResult.refid);
    } else {
      console.log("No USD balance to withdraw.");
    }
  } catch (error) {
    console.error("Error:", error);
  }
}

main();
