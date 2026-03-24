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
	if testing.Short() {
		t.Skip("use go test -short to skip; full suite needs Docker for Testcontainers")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

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

	// Postgres can briefly refuse connections right after the container reports ready.
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
	defer pool.Close()

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			Store: store.New(pool),
			Queue: queue.NoOpPublisher{},
		},
	}))

	mux := http.NewServeMux()
	mux.Handle("/query", srv)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	q := `mutation { createOrder(input: {}) { id status } }`
	body := map[string]string{"query": q}
	raw, _ := json.Marshal(body)
	res, err := http.Post(ts.URL+"/query", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", res.StatusCode, b)
	}

	var createOut struct {
		Data struct {
			CreateOrder struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"createOrder"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	if err := json.Unmarshal(b, &createOut); err != nil {
		t.Fatal(err)
	}
	if len(createOut.Errors) > 0 {
		t.Fatalf("graphql errors: %+v body=%s", createOut.Errors, b)
	}
	id := createOut.Data.CreateOrder.ID
	if id == "" || createOut.Data.CreateOrder.Status != "PENDING" {
		t.Fatalf("unexpected createOrder: %+v", createOut.Data.CreateOrder)
	}

	q2 := `query Q($id: ID!) { order(id: $id) { id status } }`
	raw2, _ := json.Marshal(map[string]any{
		"query":     q2,
		"variables": map[string]string{"id": id},
	})
	res2, err := http.Post(ts.URL+"/query", "application/json", bytes.NewReader(raw2))
	if err != nil {
		t.Fatal(err)
	}
	defer res2.Body.Close()
	b2, _ := io.ReadAll(res2.Body)
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("query status %d: %s", res2.StatusCode, b2)
	}
	var getOut struct {
		Data struct {
			Order *struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"order"`
		} `json:"data"`
		Errors []any `json:"errors"`
	}
	if err := json.Unmarshal(b2, &getOut); err != nil {
		t.Fatal(err)
	}
	if len(getOut.Errors) > 0 {
		t.Fatalf("graphql errors: %+v", getOut.Errors)
	}
	if getOut.Data.Order == nil || getOut.Data.Order.ID != id {
		t.Fatalf("unexpected order: %+v body=%s", getOut.Data, b2)
	}
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
