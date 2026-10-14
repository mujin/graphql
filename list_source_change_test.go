package graphql_test

import (
	"testing"

	"github.com/graphql-go/graphql"
)

type listItemForTest struct {
	name string
}

// Returns a schema whose list field resolves to a pointer to a slice, and whose item resolver appends
// to that slice, like another goroutine changing a shared source while the list is completed.
func newGrowingListSchemaForTest(t *testing.T) graphql.Schema {
	t.Helper()
	items := []*listItemForTest{{name: "item0"}, {name: "item1"}}
	itemType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Item",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type: graphql.String,
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					if len(items) < 10 {
						items = append(items, &listItemForTest{name: "added"})
					}
					return params.Source.(*listItemForTest).name, nil
				},
			},
		},
	})
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"items": &graphql.Field{
					Type: graphql.NewList(itemType),
					Resolve: func(params graphql.ResolveParams) (interface{}, error) {
						return &items, nil
					},
				},
			},
		}),
	})
	if err != nil {
		t.Fatalf("failed to build the schema: %s", err)
	}
	return schema
}

// A list is completed from the slice it was resolved to, so a source that grows meanwhile does not
// lengthen the list, possibly without end, or append past the capacity taken from the result pool.
func TestListIsCompletedFromTheResolvedSlice(t *testing.T) {
	for _, executorName := range []string{"planned", "unplanned"} {
		t.Run(executorName, func(t *testing.T) {
			schema := newGrowingListSchemaForTest(t)
			params := graphql.Params{Schema: schema, RequestString: `{ items { name } }`}
			var result *graphql.Result
			if executorName == "planned" {
				result = graphql.DoPlannedWithPool(params, &schema, graphql.NewPlanCache(graphql.PlanCacheOptions{}), &graphql.SimpleResultPool{})
			} else {
				result = graphql.Do(params)
			}
			if len(result.Errors) > 0 {
				t.Fatalf("unexpected errors: %v", result.Errors)
			}
			items := result.Data.(map[string]interface{})["items"].([]interface{})
			if len(items) != 2 {
				t.Fatalf("expected the 2 items the field resolved to, got %d", len(items))
			}
			for index, expectedName := range []string{"item0", "item1"} {
				if name := items[index].(map[string]interface{})["name"]; name != expectedName {
					t.Errorf("expected item %d named %q, got %v", index, expectedName, name)
				}
			}
		})
	}
}
