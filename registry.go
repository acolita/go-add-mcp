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
	stdioOnly          bool                                        // reject HTTP/SSE installs (e.g. Claude Desktop)
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
		stdioOnly:          true, // remote servers must be added through the Claude Desktop app
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
		installTransform:   transformAntigravityInstall, // remote uses `serverUrl`, like Antigravity
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	VSCode: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), ".vscode", "mcp.json")}
			}
			if dir := vsCodeUserDir(p); dir != "" {
				return []string{filepath.Join(dir, "mcp.json")}
			}
			return nil
		},
		detect: func(p Platform, d Detector) bool {
			if dir := vsCodeUserDir(p); dir != "" && d.DirExists(dir) {
				return true
			}
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
			if dir := zedConfigDir(p); dir != "" {
				return []string{filepath.Join(dir, "settings.json")}
			}
			return nil
		},
		detect: func(p Platform, d Detector) bool {
			if dir := zedConfigDir(p); dir != "" && d.DirExists(dir) {
				return true
			}
			return d.CommandExists("zed")
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
		installTransform:   transformClineInstall,
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	ClineCLI: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return nil
			}
			return []string{filepath.Join(clineCLIHome(p), "data", "settings", "cline_mcp_settings.json")}
		},
		detect: func(p Platform, d Detector) bool {
			return d.DirExists(clineCLIHome(p))
		},
		format:             formatJSON,
		installTransform:   transformClineInstall,
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
			return []string{filepath.Join(codexHome(p), "config.toml")}
		},
		detect: func(p Platform, d Detector) bool {
			return d.DirExists(codexHome(p)) || d.CommandExists("codex")
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
			return []string{gooseGlobalPath(p)}
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

	Antigravity: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return nil
			}
			return []string{filepath.Join(p.HomeDir, ".gemini", "antigravity", "mcp_config.json")}
		},
		detect: func(p Platform, d Detector) bool {
			return d.DirExists(filepath.Join(p.HomeDir, ".gemini"))
		},
		format:             formatJSON,
		installTransform:   transformAntigravityInstall,
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	OpenCode: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), "opencode.json")}
			}
			return []string{filepath.Join(xdgConfigHome(p), "opencode", "opencode.json")}
		},
		detect: func(p Platform, d Detector) bool {
			return d.DirExists(filepath.Join(xdgConfigHome(p), "opencode"))
		},
		format:             formatJSON,
		installTransform:   transformOpenCodeInstall,
		uninstallTransform: transformStdUninstall("mcp"),
	},

	MCPorter: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return []string{filepath.Join(projectDir(p, o), "config", "mcporter.json")}
			}
			return []string{filepath.Join(p.HomeDir, ".mcporter", "mcporter.json")}
		},
		detect: func(p Platform, d Detector) bool {
			return d.DirExists(filepath.Join(p.HomeDir, ".mcporter"))
		},
		format:             formatJSON,
		installTransform:   transformStdInstall("mcpServers"),
		uninstallTransform: transformStdUninstall("mcpServers"),
	},

	GitHubCopilotCLI: {
		paths: func(p Platform, o *options) []string {
			if o.scope == Project {
				return nil
			}
			return []string{filepath.Join(p.HomeDir, ".copilot", "mcp-config.json")}
		},
		detect: func(p Platform, d Detector) bool {
			return d.DirExists(filepath.Join(p.HomeDir, ".copilot")) || d.CommandExists("copilot")
		},
		format:             formatJSON,
		installTransform:   transformCopilotInstall,
		uninstallTransform: transformStdUninstall("mcpServers"),
	},
}

// --- helpers ---

func vsCodeExtPaths(p Platform, extensionID, filename string) []string {
	dir := vsCodeUserDir(p)
	if dir == "" {
		return nil
	}
	return []string{filepath.Join(dir, "globalStorage", extensionID, "settings", filename)}
}

// vsCodeUserDir is the platform-specific "Code/User" directory containing
// mcp.json (global) and globalStorage subtree (per-extension configs).
func vsCodeUserDir(p Platform) string {
	switch p.GOOS {
	case "darwin":
		return filepath.Join(p.HomeDir, "Library", "Application Support", "Code", "User")
	case "linux":
		return filepath.Join(xdgConfigHome(p), "Code", "User")
	case "windows":
		if p.AppData != "" {
			return filepath.Join(p.AppData, "Code", "User")
		}
	}
	return ""
}

// zedConfigDir is the directory containing Zed's settings.json (global scope).
// Upstream packages darwin and windows under "Zed"; linux uses lowercase "zed".
func zedConfigDir(p Platform) string {
	switch p.GOOS {
	case "darwin":
		return filepath.Join(p.HomeDir, "Library", "Application Support", "Zed")
	case "linux":
		return filepath.Join(xdgConfigHome(p), "zed")
	case "windows":
		if p.AppData != "" {
			return filepath.Join(p.AppData, "Zed")
		}
	}
	return ""
}

func codexHome(p Platform) string {
	if p.CodexHome != "" {
		return p.CodexHome
	}
	return filepath.Join(p.HomeDir, ".codex")
}

// clineCLIHome returns the Cline CLI base directory ($CLINE_DIR or ~/.cline).
func clineCLIHome(p Platform) string {
	if p.ClineDir != "" {
		return p.ClineDir
	}
	return filepath.Join(p.HomeDir, ".cline")
}

func gooseGlobalPath(p Platform) string {
	switch p.GOOS {
	case "windows":
		if p.AppData != "" {
			return filepath.Join(p.AppData, "Block", "goose", "config", "config.yaml")
		}
		return filepath.Join(p.HomeDir, ".config", "goose", "config.yaml")
	default:
		return filepath.Join(xdgConfigHome(p), "goose", "config.yaml")
	}
}

// xdgConfigHome returns $XDG_CONFIG_HOME on Linux/macOS fallback, or ~/.config.
func xdgConfigHome(p Platform) string {
	if p.XDGConfigHome != "" {
		return p.XDGConfigHome
	}
	return filepath.Join(p.HomeDir, ".config")
}

func projectDir(p Platform, o *options) string {
	if o.projectDir != "" {
		return o.projectDir
	}
	return p.WorkingDir
}
