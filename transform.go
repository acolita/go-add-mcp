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
	servers[s.Name] = cfg
	m["context_servers"] = servers
	return m
}

// --- Amazon Q: {"mcpServers": [{"name": "...", "transport": "stdio", ...}]} ---

func transformAmazonQInstall(m map[string]any, s Server) map[string]any {
	var servers []any
	if existing, ok := m["mcpServers"].([]any); ok {
		for _, entry := range existing {
			if e, ok := entry.(map[string]any); ok && e["name"] == s.Name {
				continue
			}
			servers = append(servers, entry)
		}
	}

	entry := map[string]any{"name": s.Name}
	if s.IsHTTP() {
		entry["transport"] = "http"
		entry["url"] = s.URL
		if len(s.Headers) > 0 {
			entry["headers"] = s.Headers
		}
	} else {
		entry["transport"] = "stdio"
		entry["command"] = s.Command
		if len(s.Args) > 0 {
			entry["arguments"] = s.Args // Amazon Q uses "arguments", not "args"
		}
	}
	if len(s.Env) > 0 {
		entry["env"] = s.Env
	}

	servers = append(servers, entry)
	m["mcpServers"] = servers
	return m
}

func transformAmazonQUninstall(m map[string]any, name string) map[string]any {
	existing, ok := m["mcpServers"].([]any)
	if !ok {
		return m
	}
	var filtered []any
	for _, entry := range existing {
		if e, ok := entry.(map[string]any); ok && e["name"] == name {
			continue
		}
		filtered = append(filtered, entry)
	}
	m["mcpServers"] = filtered
	return m
}

// --- Goose: YAML {"extensions": {"name": {"cmd": "...", "envs": {...}}}} ---

func transformGooseInstall(m map[string]any, s Server) map[string]any {
	extensions, _ := m["extensions"].(map[string]any)
	if extensions == nil {
		extensions = make(map[string]any)
	}
	entry := map[string]any{
		"name":    s.Name,
		"enabled": true,
		"type":    "stdio",
	}
	if s.IsHTTP() {
		entry["type"] = "sse"
		entry["uri"] = s.URL
		if len(s.Headers) > 0 {
			entry["headers"] = s.Headers
		}
	} else {
		entry["cmd"] = s.Command
		if len(s.Args) > 0 {
			entry["args"] = s.Args
		}
	}
	if len(s.Env) > 0 {
		entry["envs"] = s.Env
	}
	extensions[s.Name] = entry
	m["extensions"] = extensions
	return m
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
