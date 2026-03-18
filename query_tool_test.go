package querytool

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/Masterminds/squirrel"
)

// helper to build SQL from a JSON query string
func buildSQL(t *testing.T, scheme *Scheme, jsonQuery string) (string, []interface{}) {
	t.Helper()
	query := &Query{}
	if err := json.Unmarshal([]byte(jsonQuery), query); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	q := squirrel.Select("*").From("users")
	if err := ApplyQuery(&q, scheme, query); err != nil {
		t.Fatalf("ApplyQuery: %v", err)
	}
	sql, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("ToSql: %v", err)
	}
	return sql, args
}

func defaultScheme() *Scheme {
	return &Scheme{
		Resolvers: map[string]FilterResolver{
			"user_id":    Int,
			"name":       String,
			"flag":       Boolean,
			"balance":    Float,
			"created_at": Timestamp,
		},
		DefaultSort: []string{"user_id DESC"},
	}
}

// ============================
// IntResolver tests
// ============================

func TestIntResolver_EqualDirect(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"user_id": 42}}`)
	if !strings.Contains(sql, "user_id = ?") {
		t.Errorf("expected 'user_id = ?', got: %s", sql)
	}
	if len(args) < 1 || args[0] != int64(42) {
		t.Errorf("expected arg 42, got: %v", args)
	}
}

func TestIntResolver_EqualOperator(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"user_id": {"=": 10}}}`)
	if !strings.Contains(sql, "user_id = ?") {
		t.Errorf("expected 'user_id = ?', got: %s", sql)
	}
	if len(args) < 1 {
		t.Fatalf("expected args, got none")
	}
	if v, ok := args[0].(float64); !ok || v != 10 {
		t.Errorf("expected arg 10, got: %v", args)
	}
}

func TestIntResolver_NotEqual(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"user_id": {"!=": 5}}}`)
	if !strings.Contains(sql, "user_id <> ?") {
		t.Errorf("expected 'user_id <> ?', got: %s", sql)
	}
	if len(args) < 1 {
		t.Fatalf("expected args, got none")
	}
	if v, ok := args[0].(float64); !ok || v != 5 {
		t.Errorf("expected arg 5, got: %v", args)
	}
}

func TestIntResolver_In(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"user_id": {"in": [1, 2, 3]}}}`)
	if !strings.Contains(sql, "user_id IN (?,?,?)") {
		t.Errorf("expected 'user_id IN (?,?,?)', got: %s", sql)
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got: %d (%v)", len(args), args)
	}
	// After fix: JSON float64 values should be converted to int64
	for i, expected := range []int64{1, 2, 3} {
		if args[i] != expected {
			t.Errorf("arg[%d]: expected %d, got: %v (%T)", i, expected, args[i], args[i])
		}
	}
}

func TestIntResolver_NotIn(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"user_id": {"not in": [10, 20]}}}`)
	if !strings.Contains(sql, "user_id NOT IN (?,?)") {
		t.Errorf("expected 'user_id NOT IN (?,?)', got: %s", sql)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got: %d", len(args))
	}
}

func TestIntResolver_Gt(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"user_id": {"gt": 100}}}`)
	if !strings.Contains(sql, "user_id > ?") {
		t.Errorf("expected 'user_id > ?', got: %s", sql)
	}
	if len(args) < 1 {
		t.Fatalf("expected args, got none")
	}
}

func TestIntResolver_Gte(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"user_id": {"gte": 50}}}`)
	if !strings.Contains(sql, "user_id >= ?") {
		t.Errorf("expected 'user_id >= ?', got: %s", sql)
	}
	if len(args) < 1 {
		t.Fatalf("expected args")
	}
}

func TestIntResolver_Lt(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"user_id": {"lt": 10}}}`)
	if !strings.Contains(sql, "user_id < ?") {
		t.Errorf("expected 'user_id < ?', got: %s", sql)
	}
	if len(args) < 1 {
		t.Fatalf("expected args")
	}
}

