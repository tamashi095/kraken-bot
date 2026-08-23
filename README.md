# Kraken Bot

A small Go program that sells the account's available USDC for USD on Kraken,
then withdraws the USD balance through Kraken's Funding API.

The project builds as one self-contained executable. It has no third-party Go
modules and does not need Go, Bun, Node.js, or package files on the machine
where the compiled binary runs.

## Behavior

Each run performs one cycle:

1. Read the `USDC` account balance.
2. If it is positive, place a market sell on `USDCUSD` for the full balance.
3. Wait two seconds, then read the `ZUSD` balance.
4. If the balance meets `MINIMUM_WITHDRAWAL_USD`, quote and pin the withdrawal
   fee with `fee_included=true`.
5. Withdraw through the configured funding method and saved address IDs. The
   full balance is the total debit; the destination receives the balance minus
   the quoted fee.

Withdrawals use `GET /funding/v1/fees/{method_id}` followed by
`POST /funding/v1/withdrawals`. There is no runtime fallback to Kraken's legacy
funding endpoints.

## Funding API differences

| Area | Legacy funding API | Funding API used here |
| --- | --- | --- |
| Destination | Human-readable withdrawal key such as `Mercury` | Stable `method_id` and `address_id` |
| Request | Form body with `nonce` in the body | JSON body with nonce in `API-Nonce` |
| Signature | Signs the URI path and encoded form body | Signs the URI path including its query string and the exact JSON body |
| Fees | Current fee applied during submission | Fee quoted first and pinned with a five-minute token |
| Result | A reference ID | Withdrawal ID plus net, gross, fee, and optional approval ID |

The API key and base64 secret remain the same. Kraken describes the newer API's
main benefits as stable IDs, reusable address scopes, fee pinning, and simpler
read-only permissions. See the [Funding API guide](https://docs.kraken.com/exchange/guides/rest/funding).

## Requirements

- Go 1.22 or newer to build from source
- A Kraken API key with these permissions:
  - Funds permissions — Query
  - Orders and trades — Create & modify orders
  - Funds permissions — Withdraw
- A compatible USD funding method and saved fiat withdrawal address

## Configure

Copy the example and edit it:

```sh
cp .env.example .env
```

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `KRAKEN_PUBLIC_KEY` | Yes | — | Kraken public API key |
| `KRAKEN_SECRET_KEY` | Yes | — | Base64-encoded Kraken private API key |
| `USD_FUNDING_METHOD_ID` | Yes | — | Stable UUID for the USD withdrawal method |
| `USD_FUNDING_ADDRESS_ID` | Yes | — | Stable ID of the saved destination address |
| `KRAKEN_API_URL` | No | `https://api.kraken.com` | Kraken API base URL |
| `MINIMUM_WITHDRAWAL_USD` | No | `10` | Minimum USD balance, with at most four decimal places |

The binary reads `.env` from its working directory. Values already present in
the process environment take precedence over `.env`.

### Find the funding IDs

The discovery commands require only `KRAKEN_PUBLIC_KEY` and
`KRAKEN_SECRET_KEY`; the funding IDs do not need to be configured yet.

```sh
# List every deposit and withdrawal method.
./dist/kraken-bot funding-methods

# List only withdrawal methods, then inspect the USD rows.
./dist/kraken-bot funding-methods withdraw

# List saved destinations compatible with the selected method.
./dist/kraken-bot funding-addresses METHOD_ID
```

When the credentials are stored in Doppler, prefix the commands with
`doppler run --`. Record the USD method's `METHOD_ID`, then find the intended
verified destination and record its `ADDRESS_ID`. The commands omit the actual
destination address and banking details from their output.

These commands use Kraken's
[`GET /funding/v1/methods/{direction}`](https://docs.kraken.com/api-reference/funding-beta/list-funding-methods)
and [`GET /funding/v1/addresses`](https://docs.kraken.com/api-reference/funding-beta/list-funding-addresses)
endpoints and follow cursor pagination automatically. Add or edit fiat
destinations in the Kraken UI before running address discovery if necessary.

The old `USD_WITHDRAWAL_KEY=Mercury` value cannot be translated locally because
the new API requires Kraken-issued IDs. Startup fails with a migration hint if
only that legacy variable is present.

## Build and run

```sh
make build
./dist/kraken-bot
./dist/kraken-bot help
```

The equivalent command without Make is:

```sh
mkdir -p dist
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o dist/kraken-bot ./cmd/kraken-bot
```

To build a Linux AMD64 binary from macOS or another platform:

```sh
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/kraken-bot-linux-amd64 ./cmd/kraken-bot
```

Copy only the resulting binary and its `.env` file to the target machine.

## Test

```sh
make test
```

Tests use an in-memory HTTP transport and never contact Kraken or place orders.
They cover exact decimal conversion, configuration loading, authenticated
request signing, API payloads, errors, and the full bot workflow.

## Project structure

```text
cmd/kraken-bot/       executable entry point
internal/bot/         USDC sale and USD withdrawal workflow
internal/config/      environment and .env loading
internal/decimal/     exact fixed-point amount parsing
internal/kraken/      authenticated Kraken REST client
```

## Operational notes

- Run only one process per Kraken API key. [Kraken requires a strictly increasing
  nonce](https://docs.kraken.com/exchange/guides/rest/authentication); sharing a
  key between processes can cause invalid-nonce errors.
- The bot submits real market orders and real withdrawals. Test with small
  balances and a dedicated, minimally privileged API key before production use.
- Keep `.env` private. It is ignored by Git, and credentials are never logged.
- Keep the host clock synchronized so nonce values remain valid.
- Fee quotes expire after five minutes. The bot quotes immediately before
  submission and does not reuse stored fee tokens.

## License

MIT

## Disclaimer

This software is provided as-is. Trading and withdrawing funds carries risk;
review and test it for your account before use.
