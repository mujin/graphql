package graphql_test

import (
	"reflect"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/testutil"
)

// Covers the sub-field collection memoized on the execution context by completeObjectValue.
// The existing suite compares whole graphql.Result values, which no longer match on this fork,
// so these compare Data alone to stay a usable regression signal.

type subFieldsPost struct {
	Title string
}

type subFieldsNote struct {
	Body string
}

// Builds a schema whose "items" field is a list of an interface with two implementations, so one
// selection set resolves against a different runtime type for each element of the same list.
func newSubFieldsInterfaceSchema(t *testing.T) graphql.Schema {
	t.Helper()

	entryType := graphql.NewInterface(graphql.InterfaceConfig{
		Name: "Entry",
		Fields: graphql.Fields{
			"kind": &graphql.Field{Type: graphql.String},
		},
	})
	postType := graphql.NewObject(graphql.ObjectConfig{
		Name:       "Post",
		Interfaces: []*graphql.Interface{entryType},
		IsTypeOf: func(p graphql.IsTypeOfParams) bool {
			_, ok := p.Value.(*subFieldsPost)
			return ok
		},
		Fields: graphql.Fields{
			"kind": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return "post", nil
				},
			},
			"title": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(*subFieldsPost).Title, nil
				},
			},
		},
	})
	noteType := graphql.NewObject(graphql.ObjectConfig{
		Name:       "Note",
		Interfaces: []*graphql.Interface{entryType},
		IsTypeOf: func(p graphql.IsTypeOfParams) bool {
			_, ok := p.Value.(*subFieldsNote)
			return ok
		},
		Fields: graphql.Fields{
			"kind": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return "note", nil
				},
			},
			"body": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(*subFieldsNote).Body, nil
				},
			},
		},
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"items": &graphql.Field{
					Type: graphql.NewList(entryType),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return []interface{}{
							&subFieldsPost{Title: "first"},
							&subFieldsNote{Body: "second"},
							&subFieldsPost{Title: "third"},
						}, nil
					},
				},
			},
		}),
		Types: []graphql.Type{postType, noteType},
	})
	if err != nil {
		t.Fatalf("error in schema %v", err)
	}
	return schema
}

// The runtime type is part of the memoization key, so interleaved implementations in one list must
// each keep their own sub-selection instead of inheriting the first element's.
func TestSubFieldCollectionIsPerRuntimeTypeInAList(t *testing.T) {
	query := `{
      items {
        kind
        ... on Post { title }
        ... on Note { body }
      }
    }`

	expectedData := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"kind": "post", "title": "first"},
			map[string]interface{}{"kind": "note", "body": "second"},
			map[string]interface{}{"kind": "post", "title": "third"},
		},
	}

	result := graphql.Do(graphql.Params{
		Schema:        newSubFieldsInterfaceSchema(t),
		RequestString: query,
	})
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if !reflect.DeepEqual(expectedData, result.Data) {
		t.Fatalf("unexpected result, diff: %v", testutil.Diff(expectedData, result.Data))
	}
}

// A named fragment, an inline fragment and plain fields merge into one sub-selection that every
// element of the list must receive in full.
func TestSubFieldCollectionMergesFragmentsForEveryListElement(t *testing.T) {
	query := `
    query {
      items {
        ...KindFields
        ... on Post { title }
        ... on Note { body }
      }
    }
    fragment KindFields on Entry {
      kind
    }`

	expectedData := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"kind": "post", "title": "first"},
			map[string]interface{}{"kind": "note", "body": "second"},
			map[string]interface{}{"kind": "post", "title": "third"},
		},
	}

	result := graphql.Do(graphql.Params{
		Schema:        newSubFieldsInterfaceSchema(t),
		RequestString: query,
	})
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if !reflect.DeepEqual(expectedData, result.Data) {
		t.Fatalf("unexpected result, diff: %v", testutil.Diff(expectedData, result.Data))
	}
}

// @skip resolves against variables, so the memoization must not leak a collection from one
// execution into another that supplies different variable values.
func TestSubFieldCollectionHonoursVariablesPerExecution(t *testing.T) {
	schema := newSubFieldsInterfaceSchema(t)
	query := `
    query ($skipKind: Boolean!) {
      items {
        kind @skip(if: $skipKind)
        ... on Post { title }
        ... on Note { body }
      }
    }`

	runWithSkip := func(skipKind bool) interface{} {
		result := graphql.Do(graphql.Params{
			Schema:         schema,
			RequestString:  query,
			VariableValues: map[string]interface{}{"skipKind": skipKind},
		})
		if len(result.Errors) != 0 {
			t.Fatalf("unexpected errors: %v", result.Errors)
		}
		return result.Data
	}

	keptData := runWithSkip(false)
	expectedKept := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"kind": "post", "title": "first"},
			map[string]interface{}{"kind": "note", "body": "second"},
			map[string]interface{}{"kind": "post", "title": "third"},
		},
	}
	if !reflect.DeepEqual(expectedKept, keptData) {
		t.Fatalf("unexpected result with skip disabled, diff: %v", testutil.Diff(expectedKept, keptData))
	}

	skippedData := runWithSkip(true)
	expectedSkipped := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"title": "first"},
			map[string]interface{}{"body": "second"},
			map[string]interface{}{"title": "third"},
		},
	}
	if !reflect.DeepEqual(expectedSkipped, skippedData) {
		t.Fatalf("unexpected result with skip enabled, diff: %v", testutil.Diff(expectedSkipped, skippedData))
	}
}
