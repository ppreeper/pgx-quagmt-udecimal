package decimal

import (
	"math"
	"math/big"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/quagmt/udecimal"
)

// --- Internal conversion function benchmarks ---

func BenchmarkNumericToUDecimal_Small(b *testing.B) {
	v := pgtype.Numeric{Int: big.NewInt(12345), Exp: -3, Valid: true}
	b.ResetTimer()
	for range b.N {
		_, _ = numericToUDecimal(v)
	}
}

func BenchmarkNumericToUDecimal_Medium(b *testing.B) {
	v := pgtype.Numeric{Int: big.NewInt(123456123456), Exp: -6, Valid: true}
	b.ResetTimer()
	for range b.N {
		_, _ = numericToUDecimal(v)
	}
}

func BenchmarkNumericToUDecimal_Large(b *testing.B) {
	v := pgtype.Numeric{Int: big.NewInt(999999999999999999), Exp: -18, Valid: true}
	b.ResetTimer()
	for range b.N {
		_, _ = numericToUDecimal(v)
	}
}

func BenchmarkNumericToUDecimal_Negative(b *testing.B) {
	v := pgtype.Numeric{Int: big.NewInt(-123456), Exp: -3, Valid: true}
	b.ResetTimer()
	for range b.N {
		_, _ = numericToUDecimal(v)
	}
}

func BenchmarkNumericToUDecimal_PositiveExp(b *testing.B) {
	v := pgtype.Numeric{Int: big.NewInt(12345), Exp: 4, Valid: true}
	b.ResetTimer()
	for range b.N {
		_, _ = numericToUDecimal(v)
	}
}

func BenchmarkNumericToUDecimal_LeadingZeros(b *testing.B) {
	v := pgtype.Numeric{Int: big.NewInt(1), Exp: -12, Valid: true}
	b.ResetTimer()
	for range b.N {
		_, _ = numericToUDecimal(v)
	}
}

func BenchmarkNumericToUDecimal_Nil(b *testing.B) {
	v := pgtype.Numeric{Int: nil, Exp: 0, Valid: false}
	b.ResetTimer()
	for range b.N {
		_, _ = numericToUDecimal(v)
	}
}

func BenchmarkUDecimalToNumeric_FastPath(b *testing.B) {
	d := udecimal.MustParse("123456.123456")
	b.ResetTimer()
	for range b.N {
		_ = uDecimalToNumeric(d)
	}
}

func BenchmarkUDecimalToNumeric_FastPath_SmallInt(b *testing.B) {
	d := udecimal.MustParse("42")
	b.ResetTimer()
	for range b.N {
		_ = uDecimalToNumeric(d)
	}
}

func BenchmarkUDecimalToNumeric_FastPath_Negative(b *testing.B) {
	d := udecimal.MustParse("-123456.123456")
	b.ResetTimer()
	for range b.N {
		_ = uDecimalToNumeric(d)
	}
}

func BenchmarkUDecimalToNumeric_FastPath_HighLo(b *testing.B) {
	d := udecimal.MustParse("9999999999999999.9999")
	b.ResetTimer()
	for range b.N {
		_ = uDecimalToNumeric(d)
	}
}

func BenchmarkUDecimalToNumeric_Fallback_Large(b *testing.B) {
	// Construct a number that exceeds 128-bit coefficient (overflows ToHiLo fast path).
	// 19 decimal places (max precision), large integer part → coefficient > 2^128.
	d := udecimal.MustParse("123456789012345678901234567890123456789.1234567890123456789")
	b.ResetTimer()
	for range b.N {
		_ = uDecimalToNumeric(d)
	}
}

func BenchmarkDecimalToInt64_Small(b *testing.B) {
	d := udecimal.MustParse("42")
	b.ResetTimer()
	for range b.N {
		_, _ = decimalToInt64(d)
	}
}

func BenchmarkDecimalToInt64_MaxInt64(b *testing.B) {
	d := udecimal.MustFromInt64(math.MaxInt64, 0)
	b.ResetTimer()
	for range b.N {
		_, _ = decimalToInt64(d)
	}
}

func BenchmarkDecimalToInt64_MinInt64(b *testing.B) {
	d := udecimal.MustFromInt64(math.MinInt64, 0)
	b.ResetTimer()
	for range b.N {
		_, _ = decimalToInt64(d)
	}
}

