# go-squirrel-querytool

Package for building dynamic SQL filters, sorting and pagination with [Masterminds/squirrel](https://github.com/Masterminds/squirrel).

Accepts filter queries from JSON body or URL query parameters and applies them to a `squirrel.SelectBuilder`.

## Install

```
go get github.com/censync/go-squirrel-querytool
```

## Query structure

```json
{
   "filters": {},
   "sorting": [],
   "limit": 100,
   "offset": 0
}
```

Every field can be omitted. Default values are applied from the `Scheme` configuration.

## Available resolvers

| Resolver    | Go variable | Supported operators                                     |
|-------------|-------------|---------------------------------------------------------|
| IntResolver | `Int`       | `=`, `!=`, `gt`, `gte`, `lt`, `lte`, `in`, `not in`, `null`, `not_null` |
| FloatResolver | `Float`   | `=`, `!=`, `gt`, `gte`, `lt`, `lte`, `in`, `not in`, `null`, `not_null` |
| StringResolver | `String` | `=`, `!=`, `like`, `in`, `not in`, `null`, `not_null`   |
| BoolResolver | `Boolean`  | `=`, `!=`, `null`, `not_null`                           |
| TimestampResolver | `Timestamp` | `=`, `!=`, `gt`, `gte`, `lt`, `lte`, `in`, `null`, `not_null` |

## Filtering operators

### Strict selection (equal / not equal)

```json
{"username": "john_doe"}
```

Same as:

```json
{"username": {"=": "john_doe"}}
```

Not equal:

```json
{"status": {"!=": 3}}
```

### List entry selection (in / not in)

```json
{"country": {"in": ["ru", "us", "es"]}}
{"status": {"not in": [0, 5]}}
```

### Comparison operators

| Operator | Meaning          | SQL equivalent |
|----------|------------------|----------------|
| `gt`     | greater than     | `>`            |
| `gte`    | greater or equal | `>=`           |
| `lt`     | less than        | `<`            |
| `lte`    | less or equal    | `<=`           |

```json
{"balance": {"gte": 100.0, "lt": 10000.0}}
{"age": {"gt": 18}}
```

### Wildcard string selection (like)

```json
{"username": {"like": "%super%"}}
{"phone": {"like": "+1213%"}}
```

### NULL check

```json
{"deleted_at": {"null": true}}
{"email": {"not_null": true}}
```

### Combining operators

Multiple operators on the same field are combined with AND:

```json
{"balance": {"gte": 10.0, "lte": 1000.0}}
```

Produces: `balance >= 10.0 AND balance <= 1000.0`

## Sorting

Sorting accepts a string array with `"column direction"` format or a map `{"column": "direction"}`. Do not mix the two styles.

Array style:

```json
{"sorting": ["id DESC", "balance ASC"]}
```

Map style:

```json
{"sorting": {"id": "DESC", "balance": "ASC"}}
```

If no direction is given, `DESC` is used by default.

## Scheme configuration

```go
scheme := &querytool.Scheme{
    Resolvers: map[string]querytool.FilterResolver{
        "user_id":    querytool.Int,
        "username":   querytool.String,
        "email":      querytool.String,
        "balance":    querytool.Float,
        "is_active":  querytool.Boolean,
        "created_at": querytool.Timestamp,
    },
    DefaultLimit:  50,       // applied when client sends no limit (global default: 100)
    MaxLimit:      500,      // caps any client-requested limit to this value
    DefaultOffset: 0,
    DefaultSort:   []string{"created_at DESC"},
}
```

Only fields listed in `Resolvers` are allowed in filters and sorting. Unknown fields return an error.

`MaxLimit` prevents clients from requesting excessively large result sets (e.g. `limit: 999999`).

## Examples

### Define a query form

```go
package requests

import (
    querytool "github.com/censync/go-squirrel-querytool"
)

type UsersListQueryForm struct {
    querytool.Query
}
```

### Bind and apply a JSON request (Echo framework)

```go
func GetUsersList(c echo.Context) error {
    request := &UsersListQueryForm{}

    if err := c.Bind(request); err != nil {
        return err
    }

    scheme := &querytool.Scheme{
        Resolvers: map[string]querytool.FilterResolver{
            "user_id":    querytool.Int,
            "username":   querytool.String,
            "balance":    querytool.Float,
            "is_active":  querytool.Boolean,
            "created_at": querytool.Timestamp,
        },
        DefaultLimit: 50,
        MaxLimit:     200,
        DefaultSort:  []string{"created_at DESC"},
    }

    q := squirrel.Select("user_id", "username", "balance").From("users")

    if err := querytool.ApplyQuery(&q, scheme, &request.Query); err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }

    sql, args, _ := q.ToSql()
    // sql:  SELECT user_id, username, balance FROM users WHERE ... ORDER BY ... LIMIT ... OFFSET ...
    // args: [...]

    rows, err := db.Query(sql, args...)
    // ...
}
```

### Bind URL query parameters

```go
func GetUsersList(c echo.Context) error {
    request := &UsersListQueryForm{}

    if err := request.BindQuery(c.QueryParams()); err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }

    // ... same ApplyQuery logic as above
}
```

### URL query parameter format

```
GET /users?filters[username][like]=%john%&filters[is_active]=true&filters[balance][gte]=100&sort[]=created_at+DESC&limit=50&offset=0
```

### Example JSON request

```json
{
   "filters": {
      "phone": {"like": "+1213%"},
      "status": {"in": [1, 2, 3]},
      "firstname": {"in": ["John", "Jake"]},
      "balance": {"gte": 100.0, "lte": 5000.0},
      "deleted_at": {"null": true}
   },
   "sorting": ["created_at DESC", "balance ASC"],
   "limit": 50,
   "offset": 0
}
```

### Direct usage without a web framework

```go
package main

import (
    "encoding/json"
    "fmt"

    "github.com/Masterminds/squirrel"
    querytool "github.com/censync/go-squirrel-querytool"
)

func main() {
    jsonQuery := `{
        "filters": {
            "user_id": {"gte": 10, "lte": 100},
            "username": {"like": "john%"},
            "is_active": true
        },
        "sorting": ["user_id ASC"],
        "limit": 25
    }`

    query := &querytool.Query{}
    json.Unmarshal([]byte(jsonQuery), query)

    scheme := &querytool.Scheme{
        Resolvers: map[string]querytool.FilterResolver{
            "user_id":   querytool.Int,
            "username":  querytool.String,
            "is_active": querytool.Boolean,
        },
        DefaultLimit: 50,
        MaxLimit:     200,
        DefaultSort:  []string{"user_id DESC"},
    }

    q := squirrel.Select("*").From("users")
    if err := querytool.ApplyQuery(&q, scheme, query); err != nil {
        panic(err)
    }

    sql, args, _ := q.ToSql()
    fmt.Println("SQL:", sql)
    fmt.Println("Args:", args)
    // SQL:  SELECT * FROM users WHERE (is_active = ? AND (user_id >= ? AND user_id <= ?) AND username LIKE ?) ORDER BY user_id ASC LIMIT 25
    // Args: [true 10 100 john%]
}
```

### Error handling

```go
// Unknown field
err := querytool.ApplyQuery(&q, scheme, query)
if errors.Is(err, querytool.ErrUnknownField) {
    // client sent a filter or sort field not in Resolvers
}

// Wrong type
if errors.Is(err, querytool.ErrWrongType) {
    // e.g. string value passed to Int resolver
}

// Unknown operator (StringResolver)
if errors.Is(err, querytool.ErrUnknownOperator) {
    // e.g. {"name": {"bogus": "x"}}
}
```

## Custom resolvers

You can define custom resolvers by implementing the `FilterResolver` function signature:

```go
type FilterResolver func(arg interface{}, label string) (string, []interface{}, error)
```

Example -- an ILIKE resolver for case-insensitive search (PostgreSQL):

```go
func ILikeResolver(arg interface{}, label string) (string, []interface{}, error) {
    pattern, ok := arg.(string)
    if !ok {
        if m, ok := arg.(map[string]interface{}); ok {
            if v, ok := m["ilike"]; ok {
                pattern, ok = v.(string)
                if !ok {
                    return "", nil, querytool.ErrWrongType
                }
            }
        }
        if pattern == "" {
            return "", nil, querytool.ErrWrongType
        }
    }
    return fmt.Sprintf("%s ILIKE ?", label), []interface{}{pattern}, nil
}

// Usage:
scheme := &querytool.Scheme{
    Resolvers: map[string]querytool.FilterResolver{
        "username": ILikeResolver,
    },
}
```