// Package stake provides an idiomatic Go HTTP client for Stake's unofficial API.
//
// Create a client with a session token or credentials, then use the typed NYSE
// and ASX service groups:
//
//	client, err := stake.NewClient(stake.WithSessionToken(token))
//	positions, err := client.NYSE.Equities.List(ctx)
//	orders, err := client.ASX.Orders.List(ctx)
//
// Methods accept context.Context, use net/http, and return typed values plus errors.
package stake
