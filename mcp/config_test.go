package mcp

import (
	"path/filepath"
	"testing"
)

func TestLoadMCPConfig(t *testing.T) {
	path := filepath.Join("testdata", "mcpconfig.json")
	config, err := LoadMCPConfig(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if config == nil {
		t.Fatal("config is nil")
	}

	server, ok := config.McpServers["server-name"]
	if !ok {
		t.Fatal("server-name not found in mcpServers")
	}

	if server.Command != "npx" {
		t.Errorf("expected command 'npx', got '%s'", server.Command)
	}

	if len(server.Args) != 2 || server.Args[0] != "-y" || server.Args[1] != "mcp-server" {
		t.Errorf("unexpected args: %#v", server.Args)
	}

	if v, ok := server.Env["API_KEY"]; !ok || v != "value" {
		t.Errorf("expected env API_KEY to be 'value', got '%s'", v)
	}
}
