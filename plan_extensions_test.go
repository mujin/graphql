package graphql_test

import (
	"context"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
)

// planExtensionsSchema mirrors the schema the extension tests use, with a
// resolver that reports the ResolveInfo it was handed so a test can assert what
// reached it.
func planExtensionsSchema(t *testing.T, seen *graphql.ResolveInfo) graphql.Schema {
	t.Helper()
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Type",
			Fields: graphql.Fields{
				"a": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						*seen = p.Info
						return "foo", nil
					},
				},
			},
		}),
	})
	if err != nil {
		t.Fatalf("Error in schema %v", err.Error())
	}
	return schema
}

func executePlanned(t *testing.T, schema graphql.Schema, query string) *graphql.Result {
	t.Helper()
	src := source.NewSource(&source.Source{Body: []byte(query), Name: "plan_extensions_test"})
	doc, err := parser.Parse(parser.ParseParams{Source: src})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	plan, err := graphql.PlanQuery(&schema, doc, "")
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	return graphql.ExecutePlan(plan, graphql.ExecuteParams{Schema: schema, AST: doc})
}

// The planned executor hands ResolveInfo to the extensions by value through a
// helper, so that resolvePlannedField does not have to move the struct to the
// heap for every field. This checks the notifications still reach an extension
// on that path, since the other extension tests only cover graphql.Do.
func TestPlanExtensionResolveFieldNotifications(t *testing.T) {
	var seen graphql.ResolveInfo
	schema := planExtensionsSchema(t, &seen)

	var startedFields []string
	var finished int
	ext := newtestExt("testExt")
	ext.resolveFieldDidStartFn = func(ctx context.Context, i *graphql.ResolveInfo) (context.Context, graphql.ResolveFieldFinishFunc) {
		startedFields = append(startedFields, i.FieldName)
		return ctx, func(value interface{}, err error) {
			finished++
		}
	}
	schema.AddExtensions(ext)

	result := executePlanned(t, schema, `query Example { a }`)
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if len(startedFields) != 1 || startedFields[0] != "a" {
		t.Fatalf("expected ResolveFieldDidStart for field %q, got %v", "a", startedFields)
	}
	if finished != 1 {
		t.Fatalf("expected the finish func to run once, ran %d times", finished)
	}
}

// An extension receives a copy of ResolveInfo on the planned path rather than
// the resolver's own struct, so this pins down that a change it makes still
// reaches the resolver.
func TestPlanExtensionResolveInfoMutationReachesResolver(t *testing.T) {
	var seen graphql.ResolveInfo
	schema := planExtensionsSchema(t, &seen)

	ext := newtestExt("testExt")
	ext.resolveFieldDidStartFn = func(ctx context.Context, i *graphql.ResolveInfo) (context.Context, graphql.ResolveFieldFinishFunc) {
		i.RootValue = "set-by-extension"
		return ctx, func(value interface{}, err error) {}
	}
	schema.AddExtensions(ext)

	result := executePlanned(t, schema, `query Example { a }`)
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if seen.RootValue != "set-by-extension" {
		t.Fatalf("expected the resolver to see the extension's RootValue, got %v", seen.RootValue)
	}
}

// Without any extension registered the planned executor skips the notification
// helper entirely, which is the path that must not allocate a ResolveInfo.
func TestPlanNoExtensionsStillResolves(t *testing.T) {
	var seen graphql.ResolveInfo
	schema := planExtensionsSchema(t, &seen)

	result := executePlanned(t, schema, `query Example { a }`)
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if seen.FieldName != "a" {
		t.Fatalf("expected the resolver to receive ResolveInfo for field %q, got %q", "a", seen.FieldName)
	}
}
