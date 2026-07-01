package main

import (
	"encoding/json"
	"os"

	"github.com/luraproject/lura/v2/config"
	mcpgateway "github.com/stayforlong/krakend-mcp-gateway"
)

// selfHandledNamespaces lists extra_config namespaces whose plugin fully
// replaces an endpoint's handler and therefore never dials the endpoint's
// configured backend (see e.g. mcpgateway.Namespace: krakend-mcp-gateway
// intercepts the request before any backend is ever contacted). Lura's own
// config validation still rejects any endpoint with zero backends, so
// autoBackendParser injects a synthetic, never-dialed one for any endpoint
// that declares one of these namespaces with an empty backend list —
// sparing every config repo from hand-writing that boilerplate.
var selfHandledNamespaces = []string{
	mcpgateway.Namespace,
}

// autoBackendParser wraps another config.Parser, patching the rendered
// config file before handing it off. It never runs Init()/validation
// itself - it only rewrites the raw JSON and delegates parsing (and thus
// validation) to the wrapped Parser.
type autoBackendParser struct {
	config.Parser
}

func (p autoBackendParser) Parse(configFile string) (config.ServiceConfig, error) {
	raw, err := os.ReadFile(configFile)
	if err != nil {
		return p.Parser.Parse(configFile)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		// Malformed JSON: let the wrapped parser produce the real error.
		return p.Parser.Parse(configFile)
	}

	if !patchEmptyBackends(doc) {
		return p.Parser.Parse(configFile)
	}

	patched, err := json.Marshal(doc)
	if err != nil {
		return p.Parser.Parse(configFile)
	}

	tmpfile, err := os.CreateTemp("", "krakend_autobackend_*.json")
	if err != nil {
		return p.Parser.Parse(configFile)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(patched); err != nil {
		tmpfile.Close() //nolint:errcheck
		return p.Parser.Parse(configFile)
	}
	if err := tmpfile.Close(); err != nil {
		return p.Parser.Parse(configFile)
	}

	return p.Parser.Parse(tmpfile.Name())
}

// patchEmptyBackends injects a synthetic backend into every endpoint that
// declares a self-handled namespace with no backends. Returns true if any
// endpoint was modified, so the caller can skip the temp-file round trip
// when there's nothing to patch.
func patchEmptyBackends(doc map[string]interface{}) bool {
	endpoints, ok := doc["endpoints"].([]interface{})
	if !ok {
		return false
	}

	patched := false
	for _, e := range endpoints {
		endpoint, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		if !hasSelfHandledNamespace(endpoint) || hasBackends(endpoint) {
			continue
		}
		endpoint["backend"] = []interface{}{
			map[string]interface{}{
				"url_pattern": "/unused",
				"host":        []interface{}{"http://unused.invalid"},
			},
		}
		patched = true
	}
	return patched
}

func hasSelfHandledNamespace(endpoint map[string]interface{}) bool {
	extra, ok := endpoint["extra_config"].(map[string]interface{})
	if !ok {
		return false
	}
	for _, ns := range selfHandledNamespaces {
		if _, ok := extra[ns]; ok {
			return true
		}
	}
	return false
}

func hasBackends(endpoint map[string]interface{}) bool {
	backend, ok := endpoint["backend"].([]interface{})
	return ok && len(backend) > 0
}
