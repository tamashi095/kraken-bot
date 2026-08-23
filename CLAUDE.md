# Repository development guide

This is a dependency-free Go 1.22 project that produces one executable.

- Format changed Go files with `gofmt`.
- Run `go test ./...` and `go vet ./...` after changes.
- Build with `make build` or the command documented in `README.md`.
- Keep Kraken amounts as decimal strings or exact integer units; do not use
  floating-point arithmetic for balances or withdrawal thresholds.
- Funding withdrawals must use `/funding/v1` stable IDs and a freshly quoted
  fee token. Do not reintroduce legacy `/0/private` funding endpoints.
- Never log API keys, secrets, signatures, or complete authenticated payloads.
- Tests must use mock transports and must never call Kraken or place real orders.
