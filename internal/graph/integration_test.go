package graph_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/testsabirweb/plateful/internal/graph"
	"github.com/testsabirweb/plateful/internal/queue"
	"github.com/testsabirweb/plateful/internal/store"
)

func TestIntegration_CreateOrderAndQuery(t *testing.T) {
	baseURL := setupIntegrationGraphQL(t)

	q := `mutation { createOrder(input: {}) { id status } }`
	b := gqlPost(t, baseURL, q, nil)
	if len(graphQLErrors(t, b)) > 0 {
		t.Fatalf("graphql errors: %s", b)
	}

	var createOut struct {
		Data struct {
			CreateOrder struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"createOrder"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &createOut); err != nil {
		t.Fatal(err)
	}
	id := createOut.Data.CreateOrder.ID
	if id == "" || createOut.Data.CreateOrder.Status != "PENDING" {
		t.Fatalf("unexpected createOrder: %+v", createOut.Data.CreateOrder)
	}

	q2 := `query Q($id: ID!) { order(id: $id) { id status } }`
	b2 := gqlPost(t, baseURL, q2, map[string]any{"id": id})
	if len(graphQLErrors(t, b2)) > 0 {
		t.Fatalf("graphql errors: %s", b2)
	}
	var getOut struct {
		Data struct {
			Order *struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"order"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b2, &getOut); err != nil {
		t.Fatal(err)
	}
	if getOut.Data.Order == nil || getOut.Data.Order.ID != id {
		t.Fatalf("unexpected order: %+v body=%s", getOut.Data, b2)
	}
}

func TestIntegration_UpdateOrderStatus(t *testing.T) {
	baseURL := setupIntegrationGraphQL(t)

	createBody := gqlPost(t, baseURL, `mutation { createOrder(input: { customerName: "t" }) { id status } }`, nil)
	if len(graphQLErrors(t, createBody)) > 0 {
		t.Fatalf("create: %s", createBody)
	}
	var createOut struct {
		Data struct {
			CreateOrder struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"createOrder"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createBody, &createOut); err != nil {
		t.Fatal(err)
	}
	id := createOut.Data.CreateOrder.ID

	up := `mutation U($id: ID!) { updateOrderStatus(id: $id, status: CONFIRMED) { id status } }`
	b := gqlPost(t, baseURL, up, map[string]any{"id": id})
	if len(graphQLErrors(t, b)) > 0 {
		t.Fatalf("update: %s", b)
	}
	var upOut struct {
		Data struct {
			UpdateOrder struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"updateOrderStatus"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &upOut); err != nil {
		t.Fatal(err)
	}
	if upOut.Data.UpdateOrder.ID != id || upOut.Data.UpdateOrder.Status != "CONFIRMED" {
		t.Fatalf("unexpected update: %+v", upOut.Data.UpdateOrder)
	}
}

func TestIntegration_UpdateOrderStatus_InvalidTransition(t *testing.T) {
	baseURL := setupIntegrationGraphQL(t)

	createBody := gqlPost(t, baseURL, `mutation { createOrder(input: {}) { id } }`, nil)
	if len(graphQLErrors(t, createBody)) > 0 {
		t.Fatalf("create: %s", createBody)
	}
	var createOut struct {
		Data struct {
			CreateOrder struct {
				ID string `json:"id"`
			} `json:"createOrder"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createBody, &createOut); err != nil {
		t.Fatal(err)
	}
	id := createOut.Data.CreateOrder.ID

	// pending → delivered skips steps
	bad := `mutation { updateOrderStatus(id: "` + id + `", status: DELIVERED) { id status } }`
	b := gqlPost(t, baseURL, bad, nil)
	if len(graphQLErrors(t, b)) == 0 {
		t.Fatalf("expected GraphQL errors, got: %s", b)
	}
}

func TestIntegration_OrdersFilterByStatus(t *testing.T) {
	baseURL := setupIntegrationGraphQL(t)

	a := gqlPost(t, baseURL, `mutation { createOrder(input: {}) { id } }`, nil)
	if len(graphQLErrors(t, a)) > 0 {
		t.Fatal(a)
	}
	var outA struct {
		Data struct {
			CreateOrder struct {
				ID string `json:"id"`
			} `json:"createOrder"`
		} `json:"data"`
	}
	_ = json.Unmarshal(a, &outA)
	idA := outA.Data.CreateOrder.ID

	b := gqlPost(t, baseURL, `mutation { createOrder(input: {}) { id } }`, nil)
	if len(graphQLErrors(t, b)) > 0 {
		t.Fatal(b)
	}
	var outB struct {
		Data struct {
			CreateOrder struct {
				ID string `json:"id"`
			} `json:"createOrder"`
		} `json:"data"`
	}
	_ = json.Unmarshal(b, &outB)
	idB := outB.Data.CreateOrder.ID

	up := `mutation { updateOrderStatus(id: "` + idA + `", status: CONFIRMED) { id } }`
	if len(graphQLErrors(t, gqlPost(t, baseURL, up, nil))) > 0 {
		t.Fatal("update A")
	}

	list := `query { orders(filter: { status: PENDING }) { id status } }`
	body := gqlPost(t, baseURL, list, nil)
	if len(graphQLErrors(t, body)) > 0 {
		t.Fatalf("orders: %s", body)
	}
	var listOut struct {
		Data struct {
			Orders []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"orders"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &listOut); err != nil {
		t.Fatal(err)
	}
	if len(listOut.Data.Orders) != 1 {
		t.Fatalf("want 1 pending order, got %+v", listOut.Data.Orders)
	}
	if listOut.Data.Orders[0].ID != idB || listOut.Data.Orders[0].Status != "PENDING" {
		t.Fatalf("unexpected filtered order: %+v", listOut.Data.Orders[0])
	}
}

func TestIntegration_OrdersFilterByCreatedRange(t *testing.T) {
	baseURL := setupIntegrationGraphQL(t)

	start := time.Now().UTC().Add(-1 * time.Hour)
	end := time.Now().UTC().Add(1 * time.Hour)

	createBody := gqlPost(t, baseURL, `mutation { createOrder(input: {}) { id createdAt } }`, nil)
	if len(graphQLErrors(t, createBody)) > 0 {
		t.Fatalf("create: %s", createBody)
	}
	var createOut struct {
		Data struct {
			CreateOrder struct {
				ID        string `json:"id"`
				CreatedAt string `json:"createdAt"`
			} `json:"createOrder"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createBody, &createOut); err != nil {
		t.Fatal(err)
	}
	id := createOut.Data.CreateOrder.ID

	q := `query Q($from: Time!, $to: Time!) {
  orders(filter: { createdFrom: $from, createdTo: $to }) { id }
}`
	body := gqlPost(t, baseURL, q, map[string]any{
		"from": start.Format(time.RFC3339Nano),
		"to":   end.Format(time.RFC3339Nano),
	})
	if len(graphQLErrors(t, body)) > 0 {
		t.Fatalf("orders in range: %s", body)
	}
	var inRange struct {
		Data struct {
			Orders []struct {
				ID string `json:"id"`
			} `json:"orders"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &inRange); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, o := range inRange.Data.Orders {
		if o.ID == id {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("order %s not in [%s, %s], got %v", id, start, end, inRange.Data.Orders)
	}

	past := time.Now().UTC().Add(-48 * time.Hour)
	q2 := `query Q($from: Time!, $to: Time!) {
  orders(filter: { createdFrom: $from, createdTo: $to }) { id }
}`
	body2 := gqlPost(t, baseURL, q2, map[string]any{
		"from": past.Format(time.RFC3339Nano),
		"to":   past.Add(1 * time.Hour).Format(time.RFC3339Nano),
	})
	if len(graphQLErrors(t, body2)) > 0 {
		t.Fatalf("orders empty range: %s", body2)
	}
	var emptyRange struct {
		Data struct {
			Orders []struct {
				ID string `json:"id"`
			} `json:"orders"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body2, &emptyRange); err != nil {
		t.Fatal(err)
	}
	for _, o := range emptyRange.Data.Orders {
		if o.ID == id {
			t.Fatalf("order %s should not appear in old window", id)
		}
	}
}

// setupIntegrationGraphQL starts Postgres (Testcontainers), runs migrations, and serves GraphQL at base URL (no trailing slash).
func setupIntegrationGraphQL(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("use go test -short to skip; full suite needs Docker for Testcontainers")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)

	pgC, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("plateful"),
		postgres.WithUsername("plateful"),
		postgres.WithPassword("plateful"),
	)
	if err != nil {
		t.Fatalf("postgres container: %v", err)
	}
	t.Cleanup(func() {
		_ = pgC.Terminate(context.Background())
	})

	connStr, err := pgC.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	waitForPostgres(t, ctx, connStr)

	root := projectRoot(t)
	migPath := filepath.Join(root, "db", "migrations")
	absMig, err := filepath.Abs(migPath)
	if err != nil {
		t.Fatal(err)
	}
	fileURL := "file://" + filepath.ToSlash(absMig)

	var m *migrate.Migrate
	for range 60 {
		m, err = migrate.New(fileURL, connStr)
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("migrate.New: %v", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatal(err)
	}

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			Store: store.New(pool),
			Queue: queue.NoOpPublisher{},
		},
	}))
	mux := http.NewServeMux()
	mux.Handle("/query", srv)
	ts := httptest.NewServer(mux)
	t.Cleanup(func() {
		ts.Close()
		pool.Close()
	})
	return ts.URL
}

func gqlPost(t *testing.T, baseURL, query string, variables map[string]any) []byte {
	t.Helper()
	payload := map[string]any{"query": query}
	if variables != nil {
		payload["variables"] = variables
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(baseURL+"/query", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("HTTP %d: %s", res.StatusCode, b)
	}
	return b
}

func graphQLErrors(t *testing.T, body []byte) []any {
	t.Helper()
	var envelope struct {
		Errors []any `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("json: %v body=%s", err, body)
	}
	return envelope.Errors
}

func waitForPostgres(t *testing.T, ctx context.Context, connStr string) {
	t.Helper()
	var last error
	for range 60 {
		c, err := pgx.Connect(ctx, connStr)
		if err == nil {
			_ = c.Close(ctx)
			return
		}
		last = err
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("postgres not reachable: %v", last)
}

func projectRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("go.mod not found")
		}
		wd = parent
	}
}
