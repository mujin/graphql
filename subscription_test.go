package graphql_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/testutil"
)

func TestSchemaSubscribe(t *testing.T) {

	testutil.RunSubscribes(t, []*testutil.TestSubscription{
		{
			Name: "subscribe without resolver",
			Schema: makeSubscriptionSchema(t, graphql.ObjectConfig{
				Name: "Subscription",
				Fields: graphql.Fields{
					"sub_without_resolver": &graphql.Field{
						Type: graphql.String,
						Subscribe: makeSubscribeToMapFunction([]map[string]interface{}{
							{
								"sub_without_resolver": "a",
							},
							{
								"sub_without_resolver": "b",
							},
							{
								"sub_without_resolver": "c",
							},
						}),
					},
				},
			}),
			Query: `
				subscription {
					sub_without_resolver
				}
			`,
			ExpectedResults: []testutil.TestResponse{
				{Data: `{ "sub_without_resolver": "a" }`},
				{Data: `{ "sub_without_resolver": "b" }`},
				{Data: `{ "sub_without_resolver": "c" }`},
			},
		},
		{
			Name: "subscribe with resolver",
			Schema: makeSubscriptionSchema(t, graphql.ObjectConfig{
				Name: "Subscription",
				Fields: graphql.Fields{
					"sub_with_resolver": &graphql.Field{
						Type: graphql.String,
						Resolve: func(p graphql.ResolveParams) (interface{}, error) {
							return p.Source, nil
						},
						Subscribe: makeSubscribeToStringFunction([]string{"a", "b", "c"}),
					},
				},
			}),
			Query: `
				subscription {
					sub_with_resolver
				}
			`,
			ExpectedResults: []testutil.TestResponse{
				{Data: `{ "sub_with_resolver": "a" }`},
				{Data: `{ "sub_with_resolver": "b" }`},
				{Data: `{ "sub_with_resolver": "c" }`},
			},
		},
		{
			Name: "receive query validation error",
			Schema: makeSubscriptionSchema(t, graphql.ObjectConfig{
				Name: "Subscription",
				Fields: graphql.Fields{
					"sub_without_resolver": &graphql.Field{
						Type:      graphql.String,
						Subscribe: makeSubscribeToStringFunction([]string{"a", "b", "c"}),
					},
				},
			}),
			Query: `
				subscription {
					sub_without_resolver
					xxx
				}
			`,
			ExpectedResults: []testutil.TestResponse{
				{Errors: []string{"Cannot query field \"xxx\" on type \"Subscription\"."}},
			},
		},
		{
			Name: "panic inside subscribe is recovered",
			Schema: makeSubscriptionSchema(t, graphql.ObjectConfig{
				Name: "Subscription",
				Fields: graphql.Fields{
					"should_error": &graphql.Field{
						Type: graphql.String,
						Subscribe: func(p graphql.ResolveParams) (interface{}, error) {
							panic(errors.New("got a panic error"))
						},
					},
				},
			}),
			Query: `
				subscription {
					should_error
				}
			`,
			ExpectedResults: []testutil.TestResponse{
				{Errors: []string{"got a panic error"}},
			},
		},
		{
			Name: "subscribe with resolver changes output",
			Schema: makeSubscriptionSchema(t, graphql.ObjectConfig{
				Name: "Subscription",
				Fields: graphql.Fields{
					"sub_with_resolver": &graphql.Field{
						Type:      graphql.String,
						Subscribe: makeSubscribeToStringFunction([]string{"a", "b", "c", "d"}),
						Resolve: func(p graphql.ResolveParams) (interface{}, error) {
							return fmt.Sprintf("result=%v", p.Source), nil
						},
					},
				},
			}),
			Query: `
				subscription {
					sub_with_resolver
				}
			`,
			ExpectedResults: []testutil.TestResponse{
				{Data: `{ "sub_with_resolver": "result=a" }`},
				{Data: `{ "sub_with_resolver": "result=b" }`},
				{Data: `{ "sub_with_resolver": "result=c" }`},
				{Data: `{ "sub_with_resolver": "result=d" }`},
			},
		},
		{
			Name: "subscribe to a nested object",
			Schema: makeSubscriptionSchema(t, graphql.ObjectConfig{
				Name: "Subscription",
				Fields: graphql.Fields{
					"sub_with_object": &graphql.Field{
						Type: graphql.NewObject(graphql.ObjectConfig{
							Name: "Obj",
							Fields: graphql.Fields{
								"field": &graphql.Field{
									Type: graphql.String,
								},
							},
						}),
						Resolve: func(p graphql.ResolveParams) (interface{}, error) {
							return p.Source, nil
						},
						Subscribe: makeSubscribeToMapFunction([]map[string]interface{}{
							{
								"field": "hello",
							},
							{
								"field": "bye",
							},
							{
								"field": nil,
							},
						}),
					},
				},
			}),
			Query: `
				subscription {
					sub_with_object {
						field
					}
				}
			`,
			ExpectedResults: []testutil.TestResponse{
				{Data: `{ "sub_with_object": { "field": "hello" } }`},
				{Data: `{ "sub_with_object": { "field": "bye" } }`},
				{Data: `{ "sub_with_object": { "field": null } }`},
			},
		},

		{
			Name: "subscription_resolver_can_error",
			Schema: makeSubscriptionSchema(t, graphql.ObjectConfig{
				Name: "Subscription",
				Fields: graphql.Fields{
					"should_error": &graphql.Field{
						Type: graphql.String,
						Subscribe: func(p graphql.ResolveParams) (interface{}, error) {
							return nil, errors.New("got a subscribe error")
						},
					},
				},
			}),
			Query: `
				subscription {
					should_error
				}
			`,
			ExpectedResults: []testutil.TestResponse{
				{
					Errors: []string{"got a subscribe error"},
				},
			},
		},
		{
			Name: "schema_without_subscribe_errors",
			Schema: makeSubscriptionSchema(t, graphql.ObjectConfig{
				Name: "Subscription",
				Fields: graphql.Fields{
					"should_error": &graphql.Field{
						Type: graphql.String,
					},
				},
			}),
			Query: `
				subscription {
					should_error
				}
			`,
			ExpectedResults: []testutil.TestResponse{
				{
					Errors: []string{"the subscription function \"should_error\" is not defined"},
				},
			},
		},
	})
}

