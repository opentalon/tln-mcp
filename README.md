# tln-mcp

[![CI](https://github.com/opentalon/tln-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/opentalon/tln-mcp/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**MCP tool plugin for [tln](https://github.com/opentalon/tln-language).**

tln's language core is transport-free: it decides *which* tool calls fire and
returns them as data, and a host-injected `tln.ToolResolver` performs the IO.
`tln-mcp` is the ready-made resolver that speaks the
[Model Context Protocol](https://modelcontextprotocol.io) — so a host doesn't
hand-roll one.

This is the tools leg of tln's plugin model, mirroring
[`tln-db`](https://github.com/opentalon/tln-db) (storage): core is a pure
language + planner + SPIs, and every IO edge is a plugin.

## Usage

```go
import (
    "github.com/opentalon/tln-language/pkg/tln"
    tlnmcp "github.com/opentalon/tln-mcp"
)

r := tlnmcp.New(
    tlnmcp.WithServer("inventory", "https://mcp.example.com/rpc"),
    tlnmcp.WithServer("slack", "https://slack-mcp.example.com/rpc"),
)

result, err := tln.Run(ctx, program, tln.WithToolResolver(r))
```

A program's `tool "inventory" "list_items" { … }` steps (and `collect` /
`enrich` / `remediate`) then dispatch to the named MCP server via JSON-RPC
`tools/call`. The tool-call verb is the plugin-neutral **`tool`** ([ADR 0012](https://github.com/opentalon/tln-language/blob/master/docs/design/0012-tool-verb-and-connectors.md)) — `mcp` is no longer a keyword; the server name (here `"inventory"`) routes to this resolver.

Instead of wiring servers in Go, a program can define them in tln with a
`connector`, so it runs with no host — endpoints and credentials come from the
environment, never inlined:

```tln
connector "inventory" via mcp {
  endpoint env "INVENTORY_ENDPOINT"
  auth bearer env "INVENTORY_TOKEN"
}
tool "inventory" "list_items" { query "status:defective" }
```

## Status

Minimal HTTP JSON-RPC (`tools/call`) transport implementing `tln.ToolResolver`.
Planned: stdio transport, the MCP `initialize` handshake, and auth options.

## License

Apache 2.0 — see [LICENSE](LICENSE).
