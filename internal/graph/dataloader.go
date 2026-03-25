package graph

import (
	"context"
	"errors"
	"net/http"

	"github.com/vikstrous/dataloadgen"
	"github.com/testsabirweb/plateful/internal/graph/model"
	"github.com/testsabirweb/plateful/internal/store"
)

type contextKey string

const loadersKey contextKey = "dataloaders"

// Loaders holds all dataloaders for a single request.
type Loaders struct {
	OrderByID *dataloadgen.Loader[string, *model.Order]
}

func newLoaders(s *store.Store) *Loaders {
	return &Loaders{
		OrderByID: dataloadgen.NewLoader(func(ctx context.Context, ids []string) ([]*model.Order, []error) {
			out := make([]*model.Order, len(ids))
			errs := make([]error, len(ids))
			for i, id := range ids {
				pgid, err := parseUUID(id)
				if err != nil {
					errs[i] = err
					continue
				}
				o, err := s.GetOrderByID(ctx, pgid)
				if err != nil {
					if errors.Is(err, store.ErrNotFound) {
						// nil, nil = not found (GraphQL convention)
						continue
					}
					errs[i] = err
					continue
				}
				gql, err := orderToGQL(o)
				if err != nil {
					errs[i] = err
					continue
				}
				out[i] = gql
			}
			return out, errs
		}),
	}
}

// DataloaderMiddleware injects per-request Loaders into the context.
func DataloaderMiddleware(s *store.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loadersKey, newLoaders(s))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func loadersFromCtx(ctx context.Context) *Loaders {
	l, _ := ctx.Value(loadersKey).(*Loaders)
	return l
}
