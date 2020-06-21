package querytool

import (
	"errors"
	"github.com/lib/pq"
	"reflect"
	"time"

	"fmt"

	"github.com/Masterminds/squirrel"
)

var (
	ErrWrongType = errors.New("wrong_type")
)

var (
	Int       = IntResolver{}.ToExpr
	Float     = FloatResolver{}.ToExpr
	String    = StringResolver{}.ToExpr
	Boolean   = BoolResolver{}.ToExpr
	Timestamp = BoolResolver{}.ToExpr
)

type FilterResolver func(arg interface{}, label string) (string, []interface{}, error)

type IntResolver struct{}

func (ir IntResolver) ToExpr(arg interface{}, label string) (string, []interface{}, error) {
	value := reflect.ValueOf(arg)
	switch value.Kind() {
	case reflect.Float64:
		return squirrel.Eq{label: int64(value.Float())}.ToSql()
	case reflect.Map:
		and := squirrel.And{}

		m, ok := arg.(map[string]interface{})
		if !ok {
			return "", nil, ErrWrongType
		}

		if _, ok := m["="]; ok {
			sl, ok := m["="].(float64)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.Eq{label: sl})
		}

		if _, ok := m["!="]; ok {
			sl, ok := m["!="].(float64)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.NotEq{label: sl})
		}

		if _, ok := m["in"]; ok {
			sl, ok := m["in"].([]interface{})
			if !ok {
				return "", nil, ErrWrongType
			}

			arr := make([]int64, 0)

			for _, val := range sl {
				s, ok := val.(int64)
				if ok {
					arr = append(arr, s)
				}
			}

			and = append(and, squirrel.Eq{label: arr})
		}

		if _, ok := m["gt"]; ok {
			sl, ok := m["gt"].(float64)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.Gt{label: sl})
		}

		if _, ok := m["gte"]; ok {
			sl, ok := m["gte"].(float64)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.GtOrEq{label: sl})
		}

		if _, ok := m["lt"]; ok {
			sl, ok := m["lt"].(float64)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.Lt{label: sl})
		}

		if _, ok := m["lte"]; ok {
			sl, ok := m["lte"].(float64)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.LtOrEq{label: sl})
		}

		return and.ToSql()
	default:
		return "", nil, ErrWrongType
	}
}

type FloatResolver struct{}

func (fr FloatResolver) ToExpr(arg interface{}, label string) (string, []interface{}, error) {
	value := reflect.ValueOf(arg)

	switch value.Kind() {
	case reflect.Float64:
		return squirrel.Eq{label: value.Float()}.ToSql()
	case reflect.Map:
		and := squirrel.And{}

		m, ok := arg.(map[string]interface{})
		if !ok {
			return "", nil, ErrWrongType
		}

		if _, ok := m["in"]; ok {
			sl, ok := m["in"].([]interface{})
			if !ok {
				return "", nil, ErrWrongType
			}

			arr := make([]float64, 0)

			for _, val := range sl {
				s, ok := val.(float64)
				if ok {
					arr = append(arr, s)
				}
			}

			and = append(and, squirrel.Eq{label: arr})
		}

		if _, ok := m["="]; ok {
			sl, ok := m["="].(float64)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.Eq{label: sl})
		}

		if _, ok := m["not in"]; ok {
			sl, ok := m["not in"].([]interface{})
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.NotEq{label: sl})
		}

		if _, ok := m["!="]; ok {
			sl, ok := m["!="].(float64)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.NotEq{label: sl})
		}

		if _, ok := m["gt"]; ok {
			sl, ok := m["gt"].(float64)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.Gt{label: sl})
		}

		if _, ok := m["gte"]; ok {
			sl, ok := m["gte"].(float64)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.GtOrEq{label: sl})
		}

		if _, ok := m["lt"]; ok {
			sl, ok := m["lt"].(float64)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.Lt{label: sl})
		}

		if _, ok := m["lte"]; ok {
			sl, ok := m["lte"].(float64)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.LtOrEq{label: sl})
		}

		return and.ToSql()
	default:
		return "", nil, ErrWrongType
	}
}

type StringResolver struct{}

