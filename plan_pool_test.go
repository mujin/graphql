package graphql_test

import (
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/benchutil"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
)

// countingResultPool wraps SimpleResultPool and records how many
// allocations went through the pool.
type countingResultPool struct {
	graphql.SimpleResultPool
	gets    int
	objects int
	lists   int
}

func (pool *countingResultPool) Get() *graphql.Result {
	pool.gets++
	return pool.SimpleResultPool.Get()
}

func (pool *countingResultPool) GetObjectFor(result *graphql.Result, capacity int) map[string]interface{} {
	pool.objects++
	return pool.SimpleResultPool.GetObjectFor(result, capacity)
}

func (pool *countingResultPool) GetListFor(result *graphql.Result, capacity int) []interface{} {
	pool.lists++
	return pool.SimpleResultPool.GetListFor(result, capacity)
}

func TestExecutePlanWithPool(t *testing.T) {
	schema := benchutil.WideArgedSchemaWithXFieldsAndYItems(3, 2)

	query := `{ wide { a(value: "x") b(value: "y") } }`
	document, err := parser.Parse(parser.ParseParams{
		Source: source.NewSource(&source.Source{Body: []byte(query)}),
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	plan, err := graphql.PlanQuery(&schema, document, "")
	if err != nil {
		t.Fatalf("plan: %v", err)
	}

	pool := &countingResultPool{}
	result := graphql.ExecutePlanWithPool(plan, graphql.ExecuteParams{Schema: schema}, pool)
	if len(result.Errors) > 0 {
		t.Fatalf("execute: %v", result.Errors)
	}
	if result.Data == nil {
		t.Fatal("expected data")
	}
	if pool.gets == 0 {
		t.Fatal("expected the Result to come from the pool")
	}
	if pool.objects == 0 || pool.lists == 0 {
		t.Fatalf("expected pooled object and list allocations, got objects=%d lists=%d", pool.objects, pool.lists)
	}
}