func BenchmarkDecimalToInt64_Zero(b *testing.B) {
	d := udecimal.MustParse("0")
	b.ResetTimer()
	for range b.N {
		_, _ = decimalToInt64(d)
	}
}

// --- Scan/Value method benchmarks ---

func BenchmarkDecimal_ScanNumeric(b *testing.B) {
	v := pgtype.Numeric{Int: big.NewInt(123456), Exp: -3, Valid: true}
	var d Decimal
	b.ResetTimer()
	for range b.N {
		_ = d.ScanNumeric(v)
	}
}

func BenchmarkDecimal_NumericValue(b *testing.B) {
	d := Decimal(udecimal.MustParse("123456.123456"))
	b.ResetTimer()
	for range b.N {
		_, _ = d.NumericValue()
	}
}

func BenchmarkDecimal_ScanFloat64(b *testing.B) {
	v := pgtype.Float8{Float64: 123456.123456, Valid: true}
	var d Decimal
	b.ResetTimer()
	for range b.N {
		_ = d.ScanFloat64(v)
	}
}

func BenchmarkDecimal_Float64Value(b *testing.B) {
	d := Decimal(udecimal.MustParse("123456.123456"))
	b.ResetTimer()
	for range b.N {
		_, _ = d.Float64Value()
	}
}

func BenchmarkDecimal_ScanInt64(b *testing.B) {
	v := pgtype.Int8{Int64: 123456, Valid: true}
	var d Decimal
	b.ResetTimer()
	for range b.N {
		_ = d.ScanInt64(v)
	}
}

func BenchmarkDecimal_Int64Value(b *testing.B) {
	d := Decimal(udecimal.MustFromInt64(123456, 0))
	b.ResetTimer()
	for range b.N {
		_, _ = d.Int64Value()
	}
}

func BenchmarkDecimal_Int64Value_MinInt64(b *testing.B) {
	d := Decimal(udecimal.MustFromInt64(math.MinInt64, 0))
	b.ResetTimer()
	for range b.N {
		_, _ = d.Int64Value()
	}
}

// --- NullDecimal Scan/Value benchmarks ---

func BenchmarkNullDecimal_ScanNumeric_Valid(b *testing.B) {
	v := pgtype.Numeric{Int: big.NewInt(123456), Exp: -3, Valid: true}
	var d NullDecimal
	b.ResetTimer()
	for range b.N {
		_ = d.ScanNumeric(v)
	}
}

func BenchmarkNullDecimal_ScanNumeric_Null(b *testing.B) {
	v := pgtype.Numeric{Int: nil, Exp: 0, Valid: false}
	var d NullDecimal
	b.ResetTimer()
	for range b.N {
		_ = d.ScanNumeric(v)
	}
}

func BenchmarkNullDecimal_NumericValue_Valid(b *testing.B) {
	d := NullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("123456.123456"), Valid: true})
	b.ResetTimer()
	for range b.N {
		_, _ = d.NumericValue()
	}
}

func BenchmarkNullDecimal_NumericValue_Null(b *testing.B) {
	d := NullDecimal(udecimal.NullDecimal{Valid: false})
	b.ResetTimer()
	for range b.N {
		_, _ = d.NumericValue()
	}
}

func BenchmarkNullDecimal_ScanFloat64_Valid(b *testing.B) {
	v := pgtype.Float8{Float64: 123456.123456, Valid: true}
	var d NullDecimal
	b.ResetTimer()
	for range b.N {
		_ = d.ScanFloat64(v)
	}
}

func BenchmarkNullDecimal_ScanFloat64_Null(b *testing.B) {
	v := pgtype.Float8{Valid: false}
	var d NullDecimal
	b.ResetTimer()
	for range b.N {
		_ = d.ScanFloat64(v)
	}
}

func BenchmarkNullDecimal_ScanInt64_Valid(b *testing.B) {
	v := pgtype.Int8{Int64: 123456, Valid: true}
	var d NullDecimal
	b.ResetTimer()
	for range b.N {
		_ = d.ScanInt64(v)
	}
}

func BenchmarkNullDecimal_ScanInt64_Null(b *testing.B) {
	v := pgtype.Int8{Valid: false}
	var d NullDecimal
	b.ResetTimer()
	for range b.N {
		_ = d.ScanInt64(v)
	}
}