func TestIntResolver_Lte(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"user_id": {"lte": 10}}}`)
	if !strings.Contains(sql, "user_id <= ?") {
		t.Errorf("expected 'user_id <= ?', got: %s", sql)
	}
	if len(args) < 1 {
		t.Fatalf("expected args")
	}
}

func TestIntResolver_WrongType(t *testing.T) {
	query := &Query{}
	_ = json.Unmarshal([]byte(`{"filters":{"user_id": "not_a_number"}}`), query)
	q := squirrel.Select("*").From("t")
	err := ApplyQuery(&q, defaultScheme(), query)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestIntResolver_CombinedOperators(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"user_id": {"gte": 10, "lte": 100}}}`)
	if !strings.Contains(sql, "user_id >= ?") || !strings.Contains(sql, "user_id <= ?") {
		t.Errorf("expected range condition, got: %s", sql)
	}
	if len(args) < 2 {
		t.Errorf("expected at least 2 args, got: %d", len(args))
	}
}

// ============================
// FloatResolver tests
// ============================

func TestFloatResolver_EqualDirect(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"balance": 99.5}}`)
	if !strings.Contains(sql, "balance = ?") {
		t.Errorf("expected 'balance = ?', got: %s", sql)
	}
	if len(args) < 1 || args[0] != 99.5 {
		t.Errorf("expected arg 99.5, got: %v", args)
	}
}

func TestFloatResolver_Gte(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"balance": {"gte": 4.32}}}`)
	if !strings.Contains(sql, "balance >= ?") {
		t.Errorf("expected 'balance >= ?', got: %s", sql)
	}
	if len(args) < 1 {
		t.Fatalf("expected args")
	}
}

func TestFloatResolver_In(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"balance": {"in": [1.1, 2.2]}}}`)
	if !strings.Contains(sql, "balance IN (?,?)") {
		t.Errorf("expected 'balance IN (?,?)', got: %s", sql)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got: %d", len(args))
	}
}

func TestFloatResolver_NotIn(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"balance": {"not in": [0.0, 1.0]}}}`)
	if !strings.Contains(sql, "balance NOT IN (?,?)") {
		t.Errorf("expected 'balance NOT IN', got: %s", sql)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got: %d", len(args))
	}
}

func TestFloatResolver_NotEqual(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"balance": {"!=": 0}}}`)
	if !strings.Contains(sql, "balance <> ?") {
		t.Errorf("expected 'balance <> ?', got: %s", sql)
	}
}

func TestFloatResolver_Range(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"balance": {"gt": 10, "lt": 100}}}`)
	if !strings.Contains(sql, "balance > ?") || !strings.Contains(sql, "balance < ?") {
		t.Errorf("expected range, got: %s", sql)
	}
	if len(args) < 2 {
		t.Errorf("expected 2+ args, got: %d", len(args))
	}
}

// ============================
// StringResolver tests
// ============================

func TestStringResolver_EqualDirect(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"name": "john"}}`)
	if !strings.Contains(sql, "name = ?") {
		t.Errorf("expected 'name = ?', got: %s", sql)
	}
	if len(args) < 1 || args[0] != "john" {
		t.Errorf("expected arg 'john', got: %v", args)
	}
}

func TestStringResolver_EqualOperator(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"name": {"=": "john"}}}`)
	if !strings.Contains(sql, "name = ?") {
		t.Errorf("expected 'name = ?', got: %s", sql)
	}
}

func TestStringResolver_NotEqual(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"name": {"!=": "admin"}}}`)
	if !strings.Contains(sql, "name <> ?") {
		t.Errorf("expected 'name <> ?', got: %s", sql)
	}
}

func TestStringResolver_Like(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"name": {"like": "%john%"}}}`)
	if !strings.Contains(sql, "name LIKE ?") {
		t.Errorf("expected 'name LIKE ?', got: %s", sql)
	}
	if len(args) < 1 || args[0] != "%john%" {
		t.Errorf("expected arg '%%john%%', got: %v", args)
	}
}

