# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build ./...          # Build
go test ./converter     # Run tests
go test -run TestName ./converter  # Run a single test
go run . -in k33.csv -out koinly.csv  # Run conversion
go run . -in k33.csv -dryrun         # Preview without writing
```

## Architecture

Single-package CLI tool that converts K33 crypto exchange CSV exports into Koinly Universal CSV format.

- `main.go` — CLI entry point, parses `-in`, `-out`, `-dryrun` flags
- `converter/converter.go` — all conversion logic: CSV parsing, record mapping, trade pairing
- `converter/converter_test.go` — unit and integration tests

**Core flow:** `Converter.parseRecords` reads K33 CSV rows, maps each to a `K33Record`, then dispatches by `TypeStatus` (Deposit/Withdrawal/Trade). Trades require pairing: two CSV rows (Buy + Sell legs) share a `TradeID` and are combined into one `KoinlyRecord`. Unpaired trades at the end of processing emit warnings.

**Key details:**
- Trade IDs may arrive in scientific notation (e.g. `1.0e+12`); `formatTradeID` uses `big.Float` to convert without precision loss.
- K33 CSVs may have a UTF-8 BOM; header parsing strips `\ufeff`.
- Amounts are stored with signs in K33 (negative for sells/withdrawals); the converter strips the `-` prefix.
- `Process` writes Koinly CSV; `ProcessDryRun` writes a human-readable summary. Both share `parseRecords`.
