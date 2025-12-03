/**
 * Cryptographic Utilities for Kraken API
 * 
 * This module handles signature generation for authenticated Kraken API requests.
 */

import { createHash, createHmac } from "crypto";
import { stringify } from "querystring";
import type { KrakenDataInput, KrakenRequestData } from "../types/kraken.ts";

/**
 * Generates a Kraken API signature for authenticated requests.
 * 
 * The signature is created by:
 * 1. Creating a SHA256 hash of (nonce + POST data)
 * 2. Combining the URL path with the hash
 * 3. Creating an HMAC-SHA512 signature using the secret key
 *
 * @param urlPath - The API endpoint path (e.g., "/0/private/Balance")
 * @param data - Request data as either a JSON string or an object with a nonce property
 * @param secret - Base64-encoded API secret key
 * @returns Base64-encoded HMAC-SHA512 signature
 * @throws {Error} If data type is invalid or nonce is missing
 */
export function generateKrakenSignature(
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

  // Create SHA256 hash of the encoded data
  const sha256Hash = createHash("sha256").update(encoded).digest();

  // Combine URL path with binary hash
  const message = urlPath + sha256Hash.toString("binary");

  // Decode secret from base64
  const secretBuffer = Buffer.from(secret, "base64");

  // Create HMAC-SHA512 signature
  const hmac = createHmac("sha512", secretBuffer);
  hmac.update(message, "binary");

  return hmac.digest("base64");
}

