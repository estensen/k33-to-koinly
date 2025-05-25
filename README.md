# K33 to Koinly Converter

A Go program to convert K33 transaction export CSV files to Koinly Universal CSV format.

## Features

- Converts K33 deposits, withdrawals, and trades to Koinly format
- Pairs buy/sell trade legs automatically
- Handles scientific notation in trade IDs
- Converts timestamps to Koinly format
- Dry run mode for testing
- Comprehensive test coverage

## Usage

### Basic conversion
```bash
go run . -in k33_export.csv -out koinly_import.csv
```

### Dry run (preview without writing file)
```bash
go run . -in k33_export.csv -dryrun
```

### Custom file paths
```bash
go run . -in /path/to/k33.csv -out /path/to/koinly.csv
```

## Building

```bash
go build -o k33-to-koinly
./k33-to-koinly -in k33.csv -out koinly.csv
```

## Testing

```bash
go test ./converter
```

## Input Format (K33)

The program expects a K33 CSV export with the following columns:
- Type/Status (Deposit Complete, Withdrawal Complete, Trade)
- TradeID (for pairing buy/sell legs)
- Side (Buy, Sell)
- Amount (positive/negative values)
- Trade Status (Filled, Reject)
- Asset (currency symbol)
- Timestamp (UTC) (YYYY/MM/DD HH:MM:SS format)
- DepositTxhash/WithdrawalTxhash (optional)

## Output Format (Koinly)

Generates Koinly Universal CSV with columns:
- Date (YYYY-MM-DD HH:MM:SS)
- Sent Amount/Currency
- Received Amount/Currency
- Fee Amount/Currency (empty)
- Net Worth Amount/Currency (empty)
- Label (empty)
- Description (transaction type)
- TxHash (if available)

## Transaction Mapping

| K33 Transaction | Koinly Mapping |
|---|---|
| Deposit | Received Amount/Currency |
| Withdrawal | Sent Amount/Currency |
| Trade (Buy+Sell) | Sent=Sell leg, Received=Buy leg |

## Notes

- Rejected trades are skipped
- Trade pairs are matched by TradeID
- Scientific notation trade IDs are converted to integers
- Unpaired trades generate warnings
- Amounts are converted to absolute values (signs removed)