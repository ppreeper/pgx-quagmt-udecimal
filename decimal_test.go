package decimal_test

import (
	"context"
	"math"
	"os"
	"testing"

	pgxdecimal "github.com/ppreeper/pgx-quagmt-udecimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxtest"
	"github.com/quagmt/udecimal"
	"github.com/stretchr/testify/require"
)

var defaultConnTestRunner pgxtest.ConnTestRunner

func init() {
	defaultConnTestRunner = pgxtest.DefaultConnTestRunner()
	defaultConnTestRunner.CreateConfig = func(ctx context.Context, t testing.TB) *pgx.ConnConfig {
		config, err := pgx.ParseConfig(os.Getenv("PGX_TEST_DATABASE"))
		if err != nil {
			t.Fatalf("ParseConfig failed: %v", err)
		}
		return config
	}
	defaultConnTestRunner.AfterConnect = func(ctx context.Context, t testing.TB, conn *pgx.Conn) {
		pgxdecimal.Register(conn.TypeMap())
	}
}

func TestCodecDecodeValue(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *pgx.Conn) {
		original := udecimal.MustParse("1.234")

		rows, err := conn.Query(context.Background(), `select $1::numeric`, original)
		require.NoError(t, err)

		for rows.Next() {
			values, err := rows.Values()
			require.NoError(t, err)

			require.Len(t, values, 1)
			v0, ok := values[0].(udecimal.Decimal)
			require.True(t, ok)
			require.Equal(t, original, v0)
		}

		require.NoError(t, rows.Err())

		rows, err = conn.Query(context.Background(), `select $1::numeric`, nil)
		require.NoError(t, err)

		for rows.Next() {
			values, err := rows.Values()
			require.NoError(t, err)

			require.Len(t, values, 1)
			require.Equal(t, nil, values[0])
		}

		require.NoError(t, rows.Err())
	})
}

func TestNaN(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *pgx.Conn) {
		var d udecimal.Decimal
		err := conn.QueryRow(context.Background(), `select 'NaN'::numeric`).Scan(&d)
		require.EqualError(t, err, `can't scan into dest[0]: cannot scan NaN into *udecimal.Decimal`)
	})
}

func TestArray(t *testing.T) {
	defaultConnTestRunner.RunTest(context.Background(), t, func(ctx context.Context, t testing.TB, conn *pgx.Conn) {
		inputSlice := []udecimal.Decimal{}

		for i := 0; i < 10; i++ {
			d := udecimal.MustFromInt64(int64(i), 0)
			inputSlice = append(inputSlice, d)
		}

		var outputSlice []udecimal.Decimal
		err := conn.QueryRow(context.Background(), `select $1::numeric[]`, inputSlice).Scan(&outputSlice)
		require.NoError(t, err)

		require.Equal(t, len(inputSlice), len(outputSlice))
		for i := 0; i < len(inputSlice); i++ {
			require.True(t, outputSlice[i].Equal(inputSlice[i]))
		}
	})
}

func isExpectedEqDecimal(a udecimal.Decimal) func(interface{}) bool {
	return func(v interface{}) bool {
		return a.Equal(v.(udecimal.Decimal))
	}
}

func isExpectedEqNullDecimal(a udecimal.NullDecimal) func(interface{}) bool {
	return func(v interface{}) bool {
		b := v.(udecimal.NullDecimal)
		return a.Valid == b.Valid && a.Decimal.Equal(b.Decimal)
	}
}

func TestValueRoundTrip(t *testing.T) {
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "numeric", []pgxtest.ValueRoundTripTest{
		{
			Param:  udecimal.MustParse("1"),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustParse("1")),
		},
		{
			Param:  udecimal.MustParse("0.000012345"),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustParse("0.000012345")),
		},
		{
			Param:  udecimal.MustParse("123456.123456"),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustParse("123456.123456")),
		},
		{
			Param:  udecimal.MustParse("-1"),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustParse("-1")),
		},
		{
			Param:  udecimal.MustParse("-0.000012345"),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustParse("-0.000012345")),
		},
		{
			Param:  udecimal.MustParse("-123456.123456"),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustParse("-123456.123456")),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustParse("1"), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("1"), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustParse("0.000012345"), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("0.000012345"), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustParse("123456.123456"), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("123456.123456"), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustParse("-1"), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("-1"), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustParse("-0.000012345"), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("-0.000012345"), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustParse("-123456.123456"), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("-123456.123456"), Valid: true}),
		},
	})
}

func TestValueRoundTripFloat8(t *testing.T) {
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "float8", []pgxtest.ValueRoundTripTest{
		{
			Param:  udecimal.MustParse("1"),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustParse("1")),
		},
		{
			Param:  udecimal.MustParse("0.000012345"),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustParse("0.000012345")),
		},
		{
			Param:  udecimal.MustParse("123456.123456"),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustParse("123456.123456")),
		},
		{
			Param:  udecimal.MustParse("-1"),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustParse("-1")),
		},
		{
			Param:  udecimal.MustParse("-0.000012345"),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustParse("-0.000012345")),
		},
		{
			Param:  udecimal.MustParse("-123456.123456"),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustParse("-123456.123456")),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustParse("1"), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("1"), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustParse("0.000012345"), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("0.000012345"), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustParse("123456.123456"), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("123456.123456"), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustParse("-1"), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("-1"), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustParse("-0.000012345"), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("-0.000012345"), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustParse("-123456.123456"), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustParse("-123456.123456"), Valid: true}),
		},
	})
}

func TestValueRoundTripInt8(t *testing.T) {
	pgxtest.RunValueRoundTripTests(context.Background(), t, defaultConnTestRunner, nil, "int8", []pgxtest.ValueRoundTripTest{
		{
			Param:  udecimal.MustFromInt64(0, 0),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustFromInt64(0, 0)),
		},
		{
			Param:  udecimal.MustFromInt64(1, 0),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustFromInt64(1, 0)),
		},
		{
			Param:  udecimal.MustFromInt64(math.MaxInt64, 0),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustFromInt64(math.MaxInt64, 0)),
		},
		{
			Param:  udecimal.MustFromInt64(-1, 0),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustFromInt64(-1, 0)),
		},
		{
			Param:  udecimal.MustFromInt64(math.MinInt64, 0),
			Result: new(udecimal.Decimal),
			Test:   isExpectedEqDecimal(udecimal.MustFromInt64(math.MinInt64, 0)),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustFromInt64(0, 0), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustFromInt64(0, 0), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustFromInt64(1, 0), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustFromInt64(1, 0), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustFromInt64(math.MaxInt64, 0), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustFromInt64(math.MaxInt64, 0), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustFromInt64(-1, 0), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustFromInt64(-1, 0), Valid: true}),
		},
		{
			Param:  udecimal.NullDecimal{Decimal: udecimal.MustFromInt64(math.MinInt64, 0), Valid: true},
			Result: new(udecimal.NullDecimal),
			Test:   isExpectedEqNullDecimal(udecimal.NullDecimal{Decimal: udecimal.MustFromInt64(math.MinInt64, 0), Valid: true}),
		},
	})
}
