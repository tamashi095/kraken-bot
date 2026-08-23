# Migration from Bun/TypeScript to Go

The bot is now implemented in Go and builds to a single executable with no
third-party modules or runtime dependency.

## What stayed the same

- One run checks USDC, sells all available USDC for USD, waits two seconds,
  checks USD, and withdraws when the configured threshold is met.
- The API key and base64 secret are unchanged.
- Kraken authentication still uses an increasing nonce and HMAC-SHA512
  `API-Sign` header, with the Funding API's new signing format for withdrawals.
- `.env` is loaded automatically from the working directory.

## What changed

- `bun start` is replaced by `./dist/kraken-bot`.
- `bun run type-check` is replaced by `go test ./...`.
- TypeScript and Bun source, configuration, and lock files were removed.
- Decimal amounts are parsed as exact integers. Invalid values and configured
  thresholds with more than four decimal places now fail instead of being
  rounded or truncated through floating-point arithmetic.
- Legacy `USD_WITHDRAWAL_KEY` configuration was replaced by stable
  `USD_FUNDING_METHOD_ID` and `USD_FUNDING_ADDRESS_ID` values.
- Withdrawals now quote and pin their fee before submission and use
  `fee_included=true`, making the requested balance the total debit.
- Withdrawal responses now expose a withdrawal ID and any required approval ID.
- Shutdown through `Ctrl-C` or `SIGTERM` now cancels an in-progress request or
  settlement wait cleanly.

## Upgrade steps

1. Build the binary:

   ```sh
   make build
   ```

2. Use Kraken's Funding API to choose the USD withdrawal `method_id`. Add the
   fiat destination through the Kraken UI and obtain its `address_id`.

3. Replace the legacy withdrawal key in `.env`:

   ```env
   KRAKEN_PUBLIC_KEY=...
   KRAKEN_SECRET_KEY=...
   USD_FUNDING_METHOD_ID=...
   USD_FUNDING_ADDRESS_ID=...
   ```

4. Run the replacement from the repository root:

   ```sh
   ./dist/kraken-bot
   ```

For deployment, copy `dist/kraken-bot` and `.env`; the Bun runtime and project
source files are no longer required.

## Endpoint migration

The withdrawal flow contains no fallback to `/0/private/Withdraw` or
`/0/private/WithdrawMethods`. It uses:

1. `GET /funding/v1/fees/{method_id}` to quote and pin the fee.
2. `POST /funding/v1/withdrawals` with the method ID, address ID, amount, and
   quote token.

Balance and order operations remain on their current Spot REST endpoints;
Kraken's Funding API only replaces funding operations.
