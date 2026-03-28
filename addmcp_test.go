package addmcp

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

var testServer = Server{
	Name:    "agend",
	Command: "agend",
	Args:    []string{"mcp"},
}

// ==================== Pure transform tests ====================
// No IO, no filesystem — just inputs and outputs.

func TestTransformStdInstallEmpty(t *testing.T) {
	m := transformStdInstall("mcpServers")(map[string]any{}, testServer)
	assertStdServer(t, m, "mcpServers")
}

func TestTransformStdInstallPreserves(t *testing.T) {
	existing := map[string]any{
		"mcpServers": map[string]any{
			"other": map[string]any{"command": "other-cmd"},
		},
		"customKey": "keep",
	}
	m := transformStdInstall("mcpServers")(existing, testServer)
	assertEqual(t, m["customKey"], "keep")
	if _, ok := m["mcpServers"].(map[string]any)["other"]; !ok {
		t.Error("existing server lost")
	}
	assertStdServer(t, m, "mcpServers")
}

func TestTransformStdInstallOverwrites(t *testing.T) {
	existing := map[string]any{
		"mcpServers": map[string]any{
			"agend": map[string]any{"transport": "sse", "url": "old"},
		},
	}
	m := transformStdInstall("mcpServers")(existing, testServer)
	agend := m["mcpServers"].(map[string]any)["agend"].(map[string]any)
	assertEqual(t, agend["command"], "agend")
	if _, ok := agend["url"]; ok {
		t.Error("old url should be gone")
	}
}

func TestTransformStdUninstall(t *testing.T) {
	existing := map[string]any{
		"mcpServers": map[string]any{
			"agend": map[string]any{"command": "agend"},
			"other": map[string]any{"command": "other"},
		},
	}
	m := transformStdUninstall("mcpServers")(existing, "agend")
	servers := m["mcpServers"].(map[string]any)
	if _, ok := servers["agend"]; ok {
		t.Error("agend should be removed")
	}
	if _, ok := servers["other"]; !ok {
		t.Error("other should remain")
	}
}

func TestTransformStdUninstallEmpty(t *testing.T) {
	m := transformStdUninstall("mcpServers")(map[string]any{}, "agend")
	if m == nil {
		t.Error("should return non-nil map")
	}
}

func TestTransformVSCode(t *testing.T) {
	m := transformVSCodeInstall(map[string]any{}, testServer)
	agend := m["servers"].(map[string]any)["agend"].(map[string]any)
	assertEqual(t, agend["type"], "stdio")
	assertEqual(t, agend["command"], "agend")
}

func TestTransformVSCodeHTTP(t *testing.T) {
	s := Server{Name: "remote", URL: "https://example.com/mcp"}
	m := transformVSCodeInstall(map[string]any{}, s)
	entry := m["servers"].(map[string]any)["remote"].(map[string]any)
	assertEqual(t, entry["type"], "http")
	assertEqual(t, entry["url"], "https://example.com/mcp")
}

func TestTransformZed(t *testing.T) {
	existing := map[string]any{"theme": "One Dark"}
	m := transformZedInstall(existing, testServer)
	assertEqual(t, m["theme"], "One Dark")
	agend := m["context_servers"].(map[string]any)["agend"].(map[string]any)
	assertEqual(t, agend["source"], "custom")
	assertEqual(t, agend["command"], "agend")
}

// Amazon Q uses standard mcpServers object format (verified from official schema).
func TestTransformAmazonQInstall(t *testing.T) {
	m := transformStdInstall("mcpServers")(map[string]any{}, testServer)
	assertStdServer(t, m, "mcpServers")
}

func TestTransformGoose(t *testing.T) {
	m := transformGooseInstall(map[string]any{}, testServer)
	agend := m["extensions"].(map[string]any)["agend"].(map[string]any)
	assertEqual(t, agend["cmd"], "agend")
	assertEqual(t, agend["type"], "stdio")
	assertEqual(t, agend["enabled"], true)
}

