package querytool

import (
	"encoding/json"
	"testing"

	"github.com/Masterminds/squirrel"
)

func TestApplyQuery(t *testing.T) {
	jsonQuery := `{
    "filters": {
        "user_id": 123,
		"name": {"in": ["sdf", "fdsg"]}
    },
    "sorting": ["user_id"],
    "offset": 1000
	}`

	query := &Query{}

	err := json.Unmarshal([]byte(jsonQuery), query)
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}

	scheme := &Scheme{
		Resolvers: map[string]FilterResolver{
			"user_id": Int,
			"name":    String,
		},
		DefaultLimit: 321,
	}

	q := squirrel.Select("user_id").From("table")

	err = ApplyQuery(&q, scheme, query)
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}

	sql, args, err := q.ToSql()
	if err != nil {
		t.Log(err.Error())
		t.FailNow()
	}

	t.Logf("query: %s \n args: %v", sql, args)
}
