package tlnmcp

import (
	"fmt"

	"github.com/opentalon/tln-language/pkg/tln"
)

// Factory builds an MCP resolver from a connector's config, so tln-mcp can be
// loaded by name from a `mod.tln` / connector (ADR 0012, ADR 0013):
//
//	connector "inventory" via mcp {
//	  endpoint env "INVENTORY_URL"
//	  bearer   env "INVENTORY_TOKEN"     # optional
//	}
//
// The returned resolver answers the connector's own server name, pointed at
// `endpoint`, with an optional bearer token.
func Factory(spec tln.ConnectorSpec) (tln.ToolResolver, error) {
	endpoint := spec.Config["endpoint"]
	if endpoint == "" {
		return nil, fmt.Errorf("tln-mcp: connector %q requires an `endpoint`", spec.Name)
	}
	opts := []Option{WithServer(spec.Name, endpoint)}
	if token := spec.Config["bearer"]; token != "" {
		opts = append(opts, WithBearer(spec.Name, token))
	}
	return New(opts...), nil
}

// Factory satisfies tln.PluginFactory.
var _ tln.PluginFactory = Factory
