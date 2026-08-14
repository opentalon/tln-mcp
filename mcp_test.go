package tlnmcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCall drives Call against a mock MCP server that echoes the tools/call
// request back as its result — exercising encode, transport, and decode.
func TestCall(t *testing.T) {
	var gotMethod, gotName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req rpcRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("bad request json: %v", err)
		}
		gotMethod, gotName = req.Method, req.Params.Name
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  map[string]any{"ok": true, "tool": req.Params.Name},
		})
	}))
	defer srv.Close()

	r := New(WithServer("inventory", srv.URL))
	out, err := r.Call(context.Background(), "inventory", "list_items", map[string]any{"category": "vehicle"})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if gotMethod != "tools/call" || gotName != "list_items" {
		t.Fatalf("server saw method=%q name=%q", gotMethod, gotName)
	}
	m, ok := out.(map[string]any)
	if !ok || m["ok"] != true || m["tool"] != "list_items" {
		t.Fatalf("unexpected result: %#v", out)
	}
}

func TestCallUnknownServer(t *testing.T) {
	r := New()
	if _, err := r.Call(context.Background(), "nope", "t", nil); err == nil {
		t.Fatal("expected error for unknown server")
	}
}

func TestCallRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": 1,
			"error": map[string]any{"code": -32601, "message": "method not found"},
		})
	}))
	defer srv.Close()

	r := New(WithServer("s", srv.URL))
	if _, err := r.Call(context.Background(), "s", "t", nil); err == nil {
		t.Fatal("expected JSON-RPC error to surface")
	}
}
