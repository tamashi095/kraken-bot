# Migration Guide

## What Changed?

The codebase has been refactored from a single `index.ts` file into a professional, modular structure. All functionality remains the same, but the code is now more maintainable, testable, and extensible.

## New Project Structure

```
kraken-bot/
├── src/
│   ├── api/                    # API endpoint modules (NEW)
│   │   ├── balance.ts         # Balance operations
│   │   ├── trading.ts         # Trading operations (sellUSDCToUSD, etc.)
│   │   └── withdrawal.ts      # Withdrawal operations (withdrawToMercury, etc.)
│   │
│   ├── client/                 # HTTP client (NEW)
│   │   └── KrakenClient.ts    # Main API client class
│   │
│   ├── types/                  # TypeScript types (NEW)
│   │   └── kraken.ts          # All type definitions
│   │
│   ├── utils/                  # Utility functions (NEW)
│   │   ├── bigint.ts          # BigInt parsing utilities
│   │   └── crypto.ts          # Signature generation
│   │
│   ├── config.ts              # Configuration management (NEW)
│   └── index.ts               # Main entry point (REFACTORED)
│
├── index.ts.old               # Your original file (BACKUP)
├── .gitignore                 # Git ignore rules (NEW)
├── README.md                  # Comprehensive documentation (UPDATED)
├── package.json               # Updated with scripts (UPDATED)
└── tsconfig.json              # Updated with paths (UPDATED)
```

## Key Improvements

### 1. **Modular Architecture**
- Separated concerns into logical modules
- Each file has a single, clear responsibility
- Easy to find and modify specific functionality

### 2. **Better Type Safety**
- All types centralized in `src/types/kraken.ts`
- Comprehensive interfaces for all API requests/responses
- Better IDE autocomplete and error detection

### 3. **Reusable API Functions**
- Balance operations in `src/api/balance.ts`
- Trading operations in `src/api/trading.ts`
- Withdrawal operations in `src/api/withdrawal.ts`
- Can now easily import and use these functions elsewhere

### 4. **Configuration Management**
- Centralized config in `src/config.ts`
- Better error messages for missing environment variables
- Easy to add new configuration options

### 5. **Utility Functions**
- Crypto utilities separated into `src/utils/crypto.ts`
- BigInt helpers in `src/utils/bigint.ts`
- Reusable across the project

### 6. **Professional Documentation**
- Comprehensive JSDoc comments throughout
- Updated README with examples and guides
- Clear code organization

## How to Use the New Structure

### Running the Bot

**Old way:**
```bash
bun run index.ts
```

**New way:**
```bash
bun start          # Run the bot
bun dev            # Run with auto-reload
bun run type-check # Type check without running
```

### Importing Functions

**Example: Using Balance API**
```typescript
import { KrakenClient } from "./client/KrakenClient.ts";
import { getBalance, getAssetBalance } from "./api/balance.ts";

const client = new KrakenClient({
  apiKey: "your-key",
  secret: "your-secret"
});

// Get all balances
const balances = await getBalance(client);

// Get specific asset
const usdcBalance = await getAssetBalance(client, "USDC");
```

**Example: Using Trading API**
```typescript
import { sellUSDCToUSD, placeOrder } from "./api/trading.ts";

// Sell USDC
const result = await sellUSDCToUSD(client, "100.00");

// Custom order
const order = await placeOrder(client, {
  type: "buy",
  ordertype: "market",
  pair: "XBTUSD",
  volume: "0.01"
});
```

**Example: Using Withdrawal API**
```typescript
import { withdrawToMercury, withdraw } from "./api/withdrawal.ts";

// Withdraw to Mercury
await withdrawToMercury(client, "1000.00");

// Generic withdrawal
await withdraw(client, "USD", "MyBank", "500.00");
```

## Breaking Changes

### None! 🎉

The bot's functionality remains exactly the same. It still:
1. Checks USDC balance
2. Sells USDC to USD
3. Withdraws USD to Mercury

The only difference is the internal code organization.

## Configuration Changes

### Environment Variables

The environment variable names remain the same:
- `KRAKEN_PUBLIC_KEY` - Your public API key
- `KRAKEN_SECRET_KEY` - Your secret API key

### New Optional Variables

You can now optionally set:
- `KRAKEN_API_URL` - Custom API URL (defaults to `https://api.kraken.com`)
- `MINIMUM_WITHDRAWAL_USD` - Minimum withdrawal amount (defaults to `10`)

## File Mapping

Here's where everything moved:

| Old Location | New Location | Description |
|--------------|--------------|-------------|
| `getKrakenSignature()` | `src/utils/crypto.ts` | Signature generation |
| `KrakenClient` class | `src/client/KrakenClient.ts` | Main client |
| `sellUSDCToUSD()` | `src/api/trading.ts` | Trading functions |
| `withdrawToMercury()` | `src/api/withdrawal.ts` | Withdrawal functions |
| `getWithdrawMethods()` | `src/api/withdrawal.ts` | Withdrawal methods |
| `parseDecimalToBigInt()` | `src/utils/bigint.ts` | BigInt utilities |
| Type interfaces | `src/types/kraken.ts` | All types |
| Main logic | `src/index.ts` | Entry point |

## Testing the Refactored Code

1. Ensure your `.env` file is configured
2. Run the bot: `bun start`
3. Check that it works the same as before

## Rollback (If Needed)

If you need to rollback to the old version:

```bash
# Backup new structure
mv src src.backup

# Restore old file
mv index.ts.old index.ts

# Revert package.json
git checkout package.json tsconfig.json
```

## Need Help?

- Check the [README.md](./README.md) for comprehensive documentation
- Review the JSDoc comments in each file
- Examine the example code in this guide

## Next Steps

Now that you have a modular codebase, you can easily:
- Add new trading strategies
- Support more currency pairs
- Add logging and monitoring
- Write unit tests
- Create additional bots using the same modules

