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
	server := Server{
		Name:    "agend",
		Command: "agend",
		Args:    []string{"mcp"},
	}

	tests := []struct {
		agent Agent
		file  string // golden filename
	}{
		{ClaudeDesktop, "claude-desktop.json"},
		{ClaudeCode, "claude-code.json"},
		{Cursor, "cursor.json"},
		{Windsurf, "windsurf.json"},
		{VSCode, "vscode.json"},
		{Zed, "zed.json"},
		{JetBrains, "jetbrains.json"},
		{Cline, "cline.json"},
		{RooCode, "roo-code.json"},
		{Gemini, "gemini.json"},
		{AmazonQ, "amazon-q.json"},
		{Codex, "codex.toml"},
		{Goose, "goose.yaml"},
		{Continue, "continue.json"},
		{Antigravity, "antigravity.json"},
		{OpenCode, "opencode.json"},
		{GitHubCopilotCLI, "github-copilot-cli.json"},
	}

	for _, tt := range tests {
		t.Run(string(tt.agent), func(t *testing.T) {
			def := registry[tt.agent]
			got := generateOutput(t, def, server)
			goldenPath := filepath.Join("testdata", tt.file)

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
		})
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
