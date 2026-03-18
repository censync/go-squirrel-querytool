/*
query example:

	{
	   "filters":{
	      "phone":{
	         "like":"+7%"
	      },
	      "sex":0,
	      "firstname":{
	         "in":[
	            "firstname",
	            ""
	         ]
	      }
	   },
	   "sort":[
	      "created_at ASC"
	   ],
	   "limit":200,
	   "offset":0
	}
*/
package querytool

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Masterminds/squirrel"
)

var ErrUnknownField = fmt.Errorf("unknown_field")

const (
	globalDefaultLimit  = 100
	globalDefaultOffset = 0
)

var availableOperators = map[string]bool{
	"=":        true,
	"!=":       true,
	">":        true,
	"<":        true,
	"gt":       true,
	"lt":       true,
	"gte":      true,
	"lte":      true,
	"in":       true,
	"not in":   true,
	"like":     true,
	"null":     true,
	"not null": true,
}

type Scheme struct {
	Resolvers     map[string]FilterResolver
	DefaultOffset uint64
	DefaultLimit  uint64 // global default = globalDefaultLimit
	MaxLimit      uint64 // if set, caps the user-requested limit
	DefaultSort   []string
}

type Query struct {
	Filters map[string]interface{} `json:"filters"`
	Sorting interface{}            `json:"sorting"`
	Limit   uint64                 `json:"limit"`
	Offset  uint64                 `json:"offset"`
}

func ApplyQuery(q *squirrel.SelectBuilder, scheme *Scheme, query *Query) error {
	var (
		hasFilters bool
		hasSorting bool
	)
	and := squirrel.And{}

	for field, filter := range query.Filters {
		resolver, exists := scheme.Resolvers[field]
		if !exists {
			return fmt.Errorf("%w: %s", ErrUnknownField, field)
		}

		expr, args, err := resolver(filter, field)
		if err != nil {
			return fmt.Errorf("field %s: %w", field, err)
		}

		and = append(and, squirrel.Expr(expr, args...))
		hasFilters = true
	}

	if hasFilters {
		*q = q.Where(and)
	}

	// Normalize sorting: JSON unmarshal produces []interface{} and map[string]interface{},
	// so we convert them to the expected Go types.
	switch s := query.Sorting.(type) {
	case []interface{}:
		strs := make([]string, 0, len(s))
		for _, v := range s {
			if str, ok := v.(string); ok {
				strs = append(strs, str)
			}
		}
		query.Sorting = strs
	case map[string]interface{}:
		m := make(map[string]string, len(s))
		for k, v := range s {
			if str, ok := v.(string); ok {
				m[k] = str
			}
		}
		query.Sorting = m
	}

	if sortSlice, ok := query.Sorting.([]string); ok {
		hasSorting = true
		for _, orderField := range sortSlice {
			field := orderField
			order := "DESC"

			if r := strings.Split(field, " "); len(r) == 2 &&
				(strings.ToUpper(r[1]) == "DESC" || strings.ToUpper(r[1]) == "ASC") {
				field = r[0]
				order = r[1]
			}

			_, exists := scheme.Resolvers[field]
			if !exists {
				return fmt.Errorf("%w: %s", ErrUnknownField, orderField)
			}

			*q = q.OrderBy(fmt.Sprintf("%s %s", field, order))
		}
	} else if mapSorting, ok := query.Sorting.(map[string]string); ok {
		hasSorting = true
		for orderField, orderDirection := range mapSorting {
			field := orderField
			order := "DESC"

			if strings.ToUpper(orderDirection) == "DESC" || strings.ToUpper(orderDirection) == "ASC" {
				order = orderDirection
			}

			_, exists := scheme.Resolvers[field]
			if !exists {
				return fmt.Errorf("%w: %s", ErrUnknownField, orderField)
			}

			*q = q.OrderBy(fmt.Sprintf("%s %s", field, order))
		}
	}

	if !hasSorting {
		*q = q.OrderBy(scheme.DefaultSort...)
	}

	limit := query.Limit
	if limit == 0 {
		if scheme.DefaultLimit > 0 {
			limit = scheme.DefaultLimit
		} else {
			limit = globalDefaultLimit
		}
	}
	if scheme.MaxLimit > 0 && limit > scheme.MaxLimit {
		limit = scheme.MaxLimit
	}
	*q = q.Limit(limit)

	if query.Offset > 0 {
		*q = q.Offset(query.Offset)
	} else if scheme.DefaultOffset > 0 {
		*q = q.Offset(scheme.DefaultOffset)
	}

	return nil
}

