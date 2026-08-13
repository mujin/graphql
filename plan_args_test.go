package graphql_test

import (
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
)

// The planned executor must hand resolvers the same Args map the unplanned one
// would. getArgumentValues returns nil for a field declaring no arguments and a
// non-nil empty map for a field whose declared arguments all resolved to
// nothing, so these pin both shapes down. Allocating an empty map for every
// field regardless cost a map header per resolved field.
func TestPlanArgsMatchUnplannedShape(t *testing.T) {
	type seen struct {
		args   map[string]interface{}
		called bool
	}
	var noArgs, optionalArg, suppliedArg seen

	record := func(target *seen) graphql.FieldResolveFn {
		return func(p graphql.ResolveParams) (interface{}, error) {
			target.args = p.Args
			target.called = true
			return "value", nil
		}
	}

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Type",
			Fields: graphql.Fields{
				"noArgs": &graphql.Field{
					Type:    graphql.String,
					Resolve: record(&noArgs),
				},
				"optionalArg": &graphql.Field{
					Type:    graphql.String,
					Args:    graphql.FieldConfigArgument{"only": &graphql.ArgumentConfig{Type: graphql.String}},
					Resolve: record(&optionalArg),
				},
				"suppliedArg": &graphql.Field{
					Type:    graphql.String,
					Args:    graphql.FieldConfigArgument{"only": &graphql.ArgumentConfig{Type: graphql.String}},
					Resolve: record(&suppliedArg),
				},
			},
		}),
	})
	if err != nil {
		t.Fatalf("schema: %v", err)
	}

	query := `{ noArgs optionalArg suppliedArg(only: "here") }`
	src := source.NewSource(&source.Source{Body: []byte(query), Name: "plan_args_test"})
	doc, err := parser.Parse(parser.ParseParams{Source: src})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	plan, err := graphql.PlanQuery(&schema, doc, "")
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	result := graphql.ExecutePlan(plan, graphql.ExecuteParams{Schema: schema, AST: doc})
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}

	if !noArgs.called || !optionalArg.called || !suppliedArg.called {
		t.Fatalf("expected every resolver to run, got %v %v %v", noArgs.called, optionalArg.called, suppliedArg.called)
	}

	// A field declaring no arguments gets nil, the same as getArgumentValues
	// returns on the unplanned path.
	if noArgs.args != nil {
		t.Errorf("field declaring no arguments: expected nil Args, got %#v", noArgs.args)
	}

	// A field declaring an argument that was not supplied gets a non-nil empty
	// map, so a resolver that writes to Args does not panic.
	if optionalArg.args == nil {
		t.Errorf("field with an unsupplied argument: expected a non-nil empty Args map, got nil")
	} else if len(optionalArg.args) != 0 {
		t.Errorf("field with an unsupplied argument: expected empty Args, got %#v", optionalArg.args)
	}

	if got := suppliedArg.args["only"]; got != "here" {
		t.Errorf("field with a supplied argument: expected %q, got %#v", "here", got)
	}
}

// Reading a nil Args map is well defined, so a resolver for a field declaring no
// arguments behaves the same whether it gets nil or an empty map.
func TestPlanNilArgsReadsAreSafe(t *testing.T) {
	var lookupOK bool
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Type",
			Fields: graphql.Fields{
				"noArgs": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						_, present := p.Args["missing"]
						lookupOK = !present && len(p.Args) == 0
						return "value", nil
					},
				},
			},
		}),
	})
	if err != nil {
		t.Fatalf("schema: %v", err)
	}

	src := source.NewSource(&source.Source{Body: []byte(`{ noArgs }`), Name: "plan_args_test"})
	doc, err := parser.Parse(parser.ParseParams{Source: src})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	plan, err := graphql.PlanQuery(&schema, doc, "")
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if result := graphql.ExecutePlan(plan, graphql.ExecuteParams{Schema: schema, AST: doc}); len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if !lookupOK {
		t.Errorf("expected lookup and len on a nil Args map to behave as an empty map")
	}
}
