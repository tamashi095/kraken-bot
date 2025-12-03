/**
 * Kraken API Client
 * 
 * This module provides the main KrakenClient class for interacting with the Kraken API.
 */

import { stringify } from "querystring";
import { generateKrakenSignature } from "../utils/crypto.ts";
import type {
  KrakenRequestData,
  KrakenClientConfig,
  RequestOptions,
} from "../types/kraken.ts";

/**
 * Main Kraken API client that handles authentication, nonce generation, and request signing.
 * 
 * @example
 * ```typescript
 * const client = new KrakenClient({
 *   apiKey: "your-api-key",
 *   secret: "your-secret-key"
 * });
 * 
 * const balance = await client.request("/0/private/Balance");
 * ```
 */
export class KrakenClient {
  private readonly apiKey: string;
  private readonly secret: string;
  private readonly baseUrl: string;
  private nonceCounter: number;

  /**
   * Creates a new KrakenClient instance.
   *
   * @param config - Configuration object with API credentials
   */
  constructor(config: KrakenClientConfig) {
    const { apiKey, secret, baseUrl = "https://api.kraken.com" } = config;

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
    const timestamp = Date.now();
    if (timestamp > this.nonceCounter) {
      this.nonceCounter = timestamp;
    } else {
      this.nonceCounter += 1;
    }
    return this.nonceCounter;
  }

  /**
   * Makes an authenticated request to a private Kraken API endpoint.
   *
   * @param endpoint - API endpoint path (e.g., "/0/private/Balance")
   * @param options - Request options including method, body, and optional OTP
   * @returns Promise resolving to the API response
   * @throws {Error} If the request fails or API returns an error
   */
  async request<T = unknown>(
    endpoint: string,
    options: RequestOptions = {}
  ): Promise<T> {
    const { method = "POST", body = {}, otp } = options;

    // Generate nonce and prepare payload
    const nonce = this.generateNonce();
    const payload: KrakenRequestData = {
      ...body,
      nonce,
    };

    // Add OTP if provided for 2FA
    if (otp) {
      payload.otp = otp;
    }

    // Generate signature
    const signature = generateKrakenSignature(endpoint, payload, this.secret);

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
        `Kraken API HTTP error: ${response.status} ${response.statusText} - ${errorText}`
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
   * Makes a request to a public Kraken API endpoint (no authentication required).
   *
   * @param endpoint - API endpoint path
   * @param params - Query parameters
   * @returns Promise resolving to the API response
   * @throws {Error} If the request fails or API returns an error
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
        `Kraken API HTTP error: ${response.status} ${response.statusText} - ${errorText}`
      );
    }

    const data = (await response.json()) as { error: string[]; result?: T };

    if (data.error && data.error.length > 0) {
      throw new Error(`Kraken API error: ${data.error.join(", ")}`);
    }

    return data.result as T;
  }
}