func TestStringResolver_In(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"name": {"in": ["alice", "bob"]}}}`)
	// After fix: should generate IN (?,?) not PG array literal
	if !strings.Contains(sql, "name IN (?,?)") {
		t.Errorf("expected 'name IN (?,?)', got: %s", sql)
	}
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got: %d (%v)", len(args), args)
	}
	if args[0] != "alice" || args[1] != "bob" {
		t.Errorf("expected [alice, bob], got: %v", args)
	}
}

func TestStringResolver_NotIn(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"name": {"not in": ["x", "y"]}}}`)
	if !strings.Contains(sql, "name NOT IN (?,?)") {
		t.Errorf("expected 'name NOT IN', got: %s", sql)
	}
}

func TestStringResolver_WrongType(t *testing.T) {
	query := &Query{}
	_ = json.Unmarshal([]byte(`{"filters":{"name": 123}}`), query)
	q := squirrel.Select("*").From("t")
	err := ApplyQuery(&q, defaultScheme(), query)
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

// ============================
// BoolResolver tests
// ============================

func TestBoolResolver_TrueDirect(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"flag": true}}`)
	if !strings.Contains(sql, "flag = ?") {
		t.Errorf("expected 'flag = ?', got: %s", sql)
	}
	if len(args) < 1 || args[0] != true {
		t.Errorf("expected arg true, got: %v", args)
	}
}

func TestBoolResolver_FalseDirect(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"flag": false}}`)
	if !strings.Contains(sql, "flag = ?") {
		t.Errorf("expected 'flag = ?', got: %s", sql)
	}
	if len(args) < 1 || args[0] != false {
		t.Errorf("expected arg false, got: %v", args)
	}
}

func TestBoolResolver_EqualOperator(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"flag": {"=": true}}}`)
	if !strings.Contains(sql, "flag = ?") {
		t.Errorf("expected 'flag = ?', got: %s", sql)
	}
}

func TestBoolResolver_NotEqual(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"flag": {"!=": false}}}`)
	if !strings.Contains(sql, "flag <> ?") {
		t.Errorf("expected 'flag <> ?', got: %s", sql)
	}
}

// ============================
// TimestampResolver tests
// ============================

func TestTimestampResolver_EqualDirect(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"created_at": 1700000000}}`)
	if !strings.Contains(sql, "created_at = ?") {
		t.Errorf("expected 'created_at = ?', got: %s", sql)
	}
	if len(args) < 1 {
		t.Fatalf("expected args")
	}
}

func TestTimestampResolver_Range(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"created_at": {"gte": "2024-01-01", "lte": "2024-12-31"}}}`)
	if !strings.Contains(sql, "created_at >= ?") || !strings.Contains(sql, "created_at <= ?") {
		t.Errorf("expected range, got: %s", sql)
	}
	if len(args) < 2 {
		t.Errorf("expected 2+ args, got: %d", len(args))
	}
}

func TestTimestampResolver_Gt(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"created_at": {"gt": "2024-01-01"}}}`)
	if !strings.Contains(sql, "created_at > ?") {
		t.Errorf("expected 'created_at > ?', got: %s", sql)
	}
}

func TestTimestampResolver_NotEqual(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"created_at": {"!=": "2024-06-01"}}}`)
	if !strings.Contains(sql, "created_at <> ?") {
		t.Errorf("expected 'created_at <> ?', got: %s", sql)
	}
}

// ============================
// ApplyQuery logic tests
// ============================

