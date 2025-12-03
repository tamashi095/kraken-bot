/**
 * BigInt Utilities
 * 
 * This module provides utilities for working with BigInt values,
 * particularly for handling decimal numbers with precision.
 */

/**
 * Parses a decimal string to BigInt by converting to base units.
 * 
 * This is useful for precise financial calculations where floating-point
 * arithmetic would introduce rounding errors.
 *
 * @param value - Decimal string (e.g., "366.14886400")
 * @param decimals - Number of decimal places (default: 8)
 * @returns BigInt representation in base units
 *
 * @example
 * ```typescript
 * parseDecimalToBigInt("366.14886400", 8) // 36614886400n
 * parseDecimalToBigInt("0.00000000", 8)   // 0n
 * parseDecimalToBigInt("10.50", 2)        // 1050n
 * ```
 */
export function parseDecimalToBigInt(
  value: string,
  decimals: number = 8
): bigint {
  const [whole = "0", fraction = ""] = value.split(".");
  const paddedFraction = fraction.padEnd(decimals, "0").slice(0, decimals);
  return BigInt(whole + paddedFraction);
}

/**
 * Converts a BigInt in base units back to a decimal string.
 *
 * @param value - BigInt value in base units
 * @param decimals - Number of decimal places
 * @returns Formatted decimal string
 *
 * @example
 * ```typescript
 * bigIntToDecimal(36614886400n, 8) // "366.14886400"
 * bigIntToDecimal(1050n, 2)         // "10.50"
 * ```
 */
export function bigIntToDecimal(value: bigint, decimals: number = 8): string {
  const str = value.toString().padStart(decimals + 1, "0");
  const whole = str.slice(0, -decimals) || "0";
  const fraction = str.slice(-decimals);
  return `${whole}.${fraction}`;
}

/**
 * Minimum withdrawal amount in base units (e.g., $10.00 = 100000 with 4 decimals)
 */
export const MINIMUM_USD_WITHDRAWAL = 100000n; // $10.00 with 4 decimals

