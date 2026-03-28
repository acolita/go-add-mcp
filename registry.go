package addmcp

import "path/filepath"

// format identifies the serialization format for an agent's config file.
type format int

const (
	formatJSON format = iota
	formatYAML
	formatTOML
	formatContinueDir // directory of individual JSON files
)

// agentDef is the internal definition for an MCP client agent.
// Path resolution and detection are pure functions over Platform/Detector.
// Config transformation is a pure function over map data.
// IO is handled by doInstall/doUninstall in io.go.
type agentDef struct {
	paths              func(Platform, *options) []string
	detect             func(Platform, Detector) bool
	format             format
	installTransform   func(map[string]any, Server) map[string]any
	uninstallTransform func(map[string]any, string) map[string]any // nil for Continue
}

var registry = map[Agent]agentDef{
	ClaudeDesktop: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return nil
			}
			switch p.GOOS {
			case "darwin":
				return []string{filepath.Join(p.HomeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")}
			case "linux":
				return []string{filepath.Join(p.HomeDir, ".config", "Claude", "claude_desktop_config.json")}
			case "windows":
				if p.AppData != "" {
					return []string{filepath.Join(p.AppData, "Claude", "claude_desktop_config.json")}
				}
			}
			return nil
		},
		detect: func(p Platform, d Detector) bool {
			switch p.GOOS {
			case "darwin":
				return d.DirExists(filepath.Join(p.HomeDir, "Library", "Application Support", "Claude"))
			case "linux":
				return d.DirExists(filepath.Join(p.HomeDir, ".config", "Claude"))
			case "windows":
				if p.AppData != "" {
					return d.DirExists(filepath.Join(p.AppData, "Claude"))
				}
			}
			return false
		},
		format:             formatJSON,
		installTransform:   transformStdInstall("mcpServers"),
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	ClaudeCode: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), ".mcp.json")}
			}
			return []string{filepath.Join(p.HomeDir, ".claude.json")}
		},
		detect: func(_ Platform, d Detector) bool {
			return d.CommandExists("claude")
		},
		format:             formatJSON,
		installTransform:   transformStdInstall("mcpServers"),
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	Cursor: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), ".cursor", "mcp.json")}
			}
			return []string{filepath.Join(p.HomeDir, ".cursor", "mcp.json")}
		},
		detect: func(p Platform, d Detector) bool {
			return d.DirExists(filepath.Join(p.HomeDir, ".cursor"))
		},
		format:             formatJSON,
		installTransform:   transformStdInstall("mcpServers"),
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	Windsurf: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return nil
			}
			return []string{filepath.Join(p.HomeDir, ".codeium", "windsurf", "mcp_config.json")}
		},
		detect: func(p Platform, d Detector) bool {
			return d.DirExists(filepath.Join(p.HomeDir, ".codeium", "windsurf"))
		},
		format:             formatJSON,
		installTransform:   transformStdInstall("mcpServers"),
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	VSCode: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), ".vscode", "mcp.json")}
			}
			return nil // global MCP config managed via Command Palette
		},
		detect: func(_ Platform, d Detector) bool {
			return d.CommandExists("code")
		},
		format:             formatJSON,
		installTransform:   transformVSCodeInstall,
		uninstallTransform: transformStdUninstall("servers"),
	},

	Zed: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), ".zed", "settings.json")}
			}
			switch p.GOOS {
			case "darwin", "linux":
				return []string{filepath.Join(p.HomeDir, ".config", "zed", "settings.json")}
			case "windows":
				if p.AppData != "" {
					return []string{filepath.Join(p.AppData, "Zed", "settings.json")}
				}
			}
			return nil
		},
		detect: func(p Platform, d Detector) bool {
			return d.DirExists(filepath.Join(p.HomeDir, ".config", "zed")) || d.CommandExists("zed")
		},
		format:             formatJSON,
		installTransform:   transformZedInstall,
		uninstallTransform: transformStdUninstall("context_servers"),
	},

	JetBrains: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), ".junie", "mcp", "mcp.json")}
			}
			return []string{filepath.Join(p.HomeDir, ".junie", "mcp", "mcp.json")}
		},
		detect: func(p Platform, d Detector) bool {
			return d.DirExists(filepath.Join(p.HomeDir, ".junie"))
		},
		format:             formatJSON,
		installTransform:   transformStdInstall("mcpServers"),
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	Cline: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return nil
			}
			return vsCodeExtPaths(p, "saoudrizwan.claude-dev", "cline_mcp_settings.json")
		},
		detect: func(p Platform, d Detector) bool {
			for _, path := range vsCodeExtPaths(p, "saoudrizwan.claude-dev", "cline_mcp_settings.json") {
				if d.DirExists(filepath.Dir(path)) {
					return true
				}
			}
			return false
		},
		format:             formatJSON,
		installTransform:   transformStdInstall("mcpServers"),
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	RooCode: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), ".roo", "mcp.json")}
			}
			return vsCodeExtPaths(p, "rooveterinaryinc.roo-cline", "mcp_settings.json")
		},
		detect: func(p Platform, d Detector) bool {
			for _, path := range vsCodeExtPaths(p, "rooveterinaryinc.roo-cline", "mcp_settings.json") {
				if d.DirExists(filepath.Dir(path)) {
					return true
				}
			}
			return false
		},
		format:             formatJSON,
		installTransform:   transformStdInstall("mcpServers"),
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	Gemini: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), ".gemini", "settings.json")}
			}
			return []string{filepath.Join(p.HomeDir, ".gemini", "settings.json")}
		},
		detect: func(_ Platform, d Detector) bool {
			return d.CommandExists("gemini")
		},
		format:             formatJSON,
		installTransform:   transformStdInstall("mcpServers"),
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	AmazonQ: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), ".amazonq", "default.json")}
			}
			return []string{filepath.Join(p.HomeDir, ".aws", "amazonq", "default.json")}
		},
		detect: func(p Platform, d Detector) bool {
			return d.DirExists(filepath.Join(p.HomeDir, ".aws", "amazonq")) || d.CommandExists("q")
		},
		format:             formatJSON,
		installTransform:   transformStdInstall("mcpServers"),
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	Codex: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), ".codex", "config.toml")}
			}
			return []string{filepath.Join(p.HomeDir, ".codex", "config.toml")}
		},
		detect: func(_ Platform, d Detector) bool {
			return d.CommandExists("codex")
		},
		format:             formatTOML,
		installTransform:   transformCodexInstall,
		uninstallTransform: transformCodexUninstall,
	},

	Goose: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), ".goose", "config.yaml")}
			}
			return []string{filepath.Join(p.HomeDir, ".config", "goose", "config.yaml")}
		},
		detect: func(_ Platform, d Detector) bool {
			return d.CommandExists("goose")
		},
		format:             formatYAML,
		installTransform:   transformGooseInstall,
		uninstallTransform: transformGooseUninstall,
	},

	Continue: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), ".continue", "mcpServers")}
			}
			return []string{filepath.Join(p.HomeDir, ".continue", "mcpServers")}
		},
		detect: func(p Platform, d Detector) bool {
			return d.DirExists(filepath.Join(p.HomeDir, ".continue"))
		},
		format:             formatContinueDir,
		installTransform:   transformContinueInstall,
		uninstallTransform: nil, // handled directly in doUninstall
	},
}

// --- helpers ---

func vsCodeExtPaths(p Platform, extensionID, filename string) []string {
	rel := filepath.Join("Code", "User", "globalStorage", extensionID, "settings", filename)
	switch p.GOOS {
	case "darwin":
		return []string{filepath.Join(p.HomeDir, "Library", "Application Support", rel)}
	case "linux":
		return []string{filepath.Join(p.HomeDir, ".config", rel)}
	case "windows":
		if p.AppData != "" {
			return []string{filepath.Join(p.AppData, rel)}
		}
	}
	return nil
}

func projectDir(p Platform, o *options) string {
	if o.projectDir != "" {
		return o.projectDir
	}
	return p.WorkingDir
}