func TestSubscribePlannedWithPool(t *testing.T) {
	schema := makeSubscriptionSchema(t, graphql.ObjectConfig{
		Name: "Subscription",
		Fields: graphql.Fields{
			"sub": &graphql.Field{
				Type:      graphql.String,
				Subscribe: makeSubscribeToStringFunction([]string{"a", "b", "c"}),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source, nil
				},
			},
		},
	})
	cache := graphql.NewPlanCache(graphql.PlanCacheOptions{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	channel := graphql.SubscribePlannedWithPool(graphql.Params{
		RequestString: `subscription { sub }`,
		Context:       ctx,
	}, &schema, cache, &graphql.SimpleResultPool{})

	var received []string
	for result := range channel {
		if len(result.Errors) > 0 {
			t.Fatalf("unexpected errors: %v", result.Errors)
		}
		if result.Request == nil {
			t.Fatal("expected Request to be set on subscription results")
		}
		data, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("unexpected data shape: %T", result.Data)
		}
		received = append(received, fmt.Sprintf("%v", data["sub"]))
		if len(received) == 3 {
			cancel()
			break
		}
	}
	if fmt.Sprintf("%v", received) != "[a b c]" {
		t.Fatalf("unexpected events: %v", received)
	}
	if _, misses := cache.HitsMisses(); misses != 1 {
		t.Fatalf("expected one plan-cache miss, got %d", misses)
	}
}