// Parse query from url values
func (f *Query) BindQuery(params url.Values) (err error) {
	for param, values := range params {
		if f.isArray(param) {
			rootKey, rootSuffix := f.getRootKey(param)
			if rootKey == "filters" {
				err = f.parseFilters(rootSuffix, values)
			} else if rootKey == "sort" {
				err = f.parseSorting(rootSuffix, values)
			}
			if err != nil {
				return
			}
		} else if param == "limit" && len(values) == 1 {
			f.Limit, _ = strconv.ParseUint(values[0], 10, 64)
		} else if param == "offset" {
			f.Offset, _ = strconv.ParseUint(values[0], 10, 64)
		}
	}

	return
}

func (q *Query) parseFilters(param string, values []string) (err error) {
	if len(values) == 0 {
		return
	}
	if q.Filters == nil {
		q.Filters = map[string]interface{}{}
	}

	exists, filterField, filterValues := q.getNested(param)
	if exists {
		if q.Filters[filterField] == nil {
			q.Filters[filterField] = map[string]interface{}{}
		}
	}

	tmpFiltersByField := q.Filters[filterField].(map[string]interface{})

	if q.isArray(filterValues) {
		exists, filterOperator, _ := q.getNested(filterValues)
		if !exists {
			return
		}

		if _, ok := availableOperators[filterOperator]; !ok {
			return fmt.Errorf("filter operator %q not supported", filterOperator)
		}

		if tmpFiltersByField[filterOperator] == nil {
			tmpFiltersByField[filterOperator] = map[string]interface{}{}
		}

		if filterOperator == "in" {
			tmpFiltersByField[filterOperator] = values
		} else {
			if len(values) > 1 {
				return fmt.Errorf("filter operator %q does not support array values", filterOperator)
			}
			tmpFiltersByField[filterOperator] = values[0]
		}

	} else {
		if len(values) > 1 {
			return fmt.Errorf("strict filter does not support array values")
		}
		tmpFiltersByField["="] = values[0]
	}

	q.Filters[filterField] = tmpFiltersByField

	return
}

func (q *Query) parseSorting(param string, values []string) (err error) {
	if len(values) == 0 {
		return
	}
	if param == "[]" {
		q.Sorting = values
	} else {
		if len(values) == 0 {
			return
		}
		if q.Sorting == nil {
			q.Sorting = map[string]string{}
		}

		exists, sortingField, _ := q.getNested(param)
		if exists {
			if _, ok := q.Sorting.(map[string]string); !ok {
				return fmt.Errorf("mixed sorting types is not supported")
			}

			if len(values) == 0 {
				return fmt.Errorf("sorting does not support array values")
			}

			q.Sorting.(map[string]string)[sortingField] = values[0]
		}
	}
	return
}

func (Query) getRootKey(src string) (key string, suffix string) {
	startIdx := strings.Index(src, "[")
	if startIdx != -1 {
		key = src[:startIdx]
		suffix = src[startIdx:]
	}
	return
}

func (Query) getNested(src string) (exists bool, key string, suffix string) {
	startIdx := strings.Index(src, "[")
	if startIdx != -1 {
		endIdx := strings.Index(src[startIdx:], "]")
		if endIdx != -1 {
			exists = true
			key = src[startIdx+1 : startIdx+endIdx]
			suffix = src[endIdx+1:]
		}
	}
	return
}

func (Query) isArray(src string) bool {
	startIdx := strings.Index(src, "[")
	endIdx := strings.Index(src, "]")
	return startIdx > -1 && endIdx > -1 && endIdx > startIdx
}
