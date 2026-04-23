package decimal

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/quagmt/udecimal"
)

type Decimal udecimal.Decimal

func (d *Decimal) ScanNumeric(v pgtype.Numeric) error {
	if !v.Valid {
		return fmt.Errorf("cannot scan NULL into *udecimal.Decimal")
	}

	if v.NaN {
		return fmt.Errorf("cannot scan NaN into *udecimal.Decimal")
	}

	if v.InfinityModifier != pgtype.Finite {
		return fmt.Errorf("cannot scan %v into *udecimal.Decimal", v.InfinityModifier)
	}

	dec, err := numericToUDecimal(v)
	if err != nil {
		return fmt.Errorf("cannot scan numeric into *udecimal.Decimal: %w", err)
	}

	*d = Decimal(dec)

	return nil
}

func (d Decimal) NumericValue() (pgtype.Numeric, error) {
	dd := udecimal.Decimal(d)
	return uDecimalToNumeric(dd), nil
}

func (d *Decimal) ScanFloat64(v pgtype.Float8) error {
	if !v.Valid {
		return fmt.Errorf("cannot scan NULL into *udecimal.Decimal")
	}

	if math.IsNaN(v.Float64) {
		return fmt.Errorf("cannot scan NaN into *udecimal.Decimal")
	}

	if math.IsInf(v.Float64, 0) {
		return fmt.Errorf("cannot scan %v into *udecimal.Decimal", v.Float64)
	}

	dec, err := udecimal.NewFromFloat64(v.Float64)
	if err != nil {
		return fmt.Errorf("cannot scan %v into *udecimal.Decimal: %w", v.Float64, err)
	}

	*d = Decimal(dec)

	return nil
}

func (d Decimal) Float64Value() (pgtype.Float8, error) {
	dd := udecimal.Decimal(d)
	return pgtype.Float8{Float64: dd.InexactFloat64(), Valid: true}, nil
}

func (d *Decimal) ScanInt64(v pgtype.Int8) error {
	if !v.Valid {
		return fmt.Errorf("cannot scan NULL into *udecimal.Decimal")
	}

	dec, err := udecimal.NewFromInt64(v.Int64, 0)
	if err != nil {
		return fmt.Errorf("cannot scan %v into *udecimal.Decimal: %w", v.Int64, err)
	}

	*d = Decimal(dec)

	return nil
}

func (d Decimal) Int64Value() (pgtype.Int8, error) {
	dd := udecimal.Decimal(d)

	truncated := dd.Trunc(0)
	if !dd.Equal(truncated) {
		return pgtype.Int8{}, fmt.Errorf("cannot convert %v to int64", dd)
	}

	i, err := decimalToInt64(dd)
	if err != nil {
		return pgtype.Int8{}, fmt.Errorf("cannot convert %v to int64", dd)
	}

	return pgtype.Int8{Int64: i, Valid: true}, nil
}

type NullDecimal udecimal.NullDecimal

func (d *NullDecimal) ScanNumeric(v pgtype.Numeric) error {
	if !v.Valid {
		*d = NullDecimal{}
		return nil
	}

	if v.NaN {
		return fmt.Errorf("cannot scan NaN into *udecimal.NullDecimal")
	}

	if v.InfinityModifier != pgtype.Finite {
		return fmt.Errorf("cannot scan %v into *udecimal.NullDecimal", v.InfinityModifier)
	}

	dec, err := numericToUDecimal(v)
	if err != nil {
		return fmt.Errorf("cannot scan numeric into *udecimal.NullDecimal: %w", err)
	}

	*d = NullDecimal(udecimal.NullDecimal{Decimal: dec, Valid: true})

	return nil
}

func (d NullDecimal) NumericValue() (pgtype.Numeric, error) {
	if !d.Valid {
		return pgtype.Numeric{}, nil
	}

	return uDecimalToNumeric(d.Decimal), nil
}

func (d *NullDecimal) ScanFloat64(v pgtype.Float8) error {
	if !v.Valid {
		*d = NullDecimal{}
		return nil
	}

	if math.IsNaN(v.Float64) {
		return fmt.Errorf("cannot scan NaN into *udecimal.NullDecimal")
	}

	if math.IsInf(v.Float64, 0) {
		return fmt.Errorf("cannot scan %v into *udecimal.NullDecimal", v.Float64)
	}

	dec, err := udecimal.NewFromFloat64(v.Float64)
	if err != nil {
		return fmt.Errorf("cannot scan %v into *udecimal.NullDecimal: %w", v.Float64, err)
	}

	*d = NullDecimal(udecimal.NullDecimal{Decimal: dec, Valid: true})

	return nil
}

