package graphql_test

import (
	"reflect"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/testutil"
)

// getArgumentValues returns a nil map for a field that declares no arguments, which resolvers must
// still be able to read from.
func TestArgumentsAreReadableWhenFieldDeclaresNone(t *testing.T) {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"noArguments": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						if len(p.Args) != 0 {
							return nil, nil
						}
						// reading an absent key must be safe on the nil map
						if value, ok := p.Args["missing"]; ok || value != nil {
							return nil, nil
						}
						return "resolved", nil
					},
				},
			},
		}),
	})
	if err != nil {
		t.Fatalf("error in schema %v", err)
	}

	expectedData := map[string]interface{}{"noArguments": "resolved"}
	result := graphql.Do(graphql.Params{Schema: schema, RequestString: `{ noArguments }`})
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if !reflect.DeepEqual(expectedData, result.Data) {
		t.Fatalf("unexpected result, diff: %v", testutil.Diff(expectedData, result.Data))
	}
}

// Declared arguments must still arrive, both when supplied by the query and when falling back to a
// default value.
func TestArgumentsStillResolveWhenFieldDeclaresThem(t *testing.T) {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"echo": &graphql.Field{
					Type: graphql.String,
					Args: graphql.FieldConfigArgument{
						"value": &graphql.ArgumentConfig{
							Type:         graphql.String,
							DefaultValue: "fallback",
						},
					},
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return p.Args["value"], nil
					},
				},
			},
		}),
	})
	if err != nil {
		t.Fatalf("error in schema %v", err)
	}

	for query, expectedValue := range map[string]string{
		`{ echo(value: "supplied") }`: "supplied",
		`{ echo }`:                    "fallback",
	} {
		result := graphql.Do(graphql.Params{Schema: schema, RequestString: query})
		if len(result.Errors) != 0 {
			t.Fatalf("unexpected errors for %s: %v", query, result.Errors)
		}
		expectedData := map[string]interface{}{"echo": expectedValue}
		if !reflect.DeepEqual(expectedData, result.Data) {
			t.Fatalf("unexpected result for %s, diff: %v", query, testutil.Diff(expectedData, result.Data))
		}
	}
}
