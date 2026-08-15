package tlnmcp_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	tlnmcp "github.com/opentalon/tln-mcp"
	"github.com/opentalon/tln-language/pkg/tln"
)

func TestFactory_SatisfiesPluginFactory(t *testing.T) {
	var _ tln.PluginFactory = tlnmcp.Factory
}

// TestFactory_DispatchesWithBearer builds a resolver from a connector spec and
// checks the call reaches the configured endpoint with the bearer header set.
func TestFactory_DispatchesWithBearer(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotAuth = req.Header.Get("Authorization")
		b, _ := io.ReadAll(req.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
\t\t_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`)
	}))
	defer srv.Close()

	r, err := tlnmcp.Factory(tln.ConnectorSpec{
		Name:   "inventory",
		Plugin: "mcp",
		Config: map[string]string{"endpoint": srv.URL, "bearer": "secret123"},
	})
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	res, err := r.Call(context.Background(), "inventory", "list-items", map[string]any{"query": "x"})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if gotAuth != "Bearer secret123" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer secret123")
	}
	if gotBody == "" {
		t.Error("server received an empty request body")
	}
	if m, ok := res.(map[string]any); !ok || m["ok"] != true {
		t.Errorf("result = %#v, want {ok:true}", res)
	}
}

// TestFactory_NoBearerOmitsHeader confirms a connector without a bearer sends no
// Authorization header.
func TestFactory_NoBearerOmitsHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotAuth = req.Header.Get("Authorization")
\t\t_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":null}`)
	}))
	defer srv.Close()

	r, _ := tlnmcp.Factory(tln.ConnectorSpec{Name: "svc", Config: map[string]string{"endpoint": srv.URL}})
	if _, err := r.Call(context.Background(), "svc", "ping", nil); err != nil {
		t.Fatalf("call: %v", err)
	}
	if gotAuth != "" {
		t.Errorf("no bearer configured, but Authorization = %q", gotAuth)
	}
}

func TestFactory_RequiresEndpoint(t *testing.T) {
	if _, err := tlnmcp.Factory(tln.ConnectorSpec{Name: "svc"}); err == nil {
		t.Fatal("expected an error when endpoint is missing")
	}
}
