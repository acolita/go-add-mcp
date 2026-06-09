package addmcp

import "strings"

// This file contains ALL pure functions: no filesystem, no env, no side effects.
// Every function is deterministic and testable with just inputs/outputs.

// --- Server config builders ---

func stdioConfig(s Server) map[string]any {
	m := map[string]any{"command": s.Command}
	if len(s.Args) > 0 {
		m["args"] = s.Args
	}
	if len(s.Env) > 0 {
		m["env"] = s.Env
	}
	return m
}

func httpConfig(s Server) map[string]any {
	m := map[string]any{"url": s.URL}
	if len(s.Headers) > 0 {
		m["headers"] = s.Headers
	}
	return m
}

func serverConfig(s Server) map[string]any {
	if s.IsHTTP() {
		return httpConfig(s)
	}
	return stdioConfig(s)
}

// --- Standard format: {"key": {"name": {...}}} ---
// Used by: Claude Desktop, Claude Code, Cursor, Windsurf, JetBrains, Cline, Roo Code, Gemini

func transformStdInstall(key string) func(map[string]any, Server) map[string]any {
	return func(m map[string]any, s Server) map[string]any {
		servers, _ := m[key].(map[string]any)
		if servers == nil {
			servers = make(map[string]any)
		}
		servers[s.Name] = serverConfig(s)
		m[key] = servers
		return m
	}
}

func transformStdUninstall(key string) func(map[string]any, string) map[string]any {
	return func(m map[string]any, name string) map[string]any {
		servers, _ := m[key].(map[string]any)
		if servers == nil {
			return m
		}
		delete(servers, name)
		m[key] = servers
		return m
	}
}

// --- VS Code: {"servers": {"name": {"type": "stdio", ...}}} ---

func transformVSCodeInstall(m map[string]any, s Server) map[string]any {
	servers, _ := m["servers"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
	}
	cfg := serverConfig(s)
	if s.IsHTTP() {
		cfg["type"] = "http"
	} else {
		cfg["type"] = "stdio"
	}
	servers[s.Name] = cfg
	m["servers"] = servers
	return m
}

// --- Zed: {"context_servers": {"name": {"source": "custom", ...}}} ---

func transformZedInstall(m map[string]any, s Server) map[string]any {
	servers, _ := m["context_servers"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
	}
	cfg := serverConfig(s)
	cfg["source"] = "custom"
	if s.IsHTTP() {
		cfg["type"] = remoteType(s)
	}
	servers[s.Name] = cfg
	m["context_servers"] = servers
	return m
}

// remoteType returns the transport subtype for a remote server.
// Defaults to "http"; callers passing Transport="sse" get "sse".
func remoteType(s Server) string {
	if s.IsSSE() {
		return "sse"
	}
	return "http"
}

// --- Amazon Q: standard mcpServers object format ---
// Verified from aws/amazon-q-developer-cli schemas/agent-v1.json:
// mcpServers is an object (not array), uses "args" (not "arguments"),
// "type" field for transport defaults to "stdio".

// --- Goose: YAML {"extensions": {"name": {"cmd": "...", "envs": {...}}}} ---

func transformGooseInstall(m map[string]any, s Server) map[string]any {
	extensions, _ := m["extensions"].(map[string]any)
	if extensions == nil {
		extensions = make(map[string]any)
	}
	entry := map[string]any{
		"name":        s.Name,
		"description": "",
		"enabled":     true,
		"timeout":     300,
	}
	if s.IsHTTP() {
		gooseType := "streamable_http"
		if s.IsSSE() {
			gooseType = "sse"
		}
		entry["type"] = gooseType
		entry["uri"] = s.URL
		entry["headers"] = stringMapOrEmpty(s.Headers)
	} else {
		entry["type"] = "stdio"
		entry["cmd"] = s.Command
		entry["args"] = stringSliceOrEmpty(s.Args)
		entry["envs"] = stringMapOrEmpty(s.Env)
	}
	extensions[s.Name] = entry
	m["extensions"] = extensions
	return m
}

