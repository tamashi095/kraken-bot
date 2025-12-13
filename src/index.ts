/**
 * Kraken Bot - Main Entry Point
 *
 * This bot automatically:
 * 1. Checks USDC balance
 * 2. Sells all USDC for USD
 * 3. Withdraws USD to Mercury (if balance >= minimum threshold)
 */

import { KrakenClient } from "./client/KrakenClient.ts";
import { getAssetBalance } from "./api/balance.ts";
import { sellUSDCToUSD } from "./api/trading.ts";
import { withdrawToMercury } from "./api/withdrawal.ts";
import { parseDecimalToBigInt } from "./utils/bigint.ts";
import { loadConfig, validateConfig } from "./config.ts";

/**
 * Main application logic
 */
async function main(): Promise<void> {
  try {
    // Load and validate configuration
    const config = loadConfig();
    validateConfig(config);

    // Initialize Kraken client
    const client = new KrakenClient(config.kraken);

    console.log("🤖 Kraken Bot Started\n");

    // Step 1: Check USDC balance
    console.log("📊 Checking USDC balance...");
    const usdcBalance = await getAssetBalance(client, "USDC");
    const usdcBalanceBigInt = parseDecimalToBigInt(usdcBalance, 8);
    console.log(`   USDC Balance: ${usdcBalance}`);

    // Step 2: Sell USDC to USD if balance > 0
    if (usdcBalanceBigInt > 0n) {
      console.log(`\n💱 Selling ${usdcBalance} USDC to USD...`);
      const orderResult = await sellUSDCToUSD(client, usdcBalance);

      if (orderResult.txid && orderResult.txid.length > 0) {
        console.log(`   ✅ Order placed successfully`);
        console.log(`   Transaction ID: ${orderResult.txid.join(", ")}`);
        console.log(`   Order: ${orderResult.descr.order}`);
      }
    } else {
      console.log("   ℹ️  No USDC balance to sell");
    }

    console.log("   ℹ️  Waiting 2 seconds before checking USD balance...");
    await new Promise((resolve) => setTimeout(resolve, 2_000));

    // Step 3: Check USD balance and withdraw to Mercury
    console.log("\n💵 Checking USD balance...");
    const usdBalance = await getAssetBalance(client, "ZUSD");
    const usdBalanceBigInt = parseDecimalToBigInt(usdBalance, 4);
    console.log(`   USD Balance: $${usdBalance}`);

    // Calculate minimum withdrawal in base units ($10.00 = 100000 with 4 decimals)
    const minimumWithdrawal = BigInt(config.minimumWithdrawalUSD * 10_000);

    if (usdBalanceBigInt >= minimumWithdrawal) {
      console.log(`\n🏦 Withdrawing $${usdBalance} to Mercury...`);
      const withdrawResult = await withdrawToMercury(client, usdBalance);
      console.log(`   ✅ Withdrawal initiated`);
      console.log(`   Reference ID: ${withdrawResult.refid}`);
    } else {
      console.log(
        `   ℹ️  USD balance below minimum withdrawal threshold ($${config.minimumWithdrawalUSD})`
      );
    }

    console.log("\n✨ Bot completed successfully!");
  } catch (error) {
    console.error(
      "\n❌ Error:",
      error instanceof Error ? error.message : error
    );
    process.exit(1);
  }
}

// Run the bot
main();
