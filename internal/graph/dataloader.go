package graph

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/testsabirweb/plateful/internal/graph/model"
	"github.com/testsabirweb/plateful/internal/store"
	storedb "github.com/testsabirweb/plateful/internal/store/db"
	"github.com/vikstrous/dataloadgen"
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

			// Parse UUIDs; record per-index parse errors.
			pgids := make([]pgtype.UUID, 0, len(ids))
			validIdx := make([]int, 0, len(ids))
			for i, id := range ids {
				pgid, err := parseUUID(id)
				if err != nil {
					errs[i] = err
					continue
				}
				pgids = append(pgids, pgid)
				validIdx = append(validIdx, i)
			}

			// Single batch round-trip to the DB.
			orders, err := s.GetOrdersByIDs(ctx, pgids)
			if err != nil {
				for _, i := range validIdx {
					errs[i] = err
				}
				return out, errs
			}

			// Index returned rows by UUID bytes for O(1) lookup.
			byID := make(map[[16]byte]storedb.Order, len(orders))
			for _, o := range orders {
				byID[o.ID.Bytes] = o
			}

			// Align results to original ids slice; missing IDs stay nil (GraphQL convention).
			for j, i := range validIdx {
				o, ok := byID[pgids[j].Bytes]
				if !ok {
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
