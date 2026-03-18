package querytool

import (
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
)

var (
	ErrWrongType       = errors.New("wrong_type")
	ErrUnknownOperator = errors.New("unknown_operator")
)

var (
	Int       = IntResolver{}.ToExpr
	Float     = FloatResolver{}.ToExpr
	String    = StringResolver{}.ToExpr
	Boolean   = BoolResolver{}.ToExpr
	Timestamp = TimestampResolver{}.ToExpr
)

type FilterResolver func(arg interface{}, label string) (string, []interface{}, error)

// scalarOp applies a single-value comparison operator to the AND clause.
// Accepts the operator name and the expected Go type for validation.
func scalarOp[T comparable](and *squirrel.And, m map[string]interface{}, op string, label string, mkSqlizer func(string, T) squirrel.Sqlizer) error {
	val, ok := m[op]
	if !ok {
		return nil
	}
	typed, ok := val.(T)
	if !ok {
		return ErrWrongType
	}
	*and = append(*and, mkSqlizer(label, typed))
	return nil
}

func eq[T comparable](label string, v T) squirrel.Sqlizer    { return squirrel.Eq{label: v} }
func notEq[T comparable](label string, v T) squirrel.Sqlizer { return squirrel.NotEq{label: v} }
func gt[T comparable](label string, v T) squirrel.Sqlizer    { return squirrel.Gt{label: v} }
func gtEq[T comparable](label string, v T) squirrel.Sqlizer  { return squirrel.GtOrEq{label: v} }
func lt[T comparable](label string, v T) squirrel.Sqlizer    { return squirrel.Lt{label: v} }
func ltEq[T comparable](label string, v T) squirrel.Sqlizer  { return squirrel.LtOrEq{label: v} }

// sliceOp extracts a []interface{} value from m[op], converts each element via conv,
// and appends the appropriate Eq or NotEq clause.
func sliceOp[T any](and *squirrel.And, m map[string]interface{}, op string, label string, conv func(interface{}) (T, bool), negate bool) error {
	raw, ok := m[op]
	if !ok {
		return nil
	}
	sl, ok := raw.([]interface{})
	if !ok {
		return ErrWrongType
	}
	arr := make([]T, 0, len(sl))
	for _, v := range sl {
		if typed, ok := conv(v); ok {
			arr = append(arr, typed)
		}
	}
	if negate {
		*and = append(*and, squirrel.NotEq{label: arr})
	} else {
		*and = append(*and, squirrel.Eq{label: arr})
	}
	return nil
}

// nullOp handles IS NULL / IS NOT NULL operators.
func nullOp(and *squirrel.And, m map[string]interface{}, label string) {
	if _, ok := m["null"]; ok {
		*and = append(*and, squirrel.Eq{label: nil})
	}
	if _, ok := m["not_null"]; ok {
		*and = append(*and, squirrel.NotEq{label: nil})
	}
}

// numericOps applies all standard numeric comparison operators (=, !=, gt, gte, lt, lte).
func numericOps[T comparable](and *squirrel.And, m map[string]interface{}, label string) error {
	ops := []struct {
		key string
		fn  func(string, T) squirrel.Sqlizer
	}{
		{"=", eq[T]}, {"!=", notEq[T]},
		{"gt", gt[T]}, {"gte", gtEq[T]},
		{"lt", lt[T]}, {"lte", ltEq[T]},
	}
	for _, o := range ops {
		if err := scalarOp(and, m, o.key, label, o.fn); err != nil {
			return err
		}
	}
	return nil
}

func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	default:
		return 0, false
	}
}

func toFloat64(v interface{}) (float64, bool) {
	n, ok := v.(float64)
	return n, ok
}

func toString(v interface{}) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

// ------- Resolvers -------

type IntResolver struct{}

func (IntResolver) ToExpr(arg interface{}, label string) (string, []interface{}, error) {
	switch v := arg.(type) {
	case float64:
		return squirrel.Eq{label: int64(v)}.ToSql()
	case map[string]interface{}:
		and := squirrel.And{}
		if err := numericOps[float64](&and, v, label); err != nil {
			return "", nil, err
		}
		if err := sliceOp(&and, v, "in", label, toInt64, false); err != nil {
			return "", nil, err
		}
		if err := sliceOp(&and, v, "not in", label, toInt64, true); err != nil {
			return "", nil, err
		}
		nullOp(&and, v, label)
		return and.ToSql()
	default:
		return "", nil, ErrWrongType
	}
}

type FloatResolver struct{}

func (FloatResolver) ToExpr(arg interface{}, label string) (string, []interface{}, error) {
	switch v := arg.(type) {
	case float64:
		return squirrel.Eq{label: v}.ToSql()
	case map[string]interface{}:
		and := squirrel.And{}
		if err := numericOps[float64](&and, v, label); err != nil {
			return "", nil, err
		}
		if err := sliceOp(&and, v, "in", label, toFloat64, false); err != nil {
			return "", nil, err
		}
		if err := sliceOp(&and, v, "not in", label, toFloat64, true); err != nil {
			return "", nil, err
		}
		nullOp(&and, v, label)
		return and.ToSql()
	default:
		return "", nil, ErrWrongType
	}
}

type StringResolver struct{}

func (StringResolver) ToExpr(arg interface{}, label string) (string, []interface{}, error) {
	switch v := arg.(type) {
	case string:
		return squirrel.Eq{label: v}.ToSql()
	case map[string]interface{}:
		if pattern, ok := v["like"]; ok {
			return squirrel.Expr(fmt.Sprintf("%s LIKE ?", label), pattern).ToSql()
		}

		and := squirrel.And{}
		if err := scalarOp(&and, v, "=", label, eq[string]); err != nil {
			return "", nil, err
		}
		if err := scalarOp(&and, v, "!=", label, notEq[string]); err != nil {
			return "", nil, err
		}
		if err := sliceOp(&and, v, "in", label, toString, false); err != nil {
			return "", nil, err
		}
		if err := sliceOp(&and, v, "not in", label, toString, true); err != nil {
			return "", nil, err
		}
		nullOp(&and, v, label)

		if len(and) == 0 {
			return "", nil, fmt.Errorf("%w: no recognized operator for field %q", ErrUnknownOperator, label)
		}
		return and.ToSql()
	default:
		return "", nil, ErrWrongType
	}
}

type BoolResolver struct{}

func (BoolResolver) ToExpr(arg interface{}, label string) (string, []interface{}, error) {
	switch v := arg.(type) {
	case bool:
		return squirrel.Eq{label: v}.ToSql()
	case map[string]interface{}:
		and := squirrel.And{}
		if err := scalarOp(&and, v, "=", label, eq[bool]); err != nil {
			return "", nil, err
		}
		if err := scalarOp(&and, v, "!=", label, notEq[bool]); err != nil {
			return "", nil, err
		}
		nullOp(&and, v, label)
		return and.ToSql()
	default:
		return "", nil, ErrWrongType
	}
}

type TimestampResolver struct{}

func (TimestampResolver) ToExpr(arg interface{}, label string) (string, []interface{}, error) {
	switch v := arg.(type) {
	case float64:
		return squirrel.Eq{label: time.Unix(int64(v), 0)}.ToSql()
	case map[string]interface{}:
		and := squirrel.And{}
		if err := numericOps[string](&and, v, label); err != nil {
			return "", nil, err
		}
		if err := sliceOp(&and, v, "in", label, toString, false); err != nil {
			return "", nil, err
		}
		nullOp(&and, v, label)
		return and.ToSql()
	default:
		return "", nil, ErrWrongType
	}
}