// --- Round-trip benchmarks ---

func BenchmarkRoundTrip_Numeric(b *testing.B) {
	d := Decimal(udecimal.MustParse("123456.123456"))
	b.ResetTimer()
	for range b.N {
		num, _ := d.NumericValue()
		var d2 Decimal
		_ = d2.ScanNumeric(num)
	}
}

func BenchmarkRoundTrip_Float64(b *testing.B) {
	d := Decimal(udecimal.MustParse("123456.123456"))
	b.ResetTimer()
	for range b.N {
		f, _ := d.Float64Value()
		var d2 Decimal
		_ = d2.ScanFloat64(f)
	}
}

func BenchmarkRoundTrip_Int64(b *testing.B) {
	d := Decimal(udecimal.MustFromInt64(123456, 0))
	b.ResetTimer()
	for range b.N {
		i, _ := d.Int64Value()
		var d2 Decimal
		_ = d2.ScanInt64(i)
	}
}

func BenchmarkRoundTrip_NullDecimal_Numeric(b *testing.B) {
	d := NullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("123456.123456"), Valid: true})
	b.ResetTimer()
	for range b.N {
		num, _ := d.NumericValue()
		var d2 NullDecimal
		_ = d2.ScanNumeric(num)
	}
}

func BenchmarkRoundTrip_NullDecimal_Int64(b *testing.B) {
	d := NullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustFromInt64(123456, 0), Valid: true})
	b.ResetTimer()
	for range b.N {
		i, _ := d.Int64Value()
		var d2 NullDecimal
		_ = d2.ScanInt64(i)
	}
}

// --- Encode/Scan plan wrapper benchmarks ---

func BenchmarkTryWrapNumericEncodePlan_Decimal(b *testing.B) {
	d := udecimal.MustParse("123456.123456")
	b.ResetTimer()
	for range b.N {
		_, _, _ = TryWrapNumericEncodePlan(d)
	}
}

func BenchmarkTryWrapNumericEncodePlan_NullDecimal(b *testing.B) {
	d := udecimal.NullDecimal{Decimal: udecimal.MustParse("123456.123456"), Valid: true}
	b.ResetTimer()
	for range b.N {
		_, _, _ = TryWrapNumericEncodePlan(d)
	}
}

func BenchmarkTryWrapNumericEncodePlan_Miss(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		_, _, _ = TryWrapNumericEncodePlan("not a decimal")
	}
}

func BenchmarkTryWrapNumericScanPlan_Decimal(b *testing.B) {
	var d udecimal.Decimal
	b.ResetTimer()
	for range b.N {
		_, _, _ = TryWrapNumericScanPlan(&d)
	}
}

func BenchmarkTryWrapNumericScanPlan_NullDecimal(b *testing.B) {
	var d udecimal.NullDecimal
	b.ResetTimer()
	for range b.N {
		_, _, _ = TryWrapNumericScanPlan(&d)
	}
}

func BenchmarkTryWrapNumericScanPlan_Miss(b *testing.B) {
	var s string
	b.ResetTimer()
	for range b.N {
		_, _, _ = TryWrapNumericScanPlan(&s)
	}
}

// --- Array benchmarks ---

func BenchmarkDecimal_ArrayEncodeDecode_10(b *testing.B) {
	input := make([]Decimal, 10)
	for i := range input {
		input[i] = Decimal(udecimal.MustFromInt64(int64(i), 0))
	}
	b.ResetTimer()
	for range b.N {
		for _, d := range input {
			_, _ = d.NumericValue()
		}
	}
}

func BenchmarkDecimal_ArrayEncodeDecode_100(b *testing.B) {
	input := make([]Decimal, 100)
	for i := range input {
		input[i] = Decimal(udecimal.MustFromInt64(int64(i), 0))
	}
	b.ResetTimer()
	for range b.N {
		for _, d := range input {
			_, _ = d.NumericValue()
		}
	}
}

func BenchmarkDecimal_ArrayEncodeDecode_1000(b *testing.B) {
	input := make([]Decimal, 1000)
	for i := range input {
		input[i] = Decimal(udecimal.MustFromInt64(int64(i), 0))
	}
	b.ResetTimer()
	for range b.N {
		for _, d := range input {
			_, _ = d.NumericValue()
		}
	}
}
