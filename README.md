# pgx-quagmt-udecimal

[![Test](https://github.com/ppreeper/pgx-quagmt-udecimal/actions/workflows/test.yml/badge.svg)](https://github.com/ppreeper/pgx-quagmt-udecimal/actions/workflows/test.yml)
![Coverage](https://raw.githubusercontent.com/ppreeper/pgx-quagmt-udecimal/main/.github/badges/coverage.svg)

PostgreSQL `numeric`, `float8`, and `int8` type support for [jackc/pgx/v5](https://github.com/jackc/pgx) using [quagmt/udecimal](https://github.com/quagmt/udecimal).

This is a port of [jackc/pgx-shopspring-decimal](https://github.com/jackc/pgx-shopspring-decimal), rewritten to use `quagmt/udecimal` — a high-performance 128-bit fixed-point decimal library — instead of `shopspring/decimal`.

## Installation

```
go get github.com/ppreeper/pgx-quagmt-udecimal
```

## Quick Start

Register the type mappings after establishing a connection:

```go
import (
    "github.com/jackc/pgx/v5"
    "github.com/ppreeper/pgx-quagmt-udecimal"
)

conn, err := pgx.Connect(ctx, connString)
if err != nil {
    // handle error
}

decimal.Register(conn.TypeMap())
```

Or with a connection pool:

```go
pool, err := pgxpool.New(ctx, connString)
if err != nil {
    // handle error
}

decimal.Register(pool.Config().ConnConfig.TypeMap)
```

Or via `AfterConnect` in the test runner or custom config:

```go
config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
    decimal.Register(conn.TypeMap())
    return nil
}
```

## Supported Types

| Go Type | PostgreSQL Types | Nullable |
|---------|-----------------|----------|
| `udecimal.Decimal` | `numeric`, `float8`, `int8` | No |
| `udecimal.NullDecimal` | `numeric`, `float8`, `int8` | Yes |
| `decimal.Decimal` | `numeric`, `float8`, `int8` | No |
| `decimal.NullDecimal` | `numeric`, `float8`, `int8` | Yes |

Slice variants are also supported: `[]T`, `*[]T`, `[]*T`, `*[]*T`.

## Usage Examples

### Query with `numeric`

```go
var amount udecimal.Decimal
err := conn.QueryRow(ctx, `SELECT price FROM products WHERE id = $1`, id).Scan(&amount)
```

### Insert with `numeric`

```go
amount := udecimal.MustParse("19.99")
_, err := conn.Exec(ctx, `INSERT INTO orders (amount) VALUES ($1)`, amount)
```

### Nullable decimals

```go
var discount udecimal.NullDecimal
err := conn.QueryRow(ctx, `SELECT discount FROM orders WHERE id = $1`, id).Scan(&discount)

if discount.Valid {
    fmt.Println("Discount:", discount.Decimal)
} else {
    fmt.Println("No discount")
}
```

### Arrays

```go
var amounts []udecimal.Decimal
err := conn.QueryRow(ctx, `SELECT prices FROM product WHERE id = $1`, id).Scan(&amounts)
```

### Using the wrapper types

The `decimal.Decimal` and `decimal.NullDecimal` wrapper types are type aliases for `udecimal.Decimal` and `udecimal.NullDecimal`. They exist to implement pgx's scanning/encoding interfaces. You can use either the `udecimal` types directly or the `decimal` wrappers — both work after registration:

```go
// These are equivalent after decimal.Register():
var a udecimal.Decimal
var b decimal.Decimal

// Both can be scanned into:
conn.QueryRow(ctx, `SELECT price FROM products LIMIT 1`).Scan(&a)
conn.QueryRow(ctx, `SELECT price FROM products LIMIT 1`).Scan(&b)
```

## How It Works

This library hooks into pgx's type system at three levels:

1. **Custom codec** — `NumericCodec` overrides `DecodeValue` to decode PostgreSQL `numeric` binary format directly into `udecimal.Decimal`.

2. **Encode/Scan plan wrappers** — `TryWrapNumericEncodePlan` and `TryWrapNumericScanPlan` intercept pgx's encode/scan pipeline, wrapping `udecimal.Decimal` and `udecimal.NullDecimal` with local types that implement `ScanNumeric`, `NumericValue`, `ScanFloat64`, `Float64Value`, `ScanInt64`, and `Int64Value`.

3. **Default type registration** — Registers all type variants (`T`, `*T`, `[]T`, `*[]T`, `[]*T`, `*[]*T`) for both `udecimal` and wrapper types under the `numeric` PostgreSQL type name.

### Conversion details

- **`pgtype.Numeric` ↔ `udecimal.Decimal`**: The `Numeric` type stores values as `Int * 10^Exp`. Conversion uses string building for scanning and a fast-path via `ToHiLo()` (128-bit coefficient) with string fallback for encoding.
- **`pgtype.Float8` ↔ `udecimal.Decimal`**: Uses `NewFromFloat64` / `InexactFloat64`. NaN and Infinity are rejected on scan.
- **`pgtype.Int8` ↔ `udecimal.Decimal`**: Uses `NewFromInt64` / `Int64`. Includes a workaround for `math.MinInt64` (udecimal's `Int64()` rejects it due to an upstream edge case).

## Limitations

- **Max precision**: udecimal supports up to 19 decimal places. PostgreSQL `numeric` supports up to 16383. Values exceeding 19 decimal places will fail to parse.
- **NaN / Infinity**: PostgreSQL `numeric` supports NaN and Infinity. This library rejects them on scan (returns an error), as udecimal has no representation for these values.
- **Float64 conversion**: `InexactFloat64()` may lose precision for very large numbers. This is inherent to float64 representation.

## Benchmarks

Run benchmarks with:

```
go test -bench=. -run=XXX ./...
```

Key results (Ryzen 7 7730U):

| Benchmark | Time |
|-----------|------|
| Int64 round-trip | ~21 ns/op |
| Numeric round-trip | ~224 ns/op |
| Float64 round-trip | ~203 ns/op |
| Null scan (null value) | ~2-3 ns/op |
| Encode/Scan plan wrapper (hit) | ~0.25 ns/op |

## License

MIT
