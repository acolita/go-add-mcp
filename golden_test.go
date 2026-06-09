package addmcp

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

var update = flag.Bool("update", false, "update golden files")

// TestGoldenFiles verifies that installing testServer into an empty config
// produces the exact byte output stored in testdata/<agent>.<ext>.
//
// Run with -update to regenerate golden files:
//
//	go test -run TestGoldenFiles -update
func TestGoldenFiles(t *testing.T) {
	stdioServer := Server{
		Name:    "agend",
		Command: "agend",
		Args:    []string{"mcp"},
	}
	remoteServer := Server{
		Name:    "agend",
		URL:     "https://mcp.example.com/mcp",
		Headers: map[string]string{"Authorization": "Bearer xxx"},
	}

	// stdioFile is the golden for a local (stdio) install; remoteFile is the
	// golden for an HTTP (remote) install. An empty remoteFile means the agent
	// rejects remote servers (stdioOnly) and only the stdio golden applies.
	tests := []struct {
		agent      Agent
		stdioFile  string
		remoteFile string
	}{
		{ClaudeDesktop, "claude-desktop.json", ""},
		{ClaudeCode, "claude-code.json", "claude-code.remote.json"},
		{Cursor, "cursor.json", "cursor.remote.json"},
		{Windsurf, "windsurf.json", "windsurf.remote.json"},
		{VSCode, "vscode.json", "vscode.remote.json"},
		{Zed, "zed.json", "zed.remote.json"},
		{JetBrains, "jetbrains.json", "jetbrains.remote.json"},
		{Cline, "cline.json", "cline.remote.json"},
		{ClineCLI, "cline-cli.json", "cline-cli.remote.json"},
		{RooCode, "roo-code.json", "roo-code.remote.json"},
		{Gemini, "gemini.json", "gemini.remote.json"},
		{AmazonQ, "amazon-q.json", "amazon-q.remote.json"},
		{Codex, "codex.toml", "codex.remote.toml"},
		{Goose, "goose.yaml", "goose.remote.yaml"},
		{Continue, "continue.json", "continue.remote.json"},
		{Antigravity, "antigravity.json", "antigravity.remote.json"},
		{OpenCode, "opencode.json", "opencode.remote.json"},
		{MCPorter, "mcporter.json", "mcporter.remote.json"},
		{GitHubCopilotCLI, "github-copilot-cli.json", "github-copilot-cli.remote.json"},
	}

	for _, tt := range tests {
		def := registry[tt.agent]
		t.Run(string(tt.agent), func(t *testing.T) {
			checkGolden(t, def, stdioServer, tt.stdioFile)
		})
		if tt.remoteFile == "" {
			continue
		}
		t.Run(string(tt.agent)+"/remote", func(t *testing.T) {
			checkGolden(t, def, remoteServer, tt.remoteFile)
		})
	}
}

// checkGolden generates the agent's output for the given server and compares it
// to (or, with -update, writes) the named golden file under testdata/.
func checkGolden(t *testing.T, def agentDef, server Server, file string) {
	t.Helper()
	got := generateOutput(t, def, server)
	goldenPath := filepath.Join("testdata", file)

	if *update {
		os.MkdirAll("testdata", 0755)
		if err := os.WriteFile(goldenPath, got, 0644); err != nil {
			t.Fatal(err)
		}
		t.Logf("updated %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden file missing (run with -update to create): %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("output differs from golden file %s\n\ngot:\n%s\nwant:\n%s",
			goldenPath, got, want)
	}
}

// generateOutput runs the agent's install transform on an empty config
// and serializes it in the agent's native format.
func generateOutput(t *testing.T, def agentDef, server Server) []byte {
	t.Helper()

	switch def.format {
	case formatJSON:
		m := def.installTransform(map[string]any{}, server)
		out, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		return append(out, '\n')

	case formatYAML:
		m := def.installTransform(map[string]any{}, server)
		out, err := yaml.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return out

	case formatTOML:
		m := def.installTransform(map[string]any{}, server)
		out, err := toml.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return out

	case formatContinueDir:
		// Continue writes individual files; golden is the file content.
		m := def.installTransform(nil, server)
		out, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		return append(out, '\n')

	default:
		t.Fatalf("unknown format")
		return nil
	}
}
