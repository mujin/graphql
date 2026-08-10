package graphql_test

import (
	"sync"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
)

// One schema serves every in-flight request, so validating two requests at once must not write to it. Enum name/value
// lookups and the schema's possible-type map used to be filled on first use, which the race detector caught here.
func TestConcurrentValidateDocument(t *testing.T) {
	sizeEnum := graphql.NewEnum(graphql.EnumConfig{
		Name: "Size",
		Values: graphql.EnumValueConfigMap{
			"SMALL": &graphql.EnumValueConfig{Value: 0},
			"LARGE": &graphql.EnumValueConfig{Value: 1},
		},
	})
	petInterface := graphql.NewInterface(graphql.InterfaceConfig{
		Name: "Pet",
		Fields: graphql.Fields{
			"name": &graphql.Field{Type: graphql.String},
		},
	})
	dogType := graphql.NewObject(graphql.ObjectConfig{
		Name:       "Dog",
		Interfaces: []*graphql.Interface{petInterface},
		Fields: graphql.Fields{
			"name":  &graphql.Field{Type: graphql.String},
			"woofs": &graphql.Field{Type: graphql.Boolean},
		},
	})
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"pet": &graphql.Field{
					Type: petInterface,
					Args: graphql.FieldConfigArgument{
						"size": &graphql.ArgumentConfig{Type: sizeEnum},
					},
				},
			},
		}),
		Types: []graphql.Type{dogType},
	})
	if err != nil {
		t.Fatalf("failed to build schema: %v", err)
	}

	// the enum literal reaches Enum.ParseLiteral, the fragment spread reaches Schema.IsPossibleType
	document, err := parser.Parse(parser.ParseParams{Source: source.NewSource(&source.Source{Body: []byte(`
		query {
			pet(size: LARGE) {
				name
				... on Dog { woofs }
			}
		}
	`)})})
	if err != nil {
		t.Fatalf("failed to parse query: %v", err)
	}

	var waitGroup sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for round := 0; round < 20; round++ {
				if result := graphql.ValidateDocument(&schema, document, nil); !result.IsValid {
					t.Errorf("query failed to validate: %v", result.Errors)
				}
			}
		}()
	}
	waitGroup.Wait()
}
