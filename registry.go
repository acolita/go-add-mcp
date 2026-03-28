package addmcp

import (
	"os"
	"path/filepath"
	"runtime"
)

// agentDef is the internal definition for an MCP client agent.
type agentDef struct {
	paths     func(o *options) []string
	detect    func() bool
	install   func(path string, server Server) error
	uninstall func(path string, name string) error
}

var registry = map[Agent]agentDef{
	ClaudeDesktop: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return nil // Claude Desktop has no project scope
			}
			home, _ := os.UserHomeDir()
			switch runtime.GOOS {
			case "darwin":
				return []string{filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")}
			case "linux":
				return []string{filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")}
			case "windows":
				if ad := os.Getenv("APPDATA"); ad != "" {
					return []string{filepath.Join(ad, "Claude", "claude_desktop_config.json")}
				}
			}
			return nil
		},
		detect: func() bool {
			home, _ := os.UserHomeDir()
			switch runtime.GOOS {
			case "darwin":
				return dirExists(filepath.Join(home, "Library", "Application Support", "Claude"))
			case "linux":
				return dirExists(filepath.Join(home, ".config", "Claude"))
			case "windows":
				if ad := os.Getenv("APPDATA"); ad != "" {
					return dirExists(filepath.Join(ad, "Claude"))
				}
			}
			return false
		},
		install:   jsonStandardInstall("mcpServers"),
		uninstall: jsonStandardUninstall("mcpServers"),
	},

	ClaudeCode: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(o), ".mcp.json")}
			}
			home, _ := os.UserHomeDir()
			return []string{filepath.Join(home, ".claude.json")}
		},
		detect: func() bool {
			return commandExists("claude")
		},
		install:   jsonStandardInstall("mcpServers"),
		uninstall: jsonStandardUninstall("mcpServers"),
	},

	Cursor: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(o), ".cursor", "mcp.json")}
			}
			home, _ := os.UserHomeDir()
			return []string{filepath.Join(home, ".cursor", "mcp.json")}
		},
		detect: func() bool {
			home, _ := os.UserHomeDir()
			return dirExists(filepath.Join(home, ".cursor"))
		},
		install:   jsonStandardInstall("mcpServers"),
		uninstall: jsonStandardUninstall("mcpServers"),
	},

	Windsurf: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return nil // Windsurf has no documented project scope
			}
			home, _ := os.UserHomeDir()
			return []string{filepath.Join(home, ".codeium", "windsurf", "mcp_config.json")}
		},
		detect: func() bool {
			home, _ := os.UserHomeDir()
			return dirExists(filepath.Join(home, ".codeium", "windsurf"))
		},
		install:   jsonStandardInstall("mcpServers"),
		uninstall: jsonStandardUninstall("mcpServers"),
	},

	VSCode: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(o), ".vscode", "mcp.json")}
			}
			// VS Code global MCP config is managed via the "MCP: Open User Configuration"
			// command palette action. There's no well-defined cross-platform file path
			// for global MCP config, so we only support project scope.
			return nil
		},
		detect: func() bool {
			return commandExists("code")
		},
		install:   vscodeInstall,
		uninstall: jsonStandardUninstall("servers"),
	},

	Zed: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(o), ".zed", "settings.json")}
			}
			home, _ := os.UserHomeDir()
			switch runtime.GOOS {
			case "darwin", "linux":
				return []string{filepath.Join(home, ".config", "zed", "settings.json")}
			case "windows":
				if ad := os.Getenv("APPDATA"); ad != "" {
					return []string{filepath.Join(ad, "Zed", "settings.json")}
				}
			}
			return nil
		},
		detect: func() bool {
			home, _ := os.UserHomeDir()
			return dirExists(filepath.Join(home, ".config", "zed")) || commandExists("zed")
		},
		install:   zedInstall,
		uninstall: jsonStandardUninstall("context_servers"),
	},

	JetBrains: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(o), ".junie", "mcp", "mcp.json")}
			}
			home, _ := os.UserHomeDir()
			return []string{filepath.Join(home, ".junie", "mcp", "mcp.json")}
		},
		detect: func() bool {
			home, _ := os.UserHomeDir()
			return dirExists(filepath.Join(home, ".junie"))
		},
		install:   jsonStandardInstall("mcpServers"),
		uninstall: jsonStandardUninstall("mcpServers"),
	},

	Cline: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return nil // Cline has no project scope
			}
			return vsCodeExtPaths("saoudrizwan.claude-dev", "cline_mcp_settings.json")
		},
		detect: func() bool {
			for _, p := range vsCodeExtPaths("saoudrizwan.claude-dev", "cline_mcp_settings.json") {
				if dirExists(filepath.Dir(p)) {
					return true
				}
			}
			return false
		},
		install:   jsonStandardInstall("mcpServers"),
		uninstall: jsonStandardUninstall("mcpServers"),
	},

	RooCode: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(o), ".roo", "mcp.json")}
			}
			return vsCodeExtPaths("rooveterinaryinc.roo-cline", "mcp_settings.json")
		},
		detect: func() bool {
			for _, p := range vsCodeExtPaths("rooveterinaryinc.roo-cline", "mcp_settings.json") {
				if dirExists(filepath.Dir(p)) {
					return true
				}
			}
			return false
		},
		install:   jsonStandardInstall("mcpServers"),
		uninstall: jsonStandardUninstall("mcpServers"),
	},

	Gemini: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(o), ".gemini", "settings.json")}
			}
			home, _ := os.UserHomeDir()
			return []string{filepath.Join(home, ".gemini", "settings.json")}
		},
		detect: func() bool {
			return commandExists("gemini")
		},
		install:   jsonStandardInstall("mcpServers"),
		uninstall: jsonStandardUninstall("mcpServers"),
	},

	AmazonQ: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(o), ".amazonq", "default.json")}
			}
			home, _ := os.UserHomeDir()
			return []string{filepath.Join(home, ".aws", "amazonq", "default.json")}
		},
		detect: func() bool {
			home, _ := os.UserHomeDir()
			return dirExists(filepath.Join(home, ".aws", "amazonq")) || commandExists("q")
		},
		install:   amazonQInstall,
		uninstall: amazonQUninstall,
	},

	Codex: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(o), ".codex", "config.toml")}
			}
			home, _ := os.UserHomeDir()
			return []string{filepath.Join(home, ".codex", "config.toml")}
		},
		detect: func() bool {
			return commandExists("codex")
		},
		install:   codexInstall,
		uninstall: codexUninstall,
	},

	Goose: {
		paths: func(o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(o), ".goose", "config.yaml")}
			}
			home, _ := os.UserHomeDir()
			return []string{filepath.Join(home, ".config", "goose", "config.yaml")}
		},
		detect: func() bool {
			return commandExists("goose")
		},
		install:   gooseInstall,
		uninstall: gooseUninstall,
	},

	Continue: {
		paths: func(o *options) []string {
			// Continue uses a directory of config files, not a single file.
			// The path returned is the directory; install/uninstall write individual files.
			if o.scope == Project {
				return []string{filepath.Join(projectDir(o), ".continue", "mcpServers")}
			}
			home, _ := os.UserHomeDir()
			return []string{filepath.Join(home, ".continue", "mcpServers")}
		},
		detect: func() bool {
			home, _ := os.UserHomeDir()
			return dirExists(filepath.Join(home, ".continue"))
		},
		install:   continueInstall,
		uninstall: continueUninstall,
	},
}

// vsCodeExtPaths returns the globalStorage settings path for a VS Code extension.
func vsCodeExtPaths(extensionID, filename string) []string {
	home, _ := os.UserHomeDir()
	rel := filepath.Join("Code", "User", "globalStorage", extensionID, "settings", filename)
	switch runtime.GOOS {
	case "darwin":
		return []string{filepath.Join(home, "Library", "Application Support", rel)}
	case "linux":
		return []string{filepath.Join(home, ".config", rel)}
	case "windows":
		if ad := os.Getenv("APPDATA"); ad != "" {
			return []string{filepath.Join(ad, rel)}
		}
	}
	return nil
}

func projectDir(o *options) string {
	if o.projectDir != "" {
		return o.projectDir
	}
	dir, _ := os.Getwd()
	return dir
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