func TestApplyQuery_DefaultLimit(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{}}`)
	if !strings.Contains(sql, "LIMIT 100") {
		t.Errorf("expected default LIMIT 100, got: %s", sql)
	}
}

func TestApplyQuery_SchemeDefaultLimit(t *testing.T) {
	scheme := defaultScheme()
	scheme.DefaultLimit = 50
	sql, _ := buildSQL(t, scheme, `{"filters":{}}`)
	if !strings.Contains(sql, "LIMIT 50") {
		t.Errorf("expected scheme LIMIT 50, got: %s", sql)
	}
}

func TestApplyQuery_CustomLimit(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{}, "limit": 25}`)
	if !strings.Contains(sql, "LIMIT 25") {
		t.Errorf("expected LIMIT 25, got: %s", sql)
	}
}

func TestApplyQuery_CustomLimitOverridesScheme(t *testing.T) {
	scheme := defaultScheme()
	scheme.DefaultLimit = 50
	sql, _ := buildSQL(t, scheme, `{"filters":{}, "limit": 10}`)
	if !strings.Contains(sql, "LIMIT 10") {
		t.Errorf("expected LIMIT 10 to override scheme default 50, got: %s", sql)
	}
}

func TestApplyQuery_Offset(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{}, "offset": 500}`)
	if !strings.Contains(sql, "OFFSET 500") {
		t.Errorf("expected OFFSET 500, got: %s", sql)
	}
}

func TestApplyQuery_SchemeDefaultOffset(t *testing.T) {
	scheme := defaultScheme()
	scheme.DefaultOffset = 10
	sql, _ := buildSQL(t, scheme, `{"filters":{}}`)
	if !strings.Contains(sql, "OFFSET 10") {
		t.Errorf("expected scheme OFFSET 10, got: %s", sql)
	}
}

func TestApplyQuery_SortingSlice(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{}, "sorting":["name ASC"]}`)
	if !strings.Contains(sql, "ORDER BY name ASC") {
		t.Errorf("expected ORDER BY name ASC, got: %s", sql)
	}
	// Should NOT have default sort
	if strings.Contains(sql, "user_id DESC") {
		t.Errorf("default sort should not appear when sorting specified, got: %s", sql)
	}
}

func TestApplyQuery_SortingDefaultDirection(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{}, "sorting":["name"]}`)
	if !strings.Contains(sql, "ORDER BY name DESC") {
		t.Errorf("expected default DESC direction, got: %s", sql)
	}
}

func TestApplyQuery_SortingMap(t *testing.T) {
	// Map sorting should also prevent default sort
	query := &Query{
		Sorting: map[string]string{"name": "ASC"},
	}
	q := squirrel.Select("*").From("users")
	if err := ApplyQuery(&q, defaultScheme(), query); err != nil {
		t.Fatalf("ApplyQuery: %v", err)
	}
	sql, _, _ := q.ToSql()
	if !strings.Contains(sql, "ORDER BY name ASC") {
		t.Errorf("expected ORDER BY name ASC, got: %s", sql)
	}
	if strings.Contains(sql, "user_id DESC") {
		t.Errorf("default sort should not appear with map sorting, got: %s", sql)
	}
}

func TestApplyQuery_DefaultSort(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{}}`)
	if !strings.Contains(sql, "ORDER BY user_id DESC") {
		t.Errorf("expected default sort, got: %s", sql)
	}
}

func TestApplyQuery_UnknownFilterField(t *testing.T) {
	query := &Query{}
	_ = json.Unmarshal([]byte(`{"filters":{"unknown_field": 1}}`), query)
	q := squirrel.Select("*").From("t")
	err := ApplyQuery(&q, defaultScheme(), query)
	if err == nil {
		t.Error("expected error for unknown field")
	}
	if !strings.Contains(err.Error(), "unknown_field") {
		t.Errorf("error should mention the field name, got: %v", err)
	}
}

func TestApplyQuery_UnknownSortField(t *testing.T) {
	query := &Query{}
	_ = json.Unmarshal([]byte(`{"filters":{}, "sorting":["nonexistent ASC"]}`), query)
	q := squirrel.Select("*").From("t")
	err := ApplyQuery(&q, defaultScheme(), query)
	if err == nil {
		t.Error("expected error for unknown sort field")
	}
}