func stringMapOrEmpty(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

func stringSliceOrEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func transformGooseUninstall(m map[string]any, name string) map[string]any {
	extensions, _ := m["extensions"].(map[string]any)
	if extensions == nil {
		return m
	}
	delete(extensions, name)
	m["extensions"] = extensions
	return m
}

// --- Codex: TOML {"mcp_servers": {"name": {"command": "..."}}} ---

func transformCodexInstall(m map[string]any, s Server) map[string]any {
	servers, _ := m["mcp_servers"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
	}
	entry := make(map[string]any)
	if s.IsHTTP() {
		entry["type"] = remoteType(s)
		entry["url"] = s.URL
		if len(s.Headers) > 0 {
			entry["http_headers"] = s.Headers
		}
	} else {
		entry["command"] = s.Command
		if len(s.Args) > 0 {
			entry["args"] = s.Args
		}
	}
	if len(s.Env) > 0 {
		entry["env"] = s.Env
	}
	servers[s.Name] = entry
	m["mcp_servers"] = servers
	return m
}

func transformCodexUninstall(m map[string]any, name string) map[string]any {
	servers, _ := m["mcp_servers"].(map[string]any)
	if servers == nil {
		return m
	}
	delete(servers, name)
	m["mcp_servers"] = servers
	return m
}

// --- Continue: builds the individual file content ---

func transformContinueInstall(_ map[string]any, s Server) map[string]any {
	return map[string]any{
		"mcpServers": map[string]any{
			s.Name: serverConfig(s),
		},
	}
}

// --- Antigravity: {"mcpServers": {"name": {"serverUrl": "...", ...}}} ---
// Remote uses `serverUrl` (not `url`). Stdio uses standard command/args/env.

func transformAntigravityInstall(m map[string]any, s Server) map[string]any {
	servers, _ := m["mcpServers"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
	}
	var cfg map[string]any
	if s.IsHTTP() {
		cfg = map[string]any{"serverUrl": s.URL}
		if len(s.Headers) > 0 {
			cfg["headers"] = s.Headers
		}
	} else {
		cfg = map[string]any{"command": s.Command}
		if len(s.Args) > 0 {
			cfg["args"] = s.Args
		}
		if len(s.Env) > 0 {
			cfg["env"] = s.Env
		}
	}
	servers[s.Name] = cfg
	m["mcpServers"] = servers
	return m
}

// --- OpenCode: {"mcp": {"name": {"type": "local"|"remote", ...}}} ---

func transformOpenCodeInstall(m map[string]any, s Server) map[string]any {
	servers, _ := m["mcp"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
	}
	var cfg map[string]any
	if s.IsHTTP() {
		cfg = map[string]any{
			"type":    "remote",
			"url":     s.URL,
			"enabled": true,
		}
		if len(s.Headers) > 0 {
			cfg["headers"] = s.Headers
		}
	} else {
		cmd := append([]string{s.Command}, s.Args...)
		cfg = map[string]any{
			"type":        "local",
			"command":     cmd,
			"enabled":     true,
			"environment": stringMapOrEmpty(s.Env),
		}
	}
	servers[s.Name] = cfg
	m["mcp"] = servers
	return m
}

// --- GitHub Copilot CLI (global ~/.copilot/mcp-config.json) ---
// Uses `mcpServers` key. Each entry has a `type` field and a `tools: ["*"]` allowlist.
// Project-scope Copilot shares VS Code's .vscode/mcp.json — use the VSCode agent for that.

func transformCopilotInstall(m map[string]any, s Server) map[string]any {
	servers, _ := m["mcpServers"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
	}
	var cfg map[string]any
	if s.IsHTTP() {
		cfg = map[string]any{
			"type":  remoteType(s),
			"url":   s.URL,
			"tools": []string{"*"},
		}
		if len(s.Headers) > 0 {
			cfg["headers"] = s.Headers
		}
	} else {
		cfg = map[string]any{
			"type":    "stdio",
			"command": s.Command,
			"tools":   []string{"*"},
		}
		if len(s.Args) > 0 {
			cfg["args"] = s.Args
		}
		if len(s.Env) > 0 {
			cfg["env"] = s.Env
		}
	}
	servers[s.Name] = cfg
	m["mcpServers"] = servers
	return m
}

// --- JSONC comment stripping ---

// stripJSONC removes single-line (//) and block (/* */) comments from JSONC,
// leaving string contents intact. The output is valid JSON.
func stripJSONC(data []byte) []byte {
	s := string(data)
	var b strings.Builder
	b.Grow(len(s))

	i := 0
	for i < len(s) {
		// String literal — copy verbatim including escapes.
		if s[i] == '"' {
			b.WriteByte('"')
			i++
			for i < len(s) && s[i] != '"' {
				if s[i] == '\\' && i+1 < len(s) {
					b.WriteByte(s[i])
					i++
				}
				b.WriteByte(s[i])
				i++
			}
			if i < len(s) {
				b.WriteByte('"')
				i++
			}
			continue
		}

		// Line comment.
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '/' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}

		// Block comment.
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			if i+1 < len(s) {
				i += 2
			}
			continue
		}

		b.WriteByte(s[i])
		i++
	}

	return []byte(b.String())
}