func TestTransformGooseUninstall(t *testing.T) {
	m := map[string]any{
		"extensions": map[string]any{
			"agend": map[string]any{"cmd": "agend"},
			"other": map[string]any{"cmd": "other"},
		},
	}
	m = transformGooseUninstall(m, "agend")
	ext := m["extensions"].(map[string]any)
	if _, ok := ext["agend"]; ok {
		t.Error("agend should be removed")
	}
	if _, ok := ext["other"]; !ok {
		t.Error("other should remain")
	}
}

func TestTransformCodex(t *testing.T) {
	m := transformCodexInstall(map[string]any{}, testServer)
	agend := m["mcp_servers"].(map[string]any)["agend"].(map[string]any)
	assertEqual(t, agend["command"], "agend")
}

func TestTransformCodexUninstall(t *testing.T) {
	m := map[string]any{"mcp_servers": map[string]any{"agend": map[string]any{"command": "agend"}}}
	m = transformCodexUninstall(m, "agend")
	if _, ok := m["mcp_servers"].(map[string]any)["agend"]; ok {
		t.Error("should be removed")
	}
}

func TestTransformContinue(t *testing.T) {
	m := transformContinueInstall(nil, testServer)
	agend := m["mcpServers"].(map[string]any)["agend"].(map[string]any)
	assertEqual(t, agend["command"], "agend")
}

func TestTransformWithEnv(t *testing.T) {
	s := Server{Name: "test", Command: "cmd", Env: map[string]string{"KEY": "val"}}
	m := transformStdInstall("mcpServers")(map[string]any{}, s)
	entry := m["mcpServers"].(map[string]any)["test"].(map[string]any)
	env := entry["env"].(map[string]string)
	assertEqual(t, env["KEY"], "val")
}

func TestTransformHTTPServer(t *testing.T) {
	s := Server{Name: "r", URL: "https://x.com/mcp", Headers: map[string]string{"Auth": "Bearer t"}}
	m := transformStdInstall("mcpServers")(map[string]any{}, s)
	entry := m["mcpServers"].(map[string]any)["r"].(map[string]any)
	assertEqual(t, entry["url"], "https://x.com/mcp")
	headers := entry["headers"].(map[string]string)
	assertEqual(t, headers["Auth"], "Bearer t")
}

// Amazon Q HTTP + env uses standard format (covered by TestTransformHTTPServer)

// --- Goose HTTP + env + nil coverage ---

func TestTransformGooseHTTP(t *testing.T) {
	s := Server{Name: "r", URL: "https://x.com/mcp", Headers: map[string]string{"Auth": "tok"}, Env: map[string]string{"K": "V"}}
	m := transformGooseInstall(map[string]any{}, s)
	entry := m["extensions"].(map[string]any)["r"].(map[string]any)
	assertEqual(t, entry["type"], "sse")
	assertEqual(t, entry["uri"], "https://x.com/mcp")
	assertEqual(t, entry["headers"].(map[string]string)["Auth"], "tok")
	assertEqual(t, entry["envs"].(map[string]string)["K"], "V")
}

func TestTransformGooseInstallWithEnv(t *testing.T) {
	s := Server{Name: "x", Command: "x", Env: map[string]string{"K": "V"}}
	m := transformGooseInstall(map[string]any{}, s)
	entry := m["extensions"].(map[string]any)["x"].(map[string]any)
	assertEqual(t, entry["envs"].(map[string]string)["K"], "V")
}

func TestTransformGooseUninstallEmpty(t *testing.T) {
	m := transformGooseUninstall(map[string]any{}, "agend")
	if m == nil {
		t.Error("should return non-nil map")
	}
}

// --- Codex HTTP + env + nil coverage ---

func TestTransformCodexHTTP(t *testing.T) {
	s := Server{Name: "r", URL: "https://x.com/mcp", Headers: map[string]string{"Auth": "tok"}, Env: map[string]string{"K": "V"}}
	m := transformCodexInstall(map[string]any{}, s)
	entry := m["mcp_servers"].(map[string]any)["r"].(map[string]any)
	assertEqual(t, entry["url"], "https://x.com/mcp")
	assertEqual(t, entry["http_headers"].(map[string]string)["Auth"], "tok")
	assertEqual(t, entry["env"].(map[string]string)["K"], "V")
}

