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

func TestDoPlannedWithPool(t *testing.T) {
	schema := benchutil.WideArgedSchemaWithXFieldsAndYItems(3, 2)
	cache := graphql.NewPlanCache(graphql.PlanCacheOptions{})
	pool := &countingResultPool{}

	params := graphql.Params{
		Schema:        schema,
		RequestString: `{ wide { a(value: "x") } }`,
	}
	for callIndex := 0; callIndex < 2; callIndex++ {
		result := graphql.DoPlannedWithPool(params, &schema, cache, pool)
		if len(result.Errors) > 0 {
			t.Fatalf("call %d: %v", callIndex, result.Errors)
		}
		if result.Data == nil {
			t.Fatalf("call %d: expected data", callIndex)
		}
		if result.Request == nil {
			t.Fatalf("call %d: expected Request to be set for access logging", callIndex)
		}
	}
	hits, misses := cache.HitsMisses()
	if hits != 1 || misses != 1 {
		t.Fatalf("expected 1 hit and 1 miss, got hits=%d misses=%d", hits, misses)
	}
	if pool.gets == 0 || pool.objects == 0 {
		t.Fatalf("expected pooled allocations, got gets=%d objects=%d", pool.gets, pool.objects)
	}

	// Validation errors surface with the parsed request attached.
	badResult := graphql.DoPlannedWithPool(graphql.Params{
		Schema:        schema,
		RequestString: `{ nosuchfield }`,
	}, &schema, cache, pool)
	if len(badResult.Errors) == 0 {
		t.Fatal("expected validation errors")
	}
	if badResult.Request == nil {
		t.Fatal("expected Request on validation failure")
	}
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
