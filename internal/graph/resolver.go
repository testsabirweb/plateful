package graph

import "github.com/testsabirweb/plateful/internal/store"

// Resolver is the root gqlgen resolver; inject dependencies here.
type Resolver struct {
	Store *store.Store
}