func TestTransformCodexInstallWithEnv(t *testing.T) {
	s := Server{Name: "x", Command: "x", Env: map[string]string{"K": "V"}}
	m := transformCodexInstall(map[string]any{}, s)
	entry := m["mcp_servers"].(map[string]any)["x"].(map[string]any)
	assertEqual(t, entry["env"].(map[string]string)["K"], "V")
}

func TestTransformCodexUninstallEmpty(t *testing.T) {
	m := transformCodexUninstall(map[string]any{}, "agend")
	if m == nil {
		t.Error("should return non-nil map")
	}
}

// --- stripJSONC edge cases ---

func TestStripJSONCUnterminatedBlock(t *testing.T) {
	input := `{"key": "value" /* unterminated`
	got := stripJSONC([]byte(input))
	// Should not panic; block comment consumed to end
	if len(got) == 0 {
		t.Error("should produce output")
	}
}

func TestStripJSONCEscapedQuotes(t *testing.T) {
	input := `{"key": "value with \"escaped\" quotes"}`
	got := stripJSONC([]byte(input))
	var m map[string]any
	if err := json.Unmarshal(got, &m); err != nil {
		t.Fatalf("parse: %v", err)
	}
	assertEqual(t, m["key"], `value with "escaped" quotes`)
}

func TestStripJSONCUnterminatedString(t *testing.T) {
	input := `{"key": "unterminated`
	got := stripJSONC([]byte(input))
	if len(got) == 0 {
		t.Error("should produce output")
	}
}

// ==================== JSONC tests ====================

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

// ==================== Path resolution tests (pure, cross-platform) ====================

func TestPaths(t *testing.T) {
	tests := []struct {
		name  string
		agent Agent
		plat  Platform
		scope Scope
		proj  string
		want  []string
	}{
		// Claude Desktop
		{
			name: "ClaudeDesktop/darwin", agent: ClaudeDesktop,
			plat: Platform{GOOS: "darwin", HomeDir: "/Users/a"},
			want: []string{filepath.Join("/Users/a", "Library", "Application Support", "Claude", "claude_desktop_config.json")},
		},
		{
			name: "ClaudeDesktop/linux", agent: ClaudeDesktop,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			want: []string{filepath.Join("/home/a", ".config", "Claude", "claude_desktop_config.json")},
		},
		{
			name: "ClaudeDesktop/windows", agent: ClaudeDesktop,
			plat: Platform{GOOS: "windows", AppData: "/appdata"},
			want: []string{filepath.Join("/appdata", "Claude", "claude_desktop_config.json")},
		},
		{
			name: "ClaudeDesktop/project=nil", agent: ClaudeDesktop,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"}, scope: Project,
			want: nil,
		},

		// Claude Code
		{
			name: "ClaudeCode/global", agent: ClaudeCode,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			want: []string{filepath.Join("/home/a", ".claude.json")},
		},
		{
			name: "ClaudeCode/project", agent: ClaudeCode,
			plat: Platform{GOOS: "linux", WorkingDir: "/work"}, scope: Project,
			want: []string{filepath.Join("/work", ".mcp.json")},
		},
		{
			name: "ClaudeCode/project+dir", agent: ClaudeCode,
			plat: Platform{GOOS: "linux"}, scope: Project, proj: "/my/proj",
			want: []string{filepath.Join("/my/proj", ".mcp.json")},
		},

		// Cursor
		{
			name: "Cursor/global", agent: Cursor,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			want: []string{filepath.Join("/home/a", ".cursor", "mcp.json")},
		},
		{
			name: "Cursor/project", agent: Cursor,
			plat: Platform{GOOS: "linux", WorkingDir: "/work"}, scope: Project,
			want: []string{filepath.Join("/work", ".cursor", "mcp.json")},
		},

		// Windsurf (global only)
		{
			name: "Windsurf/global", agent: Windsurf,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			want: []string{filepath.Join("/home/a", ".codeium", "windsurf", "mcp_config.json")},
		},
		{
			name: "Windsurf/project=nil", agent: Windsurf,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"}, scope: Project,
			want: nil,
		},

		// VS Code (project only)
		{
			name: "VSCode/project", agent: VSCode,
			plat: Platform{GOOS: "linux", WorkingDir: "/work"}, scope: Project,
			want: []string{filepath.Join("/work", ".vscode", "mcp.json")},
		},
		{
			name: "VSCode/global=nil", agent: VSCode,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			want: nil,
		},

		// Zed
		{
			name: "Zed/linux", agent: Zed,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			want: []string{filepath.Join("/home/a", ".config", "zed", "settings.json")},
		},
		{
			name: "Zed/windows", agent: Zed,
			plat: Platform{GOOS: "windows", AppData: "/appdata"},
			want: []string{filepath.Join("/appdata", "Zed", "settings.json")},
		},
		{
			name: "Zed/project", agent: Zed,
			plat: Platform{GOOS: "linux", WorkingDir: "/work"}, scope: Project,
			want: []string{filepath.Join("/work", ".zed", "settings.json")},
		},

		// Codex
		{
			name: "Codex/global", agent: Codex,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			want: []string{filepath.Join("/home/a", ".codex", "config.toml")},
		},

		// Goose
		{
			name: "Goose/global", agent: Goose,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			want: []string{filepath.Join("/home/a", ".config", "goose", "config.yaml")},
		},

		// Continue
		{
			name: "Continue/global", agent: Continue,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			want: []string{filepath.Join("/home/a", ".continue", "mcpServers")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := registry[tt.agent]
			o := &options{scope: tt.scope, projectDir: tt.proj}
			got := def.paths(tt.plat, o)
			assertPaths(t, got, tt.want)
		})
	}
}

