// Package tlnmcp implements tln's ToolResolver over the Model Context Protocol.
//
// It is the plugin that lets a tln host dispatch `mcp` / `collect` / `enrich`
// / `remediate` tool calls to real MCP servers, so the tln language core stays
// transport-free. Wire it in with:
//
//	r := tlnmcp.New(tlnmcp.WithServer("inventory", "https://mcp.example.com/rpc"))
//	tln.Run(ctx, prog, tln.WithToolResolver(r))
package tlnmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/opentalon/tln-language/pkg/tln"
)

// Resolver dispatches (server, tool, args) triples to MCP servers over HTTP
// JSON-RPC 2.0 using the protocol's `tools/call` method. Each named server maps
// to one endpoint URL. It is safe for concurrent use.
type Resolver struct {
	servers map[string]string
	client  *http.Client
	id      atomic.Int64
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithServer registers an MCP server name and its JSON-RPC endpoint URL. The
// name is what a program references in `mcp "name" "tool" { … }`.
func WithServer(name, endpoint string) Option {
	return func(r *Resolver) { r.servers[name] = endpoint }
}

// WithHTTPClient overrides the default HTTP client (e.g. to add auth transport
// or tune timeouts).
func WithHTTPClient(c *http.Client) Option {
	return func(r *Resolver) { r.client = c }
}

// New builds a Resolver from the given options.
func New(opts ...Option) *Resolver {
	r := &Resolver{
		servers: map[string]string{},
		client:  &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Compile-time proof the plugin satisfies tln's host callback contract.
var _ tln.ToolResolver = (*Resolver)(nil)

type rpcRequest struct {
	JSONRPC string     `json:"jsonrpc"`
	ID      int64      `json:"id"`
	Method  string     `json:"method"`
	Params  callParams `json:"params"`
}

type callParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("mcp error %d: %s", e.Code, e.Message)
}

// Call implements tln.ToolResolver: it invokes `tool` on the named MCP `server`
// with `args` and returns the decoded JSON result. Unknown servers, transport
// failures, and JSON-RPC error responses are surfaced as errors so the tln
// engine can apply the call's on_error policy.
func (r *Resolver) Call(ctx context.Context, server, tool string, args map[string]any) (any, error) {
	endpoint, ok := r.servers[server]
	if !ok {
		return nil, fmt.Errorf("tln-mcp: unknown server %q", server)
	}

	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      r.id.Add(1),
		Method:  "tools/call",
		Params:  callParams{Name: tool, Arguments: args},
	})
	if err != nil {
		return nil, fmt.Errorf("tln-mcp: encode %s/%s: %w", server, tool, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("tln-mcp: build request %s/%s: %w", server, tool, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tln-mcp: call %s/%s: %w", server, tool, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tln-mcp: %s/%s: unexpected status %s", server, tool, resp.Status)
	}

	var out rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("tln-mcp: decode %s/%s: %w", server, tool, err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("tln-mcp: %s/%s: %w", server, tool, out.Error)
	}

	var result any
	if len(out.Result) > 0 {
		if err := json.Unmarshal(out.Result, &result); err != nil {
			return nil, fmt.Errorf("tln-mcp: decode result %s/%s: %w", server, tool, err)
		}
	}
	return result, nil
}