func (d NullDecimal) Float64Value() (pgtype.Float8, error) {
	if !d.Valid {
		return pgtype.Float8{}, nil
	}

	return pgtype.Float8{Float64: d.Decimal.InexactFloat64(), Valid: true}, nil
}

func (d *NullDecimal) ScanInt64(v pgtype.Int8) error {
	if !v.Valid {
		*d = NullDecimal{}
		return nil
	}

	dec, err := udecimal.NewFromInt64(v.Int64, 0)
	if err != nil {
		return fmt.Errorf("cannot scan %v into *udecimal.NullDecimal: %w", v.Int64, err)
	}

	*d = NullDecimal(udecimal.NullDecimal{Decimal: dec, Valid: true})

	return nil
}

func (d NullDecimal) Int64Value() (pgtype.Int8, error) {
	if !d.Valid {
		return pgtype.Int8{}, nil
	}

	dd := d.Decimal

	truncated := dd.Trunc(0)
	if !dd.Equal(truncated) {
		return pgtype.Int8{}, fmt.Errorf("cannot convert %v to int64", dd)
	}

	i, err := decimalToInt64(dd)
	if err != nil {
		return pgtype.Int8{}, fmt.Errorf("cannot convert %v to int64", dd)
	}

	return pgtype.Int8{Int64: i, Valid: true}, nil
}

// decimalToInt64 converts a udecimal.Decimal to int64, working around
// udecimal's Int64() rejecting math.MinInt64. udecimal checks |value| > MaxInt64,
// but |MinInt64| == MaxInt64+1 is valid as a negative int64.
func decimalToInt64(dd udecimal.Decimal) (int64, error) {
	i, err := dd.Int64()
	if err == nil {
		return i, nil
	}

	if dd.IsNeg() {
		pos := dd.Neg()
		_, hi, lo, prec, ok := pos.ToHiLo()
		if ok && hi == 0 && prec == 0 && lo == uint64(math.MaxInt64)+1 {
			return math.MinInt64, nil
		}
	}

	return 0, err
}

// numericToUDecimal converts a pgtype.Numeric to a udecimal.Decimal.
// The numeric value is v.Int * 10^v.Exp.
func numericToUDecimal(v pgtype.Numeric) (udecimal.Decimal, error) {
	if v.Int == nil {
		return udecimal.Decimal{}, nil
	}

	intStr := v.Int.Text(10)

	neg := false
	digits := intStr
	if len(digits) > 0 && digits[0] == '-' {
		neg = true
		digits = digits[1:]
	}

	exp := int(v.Exp)

	var result string
	if exp >= 0 {
		result = digits + strings.Repeat("0", exp)
	} else {
		absExp := -exp
		if absExp >= len(digits) {
			result = "0." + strings.Repeat("0", absExp-len(digits)) + digits
		} else {
			pos := len(digits) - absExp
			result = digits[:pos] + "." + digits[pos:]
		}
	}

	if neg {
		result = "-" + result
	}

	return udecimal.Parse(result)
}

// uDecimalToNumeric converts a udecimal.Decimal to a pgtype.Numeric.
func uDecimalToNumeric(d udecimal.Decimal) pgtype.Numeric {
	// Fast path: use ToHiLo for numbers that fit in 128-bit representation.
	neg, hi, lo, prec, ok := d.ToHiLo()
	if ok {
		bi := new(big.Int)
		if hi != 0 {
			bi.SetUint64(hi)
			bi.Lsh(bi, 64)
			bi.Add(bi, new(big.Int).SetUint64(lo))
		} else {
			bi.SetUint64(lo)
		}
		if neg {
			bi.Neg(bi)
		}
		return pgtype.Numeric{Int: bi, Exp: -int32(prec), Valid: true}
	}

	// Fallback: parse the string representation for numbers that overflow 128-bit.
	str := d.String()
	negStr := strings.HasPrefix(str, "-")
	if negStr {
		str = str[1:]
	}

	var coefStr string
	var numExp int32
	if dotIdx := strings.IndexByte(str, '.'); dotIdx >= 0 {
		coefStr = str[:dotIdx] + str[dotIdx+1:]
		numExp = -int32(len(str) - dotIdx - 1)
	} else {
		coefStr = str
		numExp = 0
	}

	bi, ok := new(big.Int).SetString(coefStr, 10)
	if !ok {
		bi = new(big.Int)
	}
	if negStr {
		bi.Neg(bi)
	}

	return pgtype.Numeric{Int: bi, Exp: numExp, Valid: true}
}

func TryWrapNumericEncodePlan(value interface{}) (plan pgtype.WrappedEncodePlanNextSetter, nextValue interface{}, ok bool) {
	switch value := value.(type) {
	case udecimal.Decimal:
		return &wrapDecimalEncodePlan{}, Decimal(value), true
	case udecimal.NullDecimal:
		return &wrapNullDecimalEncodePlan{}, NullDecimal(value), true
	}

	return nil, nil, false
}

