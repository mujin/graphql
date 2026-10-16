package graphql_test

import (
	"reflect"
	"testing"

	"github.com/graphql-go/graphql"
)

// planThunkSchema returns a schema whose fields resolve to thunks, at the top level and inside a
// list, for both a query and a mutation.
func planThunkSchema(t *testing.T) graphql.Schema {
	t.Helper()
	thunk := func(value interface{}) func() (interface{}, error) {
		return func() (interface{}, error) { return value, nil }
	}
	itemType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Item",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return thunk(p.Source.(map[string]interface{})["name"]), nil
				},
			},
		},
	})
	fields := func() graphql.Fields {
		return graphql.Fields{
			"value": &graphql.Field{
				Type:    graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) { return thunk("deferred"), nil },
			},
			"items": &graphql.Field{
				Type: graphql.NewList(itemType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return []interface{}{map[string]interface{}{"name": "first"}, map[string]interface{}{"name": "second"}}, nil
				},
			},
		}
	}
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    graphql.NewObject(graphql.ObjectConfig{Name: "Query", Fields: fields()}),
		Mutation: graphql.NewObject(graphql.ObjectConfig{Name: "Mutation", Fields: fields()}),
	})
	if err != nil {
		t.Fatalf("Error in schema %v", err.Error())
	}
	return schema
}

// The planned executor skips the dethunk walk when no field was deferred, so this checks a
// deferred field is still resolved on that path, for the breadth-first query walk and the
// depth-first mutation walk.
func TestPlanResolvesThunks(t *testing.T) {
	schema := planThunkSchema(t)
	expected := map[string]interface{}{
		"value": "deferred",
		"items": []interface{}{
			map[string]interface{}{"name": "first"},
			map[string]interface{}{"name": "second"},
		},
	}
	for _, query := range []string{`{ value items { name } }`, `mutation { value items { name } }`} {
		result := executePlanned(t, schema, query)
		if len(result.Errors) > 0 {
			t.Fatalf("%s: unexpected errors: %v", query, result.Errors)
		}
		if !reflect.DeepEqual(result.Data, expected) {
			t.Errorf("%s: expected %v, got %v", query, expected, result.Data)
		}
	}
}