func TestSubscribePlannedWithMapper(t *testing.T) {
	schema := makeSubscriptionSchema(t, graphql.ObjectConfig{
		Name: "Subscription",
		Fields: graphql.Fields{
			"sub": &graphql.Field{
				Type:      graphql.String,
				Subscribe: makeSubscribeToStringFunction([]string{"a", "b", "c"}),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source, nil
				},
			},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sharedResult := &graphql.Result{Data: map[string]interface{}{"sub": "shared"}}
	var mappedPayloads []string
	mapper := func(payload interface{}, execute func(payload interface{}, resultPool graphql.ResultPool) *graphql.Result) *graphql.Result {
		mappedPayloads = append(mappedPayloads, fmt.Sprintf("%v", payload))
		if payload == "b" {
			// a mapper may hand out a result it did not execute, such as one shared with other subscriptions
			return sharedResult
		}
		return execute(payload, &graphql.SimpleResultPool{})
	}
	channel := graphql.SubscribePlannedWithMapper(graphql.Params{
		RequestString: `subscription { sub }`,
		Context:       ctx,
	}, &schema, graphql.NewPlanCache(graphql.PlanCacheOptions{}), &graphql.SimpleResultPool{}, mapper)

	var received []string
	for result := range channel {
		if len(result.Errors) > 0 {
			t.Fatalf("unexpected errors: %v", result.Errors)
		}
		if result == sharedResult {
			received = append(received, "shared")
		} else {
			if result.Request == nil {
				t.Fatal("expected Request to be set on executed results")
			}
			received = append(received, fmt.Sprintf("%v", result.Data.(map[string]interface{})["sub"]))
		}
		if len(received) == 3 {
			cancel()
			break
		}
	}
	if fmt.Sprintf("%v", received) != "[a shared c]" {
		t.Fatalf("unexpected events: %v", received)
	}
	if fmt.Sprintf("%v", mappedPayloads) != "[a b c]" {
		t.Fatalf("expected the mapper to see every payload, got %v", mappedPayloads)
	}
}

func TestSubscribePlannedWithMapperReportsPlanErrors(t *testing.T) {
	schema := makeSubscriptionSchema(t, graphql.ObjectConfig{
		Name: "Subscription",
		Fields: graphql.Fields{
			"sub": &graphql.Field{
				Type:      graphql.String,
				Subscribe: makeSubscribeToStringFunction([]string{"a"}),
			},
		},
	})
	channel := graphql.SubscribePlannedWithMapper(graphql.Params{
		RequestString: `subscription { missing }`,
		Context:       context.Background(),
	}, &schema, graphql.NewPlanCache(graphql.PlanCacheOptions{}), &graphql.SimpleResultPool{}, func(payload interface{}, execute func(payload interface{}, resultPool graphql.ResultPool) *graphql.Result) *graphql.Result {
		t.Error("expected the mapper not to run for an invalid subscription")
		return nil
	})
	result := <-channel
	if len(result.Errors) == 0 {
		t.Fatal("expected a validation error")
	}
}

func makeSubscribeToStringFunction(elements []string) func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
		c := make(chan interface{})
		go func() {
			for _, r := range elements {
				select {
				case <-p.Context.Done():
					close(c)
					return
				case c <- r:
				}
			}
			close(c)
		}()
		return c, nil
	}
}

func makeSubscribeToMapFunction(elements []map[string]interface{}) func(p graphql.ResolveParams) (interface{}, error) {
	return func(p graphql.ResolveParams) (interface{}, error) {
		c := make(chan interface{})
		go func() {
			for _, r := range elements {
				select {
				case <-p.Context.Done():
					close(c)
					return
				case c <- r:
				}
			}
			close(c)
		}()
		return c, nil
	}
}

func makeSubscriptionSchema(t *testing.T, c graphql.ObjectConfig) graphql.Schema {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:        dummyQuery,
		Subscription: graphql.NewObject(c),
	})
	if err != nil {
		t.Errorf("failed to create schema: %v", err)
	}
	return schema
}

var dummyQuery = graphql.NewObject(graphql.ObjectConfig{
	Name: "Query",
	Fields: graphql.Fields{

		"hello": &graphql.Field{Type: graphql.String},
	},
})
