// Package addmcp installs and removes MCP server configurations
// across all major AI agent clients (Claude, Cursor, VS Code, etc.).
//
// Inspired by https://github.com/neondatabase/add-mcp
package addmcp

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Agent identifies a supported MCP client application.
type Agent string

const (
	ClaudeCode    Agent = "claude-code"
	ClaudeDesktop Agent = "claude-desktop"
	Cursor        Agent = "cursor"
	Windsurf      Agent = "windsurf"
	VSCode        Agent = "vscode"
	Zed           Agent = "zed"
	JetBrains     Agent = "jetbrains"
	Cline         Agent = "cline"
	RooCode       Agent = "roo-code"
	Gemini        Agent = "gemini"
	AmazonQ       Agent = "amazon-q"
	Codex         Agent = "codex"
	Goose         Agent = "goose"
	Continue      Agent = "continue"
)

// Server describes an MCP server to install into agent configs.
type Server struct {
	Name    string            // Server name (key in config objects, filename for Continue)
	Command string            // Executable path or name (stdio transport)
	Args    []string          // Command arguments (stdio transport)
	Env     map[string]string // Environment variables
	URL     string            // Server endpoint (HTTP/SSE transport; if set, Command/Args are ignored)
	Headers map[string]string // HTTP headers (HTTP/SSE transport)
}

// IsHTTP returns true if the server uses HTTP/SSE transport.
func (s Server) IsHTTP() bool { return s.URL != "" }

// Scope controls whether to install globally or per-project.
type Scope int

const (
	Global  Scope = iota // User-wide config (default)
	Project              // Project-level config
)

// Result reports the outcome for one agent.
type Result struct {
	Agent Agent
	Path  string // Config file that was written (or would be)
	Err   error
}

// OK returns true if the operation succeeded.
func (r Result) OK() bool { return r.Err == nil }

type options struct {
	scope      Scope
	projectDir string
}

// Option configures Install/Uninstall behavior.
type Option func(*options)

// WithScope sets the installation scope (default: Global).
func WithScope(s Scope) Option {
	return func(o *options) { o.scope = s }
}

// WithProjectDir sets the project directory for Project-scoped installs.
// If empty, the current working directory is used.
func WithProjectDir(dir string) Option {
	return func(o *options) { o.projectDir = dir }
}

// Install adds or updates the MCP server configuration for the given agents.
// Each agent gets its own Result; errors are per-agent, not fatal.
func Install(server Server, agents []Agent, opts ...Option) []Result {
	o := applyOpts(opts)
	results := make([]Result, 0, len(agents))
	for _, agent := range agents {
		def, ok := registry[agent]
		if !ok {
			results = append(results, Result{Agent: agent, Err: fmt.Errorf("unknown agent: %s", agent)})
			continue
		}
		paths := def.paths(o)
		if len(paths) == 0 {
			results = append(results, Result{
				Agent: agent,
				Err:   fmt.Errorf("no config path for %s on %s (scope: %v)", agent, runtime.GOOS, scopeName(o.scope)),
			})
			continue
		}
		for _, path := range paths {
			err := def.install(path, server)
			results = append(results, Result{Agent: agent, Path: path, Err: err})
		}
	}
	return results
}

// Uninstall removes the named MCP server from the given agents' configs.
func Uninstall(serverName string, agents []Agent, opts ...Option) []Result {
	o := applyOpts(opts)
	results := make([]Result, 0, len(agents))
	for _, agent := range agents {
		def, ok := registry[agent]
		if !ok {
			results = append(results, Result{Agent: agent, Err: fmt.Errorf("unknown agent: %s", agent)})
			continue
		}
		paths := def.paths(o)
		if len(paths) == 0 {
			results = append(results, Result{
				Agent: agent,
				Err:   fmt.Errorf("no config path for %s on %s (scope: %v)", agent, runtime.GOOS, scopeName(o.scope)),
			})
			continue
		}
		for _, path := range paths {
			err := def.uninstall(path, serverName)
			results = append(results, Result{Agent: agent, Path: path, Err: err})
		}
	}
	return results
}

// Detect returns agents that appear to be installed on this system.
// Detection checks for config directories and CLI commands in PATH.
func Detect() []Agent {
	var found []Agent
	for _, a := range allAgents {
		if def, ok := registry[a]; ok && def.detect() {
			found = append(found, a)
		}
	}
	return found
}

// Agents returns all supported agent identifiers in a stable order.
func Agents() []Agent {
	out := make([]Agent, len(allAgents))
	copy(out, allAgents)
	return out
}

// allAgents defines the canonical order.
var allAgents = []Agent{
	ClaudeCode, ClaudeDesktop, Cursor, Windsurf, VSCode,
	Zed, JetBrains, Cline, RooCode, Gemini,
	AmazonQ, Codex, Goose, Continue,
}

func applyOpts(opts []Option) *options {
	o := &options{scope: Global}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

func scopeName(s Scope) string {
	if s == Project {
		return "project"
	}
	return "global"
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