func TestApplyQuery_NoFilters(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{}`)
	if strings.Contains(sql, "WHERE") {
		t.Errorf("expected no WHERE clause, got: %s", sql)
	}
	if len(args) != 0 {
		t.Errorf("expected no args, got: %v", args)
	}
}

func TestApplyQuery_MultipleFilters(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{
		"filters": {
			"user_id": 123,
			"name": {"in": ["alice", "bob"]},
			"flag": true,
			"balance": {"gte": 4.32}
		},
		"sorting": ["user_id ASC"],
		"limit": 200,
		"offset": 1000
	}`)
	if !strings.Contains(sql, "LIMIT 200") {
		t.Errorf("expected LIMIT 200, got: %s", sql)
	}
	if !strings.Contains(sql, "OFFSET 1000") {
		t.Errorf("expected OFFSET 1000, got: %s", sql)
	}
	if !strings.Contains(sql, "ORDER BY user_id ASC") {
		t.Errorf("expected ORDER BY user_id ASC, got: %s", sql)
	}
	if len(args) < 4 {
		t.Errorf("expected 4+ args, got: %d", len(args))
	}
	t.Logf("SQL: %s\nArgs: %v", sql, args)
}

// ============================
// BindQuery tests
// ============================

func TestBindQuery_SimpleFilter(t *testing.T) {
	params := url.Values{}
	params.Set("filters[name]", "john")
	params.Set("limit", "50")
	params.Set("offset", "10")

	q := &Query{}
	if err := q.BindQuery(params); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}

	if q.Limit != 50 {
		t.Errorf("expected limit 50, got: %d", q.Limit)
	}
	if q.Offset != 10 {
		t.Errorf("expected offset 10, got: %d", q.Offset)
	}
	if q.Filters == nil {
		t.Fatal("expected filters")
	}
	filterMap, ok := q.Filters["name"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected filter map, got: %T", q.Filters["name"])
	}
	if filterMap["="] != "john" {
		t.Errorf("expected name = john, got: %v", filterMap)
	}
}

func TestBindQuery_OperatorFilter(t *testing.T) {
	params := url.Values{}
	params.Set("filters[balance][gte]", "100")

	q := &Query{}
	if err := q.BindQuery(params); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}

	filterMap, ok := q.Filters["balance"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected filter map, got: %T", q.Filters["balance"])
	}
	if filterMap["gte"] != "100" {
		t.Errorf("expected gte=100, got: %v", filterMap)
	}
}

func TestBindQuery_InFilter(t *testing.T) {
	params := url.Values{}
	params.Add("filters[name][in][]", "alice")
	params.Add("filters[name][in][]", "bob")

	q := &Query{}
	if err := q.BindQuery(params); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}

	filterMap, ok := q.Filters["name"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected filter map")
	}
	inVals, ok := filterMap["in"].([]string)
	if !ok {
		t.Fatalf("expected in as []string, got: %T", filterMap["in"])
	}
	if len(inVals) != 2 || inVals[0] != "alice" || inVals[1] != "bob" {
		t.Errorf("expected [alice, bob], got: %v", inVals)
	}
}

func TestBindQuery_SortingArray(t *testing.T) {
	params := url.Values{}
	params.Add("sort[]", "name ASC")
	params.Add("sort[]", "balance DESC")

	q := &Query{}
	if err := q.BindQuery(params); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}

	sorting, ok := q.Sorting.([]string)
	if !ok {
		t.Fatalf("expected []string sorting, got: %T", q.Sorting)
	}
	if len(sorting) != 2 {
		t.Errorf("expected 2 sort fields, got: %d", len(sorting))
	}
}

func TestBindQuery_SortingMap(t *testing.T) {
	params := url.Values{}
	params.Set("sort[name]", "ASC")

	q := &Query{}
	if err := q.BindQuery(params); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}

	sorting, ok := q.Sorting.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string sorting, got: %T", q.Sorting)
	}
	if sorting["name"] != "ASC" {
		t.Errorf("expected name=ASC, got: %v", sorting)
	}
}

