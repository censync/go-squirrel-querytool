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
}*/
package querytool

import (
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/friendsofgo/errors"
	"strings"
)

var ErrUnknownField = errors.New("unknown_field")

const (
	globalDefaultLimit  = 100
	globalDefaultOffset = 0
)

type Scheme struct {
	Resolvers     map[string]FilterResolver
	DefaultOffset uint64
	DefaultLimit  uint64 // global default = globalDefaultLimit
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
		sorting    []string
	)
	and := squirrel.And{}

	for field, filter := range query.Filters {
		resolver, exists := scheme.Resolvers[field]
		if !exists {
			return errors.Wrap(ErrUnknownField, field)
		}

		expr, args, err := resolver(filter, field)
		if err != nil {
			return errors.Wrap(err, field)
		}

		and = append(and, squirrel.Expr(expr, args...))
		hasFilters = true
	}

	if hasFilters {
		*q = q.Where(and)
	}

	if _, ok := query.Sorting.([]string); ok {
		sorting = query.Sorting.([]string)
		for _, orderField := range sorting {
			field := orderField
			order := "DESC"

			if r := strings.Split(field, " "); len(r) == 2 &&
				(strings.ToUpper(r[1]) == "DESC" || strings.ToUpper(r[1]) == "ASC") {
				field = r[0]
				order = r[1]
			}

			_, exists := scheme.Resolvers[field]
			if !exists {
				return errors.Wrap(ErrUnknownField, orderField)
			}

			*q = q.OrderBy(fmt.Sprintf("%s %s", field, order))
		}
	} else if _, ok := query.Sorting.(map[string]string); ok {
		mapSorting := query.Sorting.(map[string]string)
		for orderField, orderDirection := range mapSorting {
			field := orderField
			order := "DESC"

			if strings.ToUpper(orderDirection) == "DESC" || strings.ToUpper(orderDirection) == "ASC" {
				order = orderDirection
			}

			_, exists := scheme.Resolvers[field]
			if !exists {
				return errors.Wrap(ErrUnknownField, orderField)
			}

			*q = q.OrderBy(fmt.Sprintf("%s %s", field, order))
		}
	}

	if len(sorting) == 0 {
		*q = q.OrderBy(scheme.DefaultSort...)
	}

	if query.Limit == 0 {
		*q = q.Limit(globalDefaultLimit)
	} else if query.Limit > 0 {
		*q = q.Limit(query.Limit)
	} else if scheme.DefaultLimit > 0 {
		*q = q.Limit(scheme.DefaultLimit)
	} else {
		*q = q.Limit(globalDefaultLimit)
	}

	if query.Offset > 0 {
		*q = q.Offset(query.Offset)
	} else if scheme.DefaultOffset > 0 {
		*q = q.Offset(scheme.DefaultOffset)
	} else if query.Offset < 0 {
		*q = q.Limit(globalDefaultOffset)
	}

	return nil
}
