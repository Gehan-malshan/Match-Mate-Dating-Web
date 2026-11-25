package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import "github.com/gehan-malshan/matchmate/graphql-gateway/internal/upstream"

type Resolver struct {
	Upstream *upstream.Client
}