func TestBindQuery_UnsupportedOperator(t *testing.T) {
	params := url.Values{}
	params.Set("filters[name][xss]", "evil")

	q := &Query{}
	err := q.BindQuery(params)
	if err == nil {
		t.Error("expected error for unsupported operator")
	}
}

func TestBindQuery_EmptyValues(t *testing.T) {
	params := url.Values{}

	q := &Query{}
	if err := q.BindQuery(params); err != nil {
		t.Fatalf("BindQuery: %v", err)
	}
	if q.Filters != nil {
		t.Errorf("expected nil filters, got: %v", q.Filters)
	}
}

// ============================
// Original test (preserved)
// ============================

func TestApplyQuery_Original(t *testing.T) {
	jsonQuery := `{
    "filters": {
        "user_id": 123,
		"name": {"in": ["sdf", "fdsg"]},
		"flag": true,
		"balance": {"gte": 4.3242}
    },
    "sorting": ["user_id"],
    "offset": 1000
	}`

	query := &Query{}
	if err := json.Unmarshal([]byte(jsonQuery), query); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	scheme := &Scheme{
		Resolvers: map[string]FilterResolver{
			"user_id": Int,
			"name":    String,
			"flag":    Boolean,
			"balance": Float,
		},
		DefaultLimit: 321,
	}

	q := squirrel.Select("user_id").From("table")
	if err := ApplyQuery(&q, scheme, query); err != nil {
		t.Fatalf("ApplyQuery: %v", err)
	}

	sql, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("ToSql: %v", err)
	}

	t.Logf("query: %s \n args: %v", sql, args)

	// After fix: name IN should use proper SQL IN, not PG array literal
	if !strings.Contains(sql, "name IN (?,?)") {
		t.Errorf("expected 'name IN (?,?)', got: %s", sql)
	}
	// After fix: limit should use scheme default (321), not global (100)
	if !strings.Contains(sql, "LIMIT 321") {
		t.Errorf("expected LIMIT 321 (scheme default), got: %s", sql)
	}
}

// ============================
// Phase 2: NULL / NOT NULL tests
// ============================

func TestIntResolver_Null(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"user_id": {"null": true}}}`)
	if !strings.Contains(sql, "user_id IS NULL") {
		t.Errorf("expected 'user_id IS NULL', got: %s", sql)
	}
}

func TestIntResolver_NotNull(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"user_id": {"not null": true}}}`)
	if !strings.Contains(sql, "user_id IS NOT NULL") {
		t.Errorf("expected 'user_id IS NOT NULL', got: %s", sql)
	}
}

func TestStringResolver_Null(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"name": {"null": true}}}`)
	if !strings.Contains(sql, "name IS NULL") {
		t.Errorf("expected 'name IS NULL', got: %s", sql)
	}
}

func TestBoolResolver_Null(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"flag": {"null": true}}}`)
	if !strings.Contains(sql, "flag IS NULL") {
		t.Errorf("expected 'flag IS NULL', got: %s", sql)
	}
}