func (sr StringResolver) ToExpr(arg interface{}, label string) (string, []interface{}, error) {
	value := reflect.ValueOf(arg)

	switch value.Kind() {
	case reflect.String:
		return squirrel.Eq{label: value.String()}.ToSql()
	case reflect.Map:
		m, ok := arg.(map[string]interface{})
		if !ok {
			return "", nil, ErrWrongType
		}

		if i, ok := m["like"]; ok {
			return squirrel.Expr(fmt.Sprintf("%s LIKE ?", label), i).ToSql()
		}

		if _, ok := m["in"]; ok {
			sl, ok := m["in"].([]interface{})
			if !ok {
				return "", nil, ErrWrongType
			}

			arr := make([]string, 0)

			for _, val := range sl {
				s, ok := val.(string)
				if ok {
					arr = append(arr, s)
				}
			}

			return squirrel.Eq{label: pq.StringArray(arr)}.ToSql()
		}

		if _, ok := m["="]; ok {
			sl, ok := m["="].(string)
			if !ok {
				return "", nil, ErrWrongType
			}

			return squirrel.Eq{label: sl}.ToSql()
		}

		if _, ok := m["!="]; ok {
			sl, ok := m["!="].(string)
			if !ok {
				return "", nil, ErrWrongType
			}

			return squirrel.NotEq{label: sl}.ToSql()
		}

		if _, ok := m["not in"]; ok {
			sl, ok := m["not in"].([]interface{})
			if !ok {
				return "", nil, ErrWrongType
			}

			return squirrel.NotEq{label: sl}.ToSql()
		}

		return "", nil, nil
	default:
		return "", nil, ErrWrongType
	}
}

type BoolResolver struct{}

func (fr BoolResolver) ToExpr(arg interface{}, label string) (string, []interface{}, error) {
	value := reflect.ValueOf(arg)

	switch value.Kind() {
	case reflect.Bool:
		return squirrel.Eq{label: value.Bool()}.ToSql()
	case reflect.Map:
		and := squirrel.And{}

		m, ok := arg.(map[string]interface{})
		if !ok {
			return "", nil, ErrWrongType
		}

		if _, ok := m["="]; ok {
			sl, ok := m["="].(bool)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.Eq{label: sl})
		}

		if _, ok := m["!="]; ok {
			sl, ok := m["!="].(bool)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.NotEq{label: sl})
		}

		return and.ToSql()
	default:
		return "", nil, ErrWrongType
	}
}

type TimestampResolver struct{}

func (ir TimestampResolver) ToExpr(arg interface{}, label string) (string, []interface{}, error) {
	value := reflect.ValueOf(arg)
	switch value.Kind() {
	case reflect.Float64:
		// Timestamp to time
		return squirrel.Eq{label: time.Unix(int64(value.Float()), 0)}.ToSql()
	case reflect.Map:
		and := squirrel.And{}

		m, ok := arg.(map[string]interface{})
		if !ok {
			return "", nil, ErrWrongType
		}

		if _, ok := m["="]; ok {
			sl, ok := m["="].(string)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.Eq{label: sl})
		}

		if _, ok := m["!="]; ok {
			sl, ok := m["!="].(string)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.NotEq{label: sl})
		}

		if _, ok := m["in"]; ok {
			sl, ok := m["in"].([]interface{})
			if !ok {
				return "", nil, ErrWrongType
			}

			arr := make([]string, 0)

			for _, val := range sl {
				s, ok := val.(string)
				if ok {
					arr = append(arr, s)
				}
			}

			and = append(and, squirrel.Eq{label: pq.StringArray(arr)})
		}

		if _, ok := m["gt"]; ok {
			sl, ok := m["gt"].(string)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.Gt{label: sl})
		}

		if _, ok := m["gte"]; ok {
			sl, ok := m["gte"].(string)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.GtOrEq{label: sl})
		}

		if _, ok := m["lt"]; ok {
			sl, ok := m["lt"].(string)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.Lt{label: sl})
		}

		if _, ok := m["lte"]; ok {
			sl, ok := m["lte"].(string)
			if !ok {
				return "", nil, ErrWrongType
			}

			and = append(and, squirrel.LtOrEq{label: sl})
		}

		return and.ToSql()
	default:
		return "", nil, ErrWrongType
	}
}