// ==================== Detection tests (pure, with fakeDetector) ====================

func TestDetect(t *testing.T) {
	tests := []struct {
		name  string
		agent Agent
		plat  Platform
		det   fakeDetector
		want  bool
	}{
		{
			name: "ClaudeDesktop/found", agent: ClaudeDesktop,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			det:  fakeDetector{dirs: map[string]bool{filepath.Join("/home/a", ".config", "Claude"): true}},
			want: true,
		},
		{
			name: "ClaudeDesktop/missing", agent: ClaudeDesktop,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			det:  fakeDetector{},
			want: false,
		},
		{
			name: "ClaudeCode/found", agent: ClaudeCode,
			plat: Platform{GOOS: "linux"},
			det:  fakeDetector{commands: map[string]bool{"claude": true}},
			want: true,
		},
		{
			name: "ClaudeCode/missing", agent: ClaudeCode,
			plat: Platform{GOOS: "linux"},
			det:  fakeDetector{},
			want: false,
		},
		{
			name: "Zed/dir", agent: Zed,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			det:  fakeDetector{dirs: map[string]bool{filepath.Join("/home/a", ".config", "zed"): true}},
			want: true,
		},
		{
			name: "Zed/command", agent: Zed,
			plat: Platform{GOOS: "linux", HomeDir: "/home/a"},
			det:  fakeDetector{commands: map[string]bool{"zed": true}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := registry[tt.agent]
			got := def.detect(tt.plat, tt.det)
			if got != tt.want {
				t.Errorf("detect = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectWithMultipleAgents(t *testing.T) {
	plat := Platform{GOOS: "linux", HomeDir: "/home/a"}
	det := fakeDetector{
		commands: map[string]bool{"claude": true, "gemini": true},
		dirs:     map[string]bool{filepath.Join("/home/a", ".cursor"): true},
	}
	found := detectWith(det, plat)
	agents := map[Agent]bool{}
	for _, a := range found {
		agents[a] = true
	}
	for _, want := range []Agent{ClaudeCode, Cursor, Gemini} {
		if !agents[want] {
			t.Errorf("expected %s to be detected", want)
		}
	}
}

// ==================== Integration tests with memFS ====================

func TestInstallClaudeDesktopMemFS(t *testing.T) {
	fsys := newMemFS()
	plat := Platform{GOOS: "darwin", HomeDir: "/Users/alice"}
	results := installWith(fsys, plat, testServer, []Agent{ClaudeDesktop})

	if len(results) != 1 || results[0].Err != nil {
		t.Fatalf("unexpected: %+v", results)
	}
	path := filepath.Join("/Users/alice", "Library", "Application Support", "Claude", "claude_desktop_config.json")
	assertEqual(t, results[0].Path, path)

	m := parseMemJSON(t, fsys, path)
	assertStdServer(t, m, "mcpServers")
}

func TestInstallPreservesExistingMemFS(t *testing.T) {
	fsys := newMemFS()
	path := filepath.Join("/home/a", ".claude.json")
	existing, _ := json.Marshal(map[string]any{
		"mcpServers": map[string]any{"other": map[string]any{"command": "other"}},
		"customKey":  "keep",
	})
	fsys.putJSON(path, existing)

	plat := Platform{GOOS: "linux", HomeDir: "/home/a"}
	results := installWith(fsys, plat, testServer, []Agent{ClaudeCode})

	if results[0].Err != nil {
		t.Fatal(results[0].Err)
	}
	m := parseMemJSON(t, fsys, path)
	assertEqual(t, m["customKey"], "keep")
	if _, ok := m["mcpServers"].(map[string]any)["other"]; !ok {
		t.Error("existing server lost")
	}
	assertStdServer(t, m, "mcpServers")
}

func TestInstallVSCodeMemFS(t *testing.T) {
	fsys := newMemFS()
	plat := Platform{GOOS: "linux", WorkingDir: "/proj"}
	results := installWith(fsys, plat, testServer, []Agent{VSCode}, WithScope(Project))

	if results[0].Err != nil {
		t.Fatal(results[0].Err)
	}
	path := filepath.Join("/proj", ".vscode", "mcp.json")
	m := parseMemJSON(t, fsys, path)
	agend := m["servers"].(map[string]any)["agend"].(map[string]any)
	assertEqual(t, agend["type"], "stdio")
	assertEqual(t, agend["command"], "agend")
}

func TestInstallGooseMemFS(t *testing.T) {
	fsys := newMemFS()
	plat := Platform{GOOS: "linux", HomeDir: "/home/a"}
	results := installWith(fsys, plat, testServer, []Agent{Goose})

	if results[0].Err != nil {
		t.Fatal(results[0].Err)
	}
	path := filepath.Join("/home/a", ".config", "goose", "config.yaml")
	var m map[string]any
	if err := yaml.Unmarshal(fsys.files[path], &m); err != nil {
		t.Fatal(err)
	}
	agend := m["extensions"].(map[string]any)["agend"].(map[string]any)
	assertEqual(t, agend["cmd"], "agend")
	assertEqual(t, agend["type"], "stdio")
}

func TestInstallCodexMemFS(t *testing.T) {
	fsys := newMemFS()
	plat := Platform{GOOS: "linux", HomeDir: "/home/a"}
	results := installWith(fsys, plat, testServer, []Agent{Codex})

	if results[0].Err != nil {
		t.Fatal(results[0].Err)
	}
	path := filepath.Join("/home/a", ".codex", "config.toml")
	var m map[string]any
	if err := toml.Unmarshal(fsys.files[path], &m); err != nil {
		t.Fatal(err)
	}
	servers := m["mcp_servers"].(map[string]any)
	agend := servers["agend"].(map[string]any)
	assertEqual(t, agend["command"], "agend")
}

func TestInstallContinueMemFS(t *testing.T) {
	fsys := newMemFS()
	plat := Platform{GOOS: "linux", HomeDir: "/home/a"}
	results := installWith(fsys, plat, testServer, []Agent{Continue})

	if results[0].Err != nil {
		t.Fatal(results[0].Err)
	}
	dir := filepath.Join("/home/a", ".continue", "mcpServers")
	filePath := filepath.Join(dir, "agend.json")
	m := parseMemJSON(t, fsys, filePath)
	agend := m["mcpServers"].(map[string]any)["agend"].(map[string]any)
	assertEqual(t, agend["command"], "agend")
}

func TestUninstallNonexistentMemFS(t *testing.T) {
	fsys := newMemFS()
	plat := Platform{GOOS: "linux", HomeDir: "/home/a"}
	results := uninstallWith(fsys, plat, "agend", []Agent{ClaudeDesktop})

	if results[0].Err != nil {
		t.Fatalf("uninstall from missing file should be no-op: %v", results[0].Err)
	}
}

func TestUninstallContinueMemFS(t *testing.T) {
	fsys := newMemFS()
	plat := Platform{GOOS: "linux", HomeDir: "/home/a"}

	// Install first
	installWith(fsys, plat, testServer, []Agent{Continue})

	// Then uninstall
	results := uninstallWith(fsys, plat, "agend", []Agent{Continue})
	if results[0].Err != nil {
		t.Fatal(results[0].Err)
	}

	filePath := filepath.Join("/home/a", ".continue", "mcpServers", "agend.json")
	if _, ok := fsys.files[filePath]; ok {
		t.Error("file should be removed")
	}
}

func TestInstallUnknownAgent(t *testing.T) {
	fsys := newMemFS()
	plat := Platform{GOOS: "linux", HomeDir: "/home/a"}
	results := installWith(fsys, plat, testServer, []Agent{"nonexistent"})
	if len(results) != 1 || results[0].Err == nil {
		t.Fatal("expected error for unknown agent")
	}
}

func TestMalformedJSONMemFS(t *testing.T) {
	fsys := newMemFS()
	path := filepath.Join("/home/a", ".config", "Claude", "claude_desktop_config.json")
	fsys.putJSON(path, []byte("{broken"))

	plat := Platform{GOOS: "linux", HomeDir: "/home/a"}
	results := installWith(fsys, plat, testServer, []Agent{ClaudeDesktop})
	if results[0].Err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestJSONCFileMemFS(t *testing.T) {
	fsys := newMemFS()
	path := filepath.Join("/proj", ".vscode", "mcp.json")
	// Pre-populate with JSONC content
	fsys.putJSON(path, []byte(`{
  // existing servers
  "servers": {
    "other": {"type": "stdio", "command": "other"}
  }
}`))

	plat := Platform{GOOS: "linux", WorkingDir: "/proj"}
	results := installWith(fsys, plat, testServer, []Agent{VSCode}, WithScope(Project))
	if results[0].Err != nil {
		t.Fatal(results[0].Err)
	}

	m := parseMemJSON(t, fsys, path)
	servers := m["servers"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Error("existing server lost after JSONC parse")
	}
	agend := servers["agend"].(map[string]any)
	assertEqual(t, agend["type"], "stdio")
}

// ==================== Helpers ====================

func assertStdServer(t *testing.T, m map[string]any, key string) {
	t.Helper()
	servers, ok := m[key].(map[string]any)
	if !ok {
		t.Fatalf("key %q missing or wrong type", key)
	}
	agend, ok := servers["agend"].(map[string]any)
	if !ok {
		t.Fatal("agend entry missing")
	}
	assertEqual(t, agend["command"], "agend")
	assertSlice(t, agend["args"], []string{"mcp"})
}

func parseMemJSON(t *testing.T, fsys *memFS, path string) map[string]any {
	t.Helper()
	data, ok := fsys.files[path]
	if !ok {
		t.Fatalf("file not found in memFS: %s", path)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return m
}

func assertPaths(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("paths: got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("paths[%d]: got %q, want %q", i, got[i], want[i])
		}
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
	switch v := got.(type) {
	case []string:
		if len(v) != len(want) {
			t.Fatalf("len = %d, want %d", len(v), len(want))
		}
		for i := range v {
			if v[i] != want[i] {
				t.Errorf("[%d] = %v, want %v", i, v[i], want[i])
			}
		}
	case []any:
		if len(v) != len(want) {
			t.Fatalf("len = %d, want %d", len(v), len(want))
		}
		for i := range v {
			if v[i] != want[i] {
				t.Errorf("[%d] = %v, want %v", i, v[i], want[i])
			}
		}
	default:
		t.Fatalf("expected []string or []any, got %T", got)
	}
}