func TestFloatResolver_Null(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"balance": {"null": true}}}`)
	if !strings.Contains(sql, "balance IS NULL") {
		t.Errorf("expected 'balance IS NULL', got: %s", sql)
	}
}

func TestTimestampResolver_Null(t *testing.T) {
	sql, _ := buildSQL(t, defaultScheme(), `{"filters":{"created_at": {"null": true}}}`)
	if !strings.Contains(sql, "created_at IS NULL") {
		t.Errorf("expected 'created_at IS NULL', got: %s", sql)
	}
}

// ============================
// Phase 2: MaxLimit tests
// ============================

func TestApplyQuery_MaxLimitCaps(t *testing.T) {
	scheme := defaultScheme()
	scheme.MaxLimit = 50
	sql, _ := buildSQL(t, scheme, `{"filters":{}, "limit": 200}`)
	if !strings.Contains(sql, "LIMIT 50") {
		t.Errorf("expected MaxLimit to cap at 50, got: %s", sql)
	}
}

func TestApplyQuery_MaxLimitDoesNotAffectSmaller(t *testing.T) {
	scheme := defaultScheme()
	scheme.MaxLimit = 500
	sql, _ := buildSQL(t, scheme, `{"filters":{}, "limit": 25}`)
	if !strings.Contains(sql, "LIMIT 25") {
		t.Errorf("expected LIMIT 25, got: %s", sql)
	}
}

func TestApplyQuery_MaxLimitCapsDefault(t *testing.T) {
	scheme := defaultScheme()
	scheme.MaxLimit = 10
	sql, _ := buildSQL(t, scheme, `{"filters":{}}`)
	if !strings.Contains(sql, "LIMIT 10") {
		t.Errorf("expected MaxLimit 10 to cap default 100, got: %s", sql)
	}
}

// ============================
// Phase 2: FloatResolver not in (fixed)
// ============================

func TestFloatResolver_NotIn_Proper(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"balance": {"not in": [1.5, 2.5, 3.5]}}}`)
	if !strings.Contains(sql, "balance NOT IN (?,?,?)") {
		t.Errorf("expected 'balance NOT IN (?,?,?)', got: %s", sql)
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got: %d (%v)", len(args), args)
	}
	// Verify proper float64 values, not raw interface{}
	for i, expected := range []float64{1.5, 2.5, 3.5} {
		if v, ok := args[i].(float64); !ok || v != expected {
			t.Errorf("arg[%d]: expected %f, got: %v (%T)", i, expected, args[i], args[i])
		}
	}
}

// ============================
// Phase 2: StringResolver unknown operator error
// ============================

func TestStringResolver_UnknownOperator(t *testing.T) {
	query := &Query{}
	_ = json.Unmarshal([]byte(`{"filters":{"name": {"bogus": "x"}}}`), query)
	q := squirrel.Select("*").From("t")
	err := ApplyQuery(&q, defaultScheme(), query)
	if err == nil {
		t.Fatal("expected error for unknown string operator")
	}
	if !errors.Is(err, ErrUnknownOperator) {
		t.Errorf("expected ErrUnknownOperator in chain, got: %v", err)
	}
}

// ============================
// Phase 2: Error wrapping tests
// ============================

func TestApplyQuery_ErrorIsErrUnknownField(t *testing.T) {
	query := &Query{}
	_ = json.Unmarshal([]byte(`{"filters":{"no_such_field": 1}}`), query)
	q := squirrel.Select("*").From("t")
	err := ApplyQuery(&q, defaultScheme(), query)
	if !errors.Is(err, ErrUnknownField) {
		t.Errorf("expected errors.Is(err, ErrUnknownField), got: %v", err)
	}
}

func TestApplyQuery_SortErrorIsErrUnknownField(t *testing.T) {
	query := &Query{}
	_ = json.Unmarshal([]byte(`{"filters":{}, "sorting":["nonexistent ASC"]}`), query)
	q := squirrel.Select("*").From("t")
	err := ApplyQuery(&q, defaultScheme(), query)
	if !errors.Is(err, ErrUnknownField) {
		t.Errorf("expected errors.Is(err, ErrUnknownField), got: %v", err)
	}
}

// ============================
// Phase 2: Combined NULL + operator
// ============================

func TestIntResolver_NullWithRange(t *testing.T) {
	sql, args := buildSQL(t, defaultScheme(), `{"filters":{"user_id": {"gte": 10, "null": true}}}`)
	if !strings.Contains(sql, "user_id >= ?") {
		t.Errorf("expected 'user_id >= ?', got: %s", sql)
	}
	if !strings.Contains(sql, "user_id IS NULL") {
		t.Errorf("expected 'user_id IS NULL', got: %s", sql)
	}
	if len(args) < 1 {
		t.Errorf("expected args, got none")
	}
}
