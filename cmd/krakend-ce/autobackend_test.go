package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luraproject/lura/v2/config"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "krakend.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	return path
}

func TestAutoBackendParser_PatchesSelfHandledEndpointWithNoBackend(t *testing.T) {
	path := writeTempConfig(t, `{
		"version": 3,
		"endpoints": [
			{
				"endpoint": "/mcp",
				"method": "POST",
				"extra_config": {
					"github_com/stayforlong/krakend-mcp-gateway": {}
				}
			}
		]
	}`)

	parser := autoBackendParser{config.NewParser()}
	cfg, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cfg.Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(cfg.Endpoints))
	}
	if len(cfg.Endpoints[0].Backend) != 1 {
		t.Fatalf("expected a synthetic backend to be injected, got %d backends", len(cfg.Endpoints[0].Backend))
	}
}

func TestAutoBackendParser_LeavesUnrelatedEmptyBackendEndpointFailing(t *testing.T) {
	path := writeTempConfig(t, `{
		"version": 3,
		"endpoints": [
			{
				"endpoint": "/no-backend",
				"method": "GET"
			}
		]
	}`)

	parser := autoBackendParser{config.NewParser()}
	if _, err := parser.Parse(path); err == nil {
		t.Fatal("expected an error for a genuinely misconfigured endpoint with no backend and no self-handled namespace")
	}
}

func TestAutoBackendParser_LeavesExistingBackendsUntouched(t *testing.T) {
	path := writeTempConfig(t, `{
		"version": 3,
		"endpoints": [
			{
				"endpoint": "/mcp",
				"method": "POST",
				"extra_config": {
					"github_com/stayforlong/krakend-mcp-gateway": {}
				},
				"backend": [
					{"url_pattern": "/real", "host": ["http://real.internal"]}
				]
			}
		]
	}`)

	parser := autoBackendParser{config.NewParser()}
	cfg, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cfg.Endpoints[0].Backend) != 1 || cfg.Endpoints[0].Backend[0].URLPattern != "/real" {
		t.Fatalf("expected the existing backend to be preserved untouched, got %+v", cfg.Endpoints[0].Backend)
	}
}

func TestPatchEmptyBackends_ReturnsFalseWhenNothingToPatch(t *testing.T) {
	doc := map[string]interface{}{
		"endpoints": []interface{}{
			map[string]interface{}{
				"endpoint": "/plain",
			},
		},
	}
	if patchEmptyBackends(doc) {
		t.Fatal("expected no patch for an endpoint without a self-handled namespace")
	}
}
