package addmcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

var testServer = Server{
	Name:    "agend",
	Command: "agend",
	Args:    []string{"mcp"},
}

// --- Standard JSON format tests ---

func TestStandardInstallCreatesNewFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	install := jsonStandardInstall("mcpServers")

	if err := install(path, testServer); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, path)
	servers := m["mcpServers"].(map[string]any)
	agend := servers["agend"].(map[string]any)

	assertEqual(t, agend["command"], "agend")
	assertSlice(t, agend["args"], []string{"mcp"})
}

func TestStandardInstallPreservesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeTestJSON(t, path, map[string]any{
		"mcpServers": map[string]any{
			"other": map[string]any{"command": "other-cmd"},
		},
		"customKey": "keep-me",
	})

	install := jsonStandardInstall("mcpServers")
	if err := install(path, testServer); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, path)
	assertEqual(t, m["customKey"], "keep-me")

	servers := m["mcpServers"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Error("existing server was lost")
	}
	if _, ok := servers["agend"]; !ok {
		t.Error("agend not added")
	}
}

func TestStandardInstallOverwritesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeTestJSON(t, path, map[string]any{
		"mcpServers": map[string]any{
			"agend": map[string]any{"transport": "sse", "url": "https://old"},
		},
	})

	install := jsonStandardInstall("mcpServers")
	if err := install(path, testServer); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, path)
	agend := m["mcpServers"].(map[string]any)["agend"].(map[string]any)
	assertEqual(t, agend["command"], "agend")
	if _, ok := agend["url"]; ok {
		t.Error("old url field should be gone")
	}
}

func TestStandardUninstall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeTestJSON(t, path, map[string]any{
		"mcpServers": map[string]any{
			"agend": map[string]any{"command": "agend"},
			"other": map[string]any{"command": "other"},
		},
	})

	uninstall := jsonStandardUninstall("mcpServers")
	if err := uninstall(path, "agend"); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, path)
	servers := m["mcpServers"].(map[string]any)
	if _, ok := servers["agend"]; ok {
		t.Error("agend should have been removed")
	}
	if _, ok := servers["other"]; !ok {
		t.Error("other server should remain")
	}
}

func TestStandardUninstallNonexistentFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-such-file.json")
	uninstall := jsonStandardUninstall("mcpServers")
	if err := uninstall(path, "agend"); err != nil {
		t.Fatalf("uninstall from missing file should be no-op, got: %v", err)
	}
}

func TestMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte("{broken"), 0644)

	install := jsonStandardInstall("mcpServers")
	if err := install(path, testServer); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

// --- JSONC tests ---

func TestStripJSONC(t *testing.T) {
	input := `{
  // line comment
  "key": "value", // inline comment
  /* block
     comment */
  "url": "https://example.com/path" // after URL
}`
	got := stripJSONC([]byte(input))
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatalf("stripped JSONC should be valid JSON: %v\n%s", err, got)
	}
	assertEqual(t, m["key"], "value")
	assertEqual(t, m["url"], "https://example.com/path")
}

func TestStripJSONCPreservesStrings(t *testing.T) {
	input := `{"key": "has // comment and /* block */ inside"}`
	got := stripJSONC([]byte(input))
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatalf("parse: %v", err)
	}
	assertEqual(t, m["key"], "has // comment and /* block */ inside")
}

// --- VS Code tests ---

func TestVSCodeInstall(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".vscode", "mcp.json")
	if err := vscodeInstall(path, testServer); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, path)
	servers := m["servers"].(map[string]any)
	agend := servers["agend"].(map[string]any)

	assertEqual(t, agend["type"], "stdio")
	assertEqual(t, agend["command"], "agend")
}

func TestVSCodeInstallHTTP(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	s := Server{Name: "remote", URL: "https://example.com/mcp"}
	if err := vscodeInstall(path, s); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, path)
	remote := m["servers"].(map[string]any)["remote"].(map[string]any)
	assertEqual(t, remote["type"], "http")
	assertEqual(t, remote["url"], "https://example.com/mcp")
}

// --- Zed tests ---

func TestZedInstall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	// Pre-populate with existing Zed settings.
	writeTestJSON(t, path, map[string]any{
		"theme": "One Dark",
	})

	if err := zedInstall(path, testServer); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, path)
	assertEqual(t, m["theme"], "One Dark") // preserved
	servers := m["context_servers"].(map[string]any)
	agend := servers["agend"].(map[string]any)
	assertEqual(t, agend["source"], "custom")
	assertEqual(t, agend["command"], "agend")
}

// --- Amazon Q tests ---

func TestAmazonQInstall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "default.json")
	if err := amazonQInstall(path, testServer); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, path)
	servers := m["mcpServers"].([]any)
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	entry := servers[0].(map[string]any)
	assertEqual(t, entry["name"], "agend")
	assertEqual(t, entry["transport"], "stdio")
	assertEqual(t, entry["command"], "agend")
	assertSlice(t, entry["arguments"], []string{"mcp"})
}

func TestAmazonQInstallReplaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), "default.json")
	writeTestJSON(t, path, map[string]any{
		"mcpServers": []any{
			map[string]any{"name": "agend", "transport": "sse", "url": "old"},
			map[string]any{"name": "other", "transport": "stdio", "command": "other"},
		},
	})

	if err := amazonQInstall(path, testServer); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, path)
	servers := m["mcpServers"].([]any)
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
	// "other" should still be there
	found := false
	for _, s := range servers {
		e := s.(map[string]any)
		if e["name"] == "other" {
			found = true
		}
		if e["name"] == "agend" {
			assertEqual(t, e["transport"], "stdio")
			assertEqual(t, e["command"], "agend")
		}
	}
	if !found {
		t.Error("other server was lost")
	}
}

func TestAmazonQUninstall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "default.json")
	writeTestJSON(t, path, map[string]any{
		"mcpServers": []any{
			map[string]any{"name": "agend", "transport": "stdio"},
			map[string]any{"name": "other", "transport": "stdio"},
		},
	})

	if err := amazonQUninstall(path, "agend"); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, path)
	servers := m["mcpServers"].([]any)
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	assertEqual(t, servers[0].(map[string]any)["name"], "other")
}

// --- Goose YAML tests ---

func TestGooseInstall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := gooseInstall(path, testServer); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}

	ext := m["extensions"].(map[string]any)
	agend := ext["agend"].(map[string]any)
	assertEqual(t, agend["cmd"], "agend")
	assertEqual(t, agend["type"], "stdio")
	assertEqual(t, agend["enabled"], true)
}

func TestGooseUninstall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	gooseInstall(path, testServer)
	gooseInstall(path, Server{Name: "other", Command: "other"})

	if err := gooseUninstall(path, "agend"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	var m map[string]any
	yaml.Unmarshal(data, &m)

	ext := m["extensions"].(map[string]any)
	if _, ok := ext["agend"]; ok {
		t.Error("agend should be removed")
	}
	if _, ok := ext["other"]; !ok {
		t.Error("other should remain")
	}
}

// --- Codex TOML tests ---

func TestCodexInstall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := codexInstall(path, testServer); err != nil {
		t.Fatal(err)
	}

	m, err := readTOMLFile(path)
	if err != nil {
		t.Fatal(err)
	}

	servers := m["mcp_servers"].(map[string]any)
	agend := servers["agend"].(map[string]any)
	assertEqual(t, agend["command"], "agend")
}

func TestCodexUninstall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	codexInstall(path, testServer)

	if err := codexUninstall(path, "agend"); err != nil {
		t.Fatal(err)
	}

	m, err := readTOMLFile(path)
	if err != nil {
		t.Fatal(err)
	}
	servers := m["mcp_servers"].(map[string]any)
	if _, ok := servers["agend"]; ok {
		t.Error("agend should be removed")
	}
}

// --- Continue directory tests ---

func TestContinueInstall(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "mcpServers")
	if err := continueInstall(dir, testServer); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, filepath.Join(dir, "agend.json"))
	servers := m["mcpServers"].(map[string]any)
	agend := servers["agend"].(map[string]any)
	assertEqual(t, agend["command"], "agend")
}

func TestContinueUninstall(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "mcpServers")
	continueInstall(dir, testServer)

	if err := continueUninstall(dir, "agend"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "agend.json")); !os.IsNotExist(err) {
		t.Error("agend.json should be deleted")
	}
}

func TestContinueUninstallNonexistent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "mcpServers")
	if err := continueUninstall(dir, "agend"); err != nil {
		t.Fatalf("should be no-op: %v", err)
	}
}

// --- Install/Uninstall public API tests ---

func TestInstallUnknownAgent(t *testing.T) {
	results := Install(testServer, []Agent{"nonexistent"})
	if len(results) != 1 || results[0].Err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestServerWithEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	s := Server{
		Name:    "test",
		Command: "test-cmd",
		Env:     map[string]string{"API_KEY": "secret"},
	}
	install := jsonStandardInstall("mcpServers")
	if err := install(path, s); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, path)
	entry := m["mcpServers"].(map[string]any)["test"].(map[string]any)
	env := entry["env"].(map[string]any)
	assertEqual(t, env["API_KEY"], "secret")
}

func TestHTTPServer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	s := Server{
		Name:    "remote",
		URL:     "https://example.com/mcp",
		Headers: map[string]string{"Authorization": "Bearer token"},
	}
	install := jsonStandardInstall("mcpServers")
	if err := install(path, s); err != nil {
		t.Fatal(err)
	}

	m := readJSON(t, path)
	entry := m["mcpServers"].(map[string]any)["remote"].(map[string]any)
	assertEqual(t, entry["url"], "https://example.com/mcp")
	headers := entry["headers"].(map[string]any)
	assertEqual(t, headers["Authorization"], "Bearer token")
}

// --- helpers ---

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return m
}

func writeTestJSON(t *testing.T, path string, v any) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.Marshal(v)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func assertSlice(t *testing.T, got any, want []string) {
	t.Helper()
	arr, ok := got.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", got)
	}
	if len(arr) != len(want) {
		t.Fatalf("len = %d, want %d", len(arr), len(want))
	}
	for i, v := range arr {
		if v != want[i] {
			t.Errorf("[%d] = %v, want %v", i, v, want[i])
		}
	}
}
