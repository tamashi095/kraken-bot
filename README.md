# Kraken Bot 🤖

An automated trading bot for Kraken that converts USDC to USD and withdraws to Mercury.

## Features

- ✅ **Automated Trading**: Automatically sells USDC for USD using market orders
- ✅ **Smart Withdrawals**: Withdraws USD to Mercury when balance exceeds threshold
- ✅ **Type-Safe**: Built with TypeScript for maximum reliability
- ✅ **Modular Architecture**: Clean, maintainable code structure
- ✅ **Secure**: Uses Kraken's HMAC-SHA512 authentication
- ✅ **Fast**: Powered by Bun for optimal performance

## Prerequisites

- [Bun](https://bun.sh/) (latest version)
- Kraken account with API credentials
- Mercury withdrawal key configured in Kraken

## Installation

1. Clone the repository:

```bash
git clone <repository-url>
cd kraken-bot
```

2. Install dependencies:

```bash
bun install
```

3. Copy the example environment file and configure your credentials:

```bash
cp .env.example .env
```

4. Edit `.env` and add your Kraken API credentials:

```env
KRAKEN_PUBLIC_KEY=your_public_api_key_here
KRAKEN_SECRET_KEY=your_secret_api_key_here
```

## Configuration

### Environment Variables

| Variable                 | Required | Default                  | Description                                 |
| ------------------------ | -------- | ------------------------ | ------------------------------------------- |
| `KRAKEN_PUBLIC_KEY`      | Yes      | -                        | Your Kraken API public key                  |
| `KRAKEN_SECRET_KEY`      | Yes      | -                        | Your Kraken API secret key                  |
| `KRAKEN_API_URL`         | No       | `https://api.kraken.com` | Kraken API base URL                         |
| `MINIMUM_WITHDRAWAL_USD` | No       | `10`                     | Minimum USD balance required for withdrawal |

### Kraken API Permissions

Your API key needs the following permissions:

- ✅ **Query Funds** - To check account balance
- ✅ **Create & Modify Orders** - To place sell orders
- ✅ **Withdraw Funds** - To withdraw to Mercury

## Usage

### Run the bot

```bash
bun start
```

### Development mode (with auto-reload)

```bash
bun dev
```

### Type checking

```bash
bun run type-check
```

## Project Structure

```
kraken-bot/
├── src/
│   ├── api/              # API endpoint modules
│   │   ├── balance.ts    # Balance operations
│   │   ├── trading.ts    # Trading operations
│   │   └── withdrawal.ts # Withdrawal operations
│   ├── client/           # HTTP client
│   │   └── KrakenClient.ts
│   ├── types/            # TypeScript types
│   │   └── kraken.ts
│   ├── utils/            # Utility functions
│   │   ├── bigint.ts     # BigInt helpers
│   │   └── crypto.ts     # Signature generation
│   ├── config.ts         # Configuration management
│   └── index.ts          # Main entry point
├── .env.example          # Example environment file
├── package.json
├── tsconfig.json
└── README.md
```

## How It Works

1. **Check USDC Balance**: Queries your Kraken account for USDC balance
2. **Sell USDC**: If balance > 0, places a market order to sell all USDC for USD
3. **Check USD Balance**: Queries the updated USD balance
4. **Withdraw to Mercury**: If USD balance ≥ minimum threshold, withdraws to Mercury

## API Modules

### Balance API

```typescript
import { getBalance, getAssetBalance } from "./api/balance.ts";

// Get all balances
const balances = await getBalance(client);

// Get specific asset balance
const usdcBalance = await getAssetBalance(client, "USDC");
```

### Trading API

```typescript
import { sellUSDCToUSD, placeOrder } from "./api/trading.ts";

// Sell USDC for USD
const result = await sellUSDCToUSD(client, "100.00");

// Place custom order
const order = await placeOrder(client, {
  type: "buy",
  ordertype: "market",
  pair: "XBTUSD",
  volume: "0.01",
});
```

### Withdrawal API

```typescript
import { withdraw, withdrawToMercury } from "./api/withdrawal.ts";

// Withdraw to Mercury
const result = await withdrawToMercury(client, "1000.00");

// Generic withdrawal
const withdrawal = await withdraw(client, "USD", "MyBank", "500.00");
```

## Security

- ✅ Never commit your `.env` file
- ✅ Store API keys securely
- ✅ Use API keys with minimal required permissions
- ✅ Enable 2FA on your Kraken account
- ✅ Regularly rotate your API keys

## Error Handling

The bot includes comprehensive error handling:

- API connection errors
- Authentication failures
- Insufficient balance errors
- Invalid order errors

All errors are logged to the console with descriptive messages.

## Development

### Adding New Features

1. Create new API modules in `src/api/`
2. Add types to `src/types/kraken.ts`
3. Update the main logic in `src/index.ts`

### Testing

Before running in production:

1. Test with small amounts
2. Use `validate: true` option for dry runs
3. Monitor Kraken API rate limits

## Troubleshooting

### "KRAKEN_PUBLIC_KEY is not set"

- Make sure `.env` file exists and contains your API credentials
- Check that environment variables are properly formatted

### "Invalid signature"

- Verify your API secret is correct
- Ensure system time is synchronized

### "Insufficient funds"

- Check your account balance on Kraken
- Ensure you have enough funds to cover fees

## License

MIT

## Disclaimer

This bot is provided as-is. Always test thoroughly before using with real funds. Trading cryptocurrencies carries risk. Use at your own discretion.