type wrapDecimalEncodePlan struct {
	next pgtype.EncodePlan
}

func (plan *wrapDecimalEncodePlan) SetNext(next pgtype.EncodePlan) { plan.next = next }

func (plan *wrapDecimalEncodePlan) Encode(value interface{}, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(Decimal(value.(udecimal.Decimal)), buf)
}

type wrapNullDecimalEncodePlan struct {
	next pgtype.EncodePlan
}

func (plan *wrapNullDecimalEncodePlan) SetNext(next pgtype.EncodePlan) { plan.next = next }

func (plan *wrapNullDecimalEncodePlan) Encode(value interface{}, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(NullDecimal(value.(udecimal.NullDecimal)), buf)
}

func TryWrapNumericScanPlan(target interface{}) (plan pgtype.WrappedScanPlanNextSetter, nextDst interface{}, ok bool) {
	switch target := target.(type) {
	case *udecimal.Decimal:
		return &wrapDecimalScanPlan{}, (*Decimal)(target), true
	case *udecimal.NullDecimal:
		return &wrapNullDecimalScanPlan{}, (*NullDecimal)(target), true
	}

	return nil, nil, false
}

type wrapDecimalScanPlan struct {
	next pgtype.ScanPlan
}

func (plan *wrapDecimalScanPlan) SetNext(next pgtype.ScanPlan) { plan.next = next }

func (plan *wrapDecimalScanPlan) Scan(src []byte, dst interface{}) error {
	return plan.next.Scan(src, (*Decimal)(dst.(*udecimal.Decimal)))
}

type wrapNullDecimalScanPlan struct {
	next pgtype.ScanPlan
}

func (plan *wrapNullDecimalScanPlan) SetNext(next pgtype.ScanPlan) { plan.next = next }

func (plan *wrapNullDecimalScanPlan) Scan(src []byte, dst interface{}) error {
	return plan.next.Scan(src, (*NullDecimal)(dst.(*udecimal.NullDecimal)))
}

type NumericCodec struct {
	pgtype.NumericCodec
}

func (NumericCodec) DecodeValue(tm *pgtype.Map, oid uint32, format int16, src []byte) (interface{}, error) {
	if src == nil {
		return nil, nil
	}

	var target udecimal.Decimal
	scanPlan := tm.PlanScan(oid, format, &target)
	if scanPlan == nil {
		return nil, fmt.Errorf("PlanScan did not find a plan")
	}

	err := scanPlan.Scan(src, &target)
	if err != nil {
		return nil, err
	}

	return target, nil
}

// Register registers the quagmt/udecimal integration with a pgtype.Map.
func Register(m *pgtype.Map) {
	m.TryWrapEncodePlanFuncs = append([]pgtype.TryWrapEncodePlanFunc{TryWrapNumericEncodePlan}, m.TryWrapEncodePlanFuncs...)
	m.TryWrapScanPlanFuncs = append([]pgtype.TryWrapScanPlanFunc{TryWrapNumericScanPlan}, m.TryWrapScanPlanFuncs...)

	m.RegisterType(&pgtype.Type{
		Name:  "numeric",
		OID:   pgtype.NumericOID,
		Codec: NumericCodec{},
	})

	registerDefaultPgTypeVariants := func(name, arrayName string, value interface{}) {
		// T
		m.RegisterDefaultPgType(value, name)

		// *T
		valueType := reflect.TypeOf(value)
		m.RegisterDefaultPgType(reflect.New(valueType).Interface(), name)

		// []T
		sliceType := reflect.SliceOf(valueType)
		m.RegisterDefaultPgType(reflect.MakeSlice(sliceType, 0, 0).Interface(), arrayName)

		// *[]T
		m.RegisterDefaultPgType(reflect.New(sliceType).Interface(), arrayName)

		// []*T
		sliceOfPointerType := reflect.SliceOf(reflect.TypeOf(reflect.New(valueType).Interface()))
		m.RegisterDefaultPgType(reflect.MakeSlice(sliceOfPointerType, 0, 0).Interface(), arrayName)

		// *[]*T
		m.RegisterDefaultPgType(reflect.New(sliceOfPointerType).Interface(), arrayName)
	}

	registerDefaultPgTypeVariants("numeric", "_numeric", udecimal.Decimal{})
	registerDefaultPgTypeVariants("numeric", "_numeric", udecimal.NullDecimal{})
	registerDefaultPgTypeVariants("numeric", "_numeric", Decimal{})
	registerDefaultPgTypeVariants("numeric", "_numeric", NullDecimal{})
}
