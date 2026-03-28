package addmcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// --- JSON file I/O ---

func readJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}
	cleaned := stripJSONC(data)
	var m map[string]any
	if err := json.Unmarshal(cleaned, &m); err != nil {
		return nil, fmt.Errorf("%s: malformed JSON: %w", path, err)
	}
	return m, nil
}

func writeJSONFile(path string, data map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0644)
}

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

func jsonStandardInstall(key string) func(string, Server) error {
	return func(path string, s Server) error {
		m, err := readJSONFile(path)
		if err != nil {
			return err
		}
		servers, _ := m[key].(map[string]any)
		if servers == nil {
			servers = make(map[string]any)
		}
		servers[s.Name] = serverConfig(s)
		m[key] = servers
		return writeJSONFile(path, m)
	}
}

func jsonStandardUninstall(key string) func(string, string) error {
	return func(path string, name string) error {
		m, err := readJSONFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		servers, _ := m[key].(map[string]any)
		if servers == nil {
			return nil
		}
		delete(servers, name)
		m[key] = servers
		return writeJSONFile(path, m)
	}
}

// --- VS Code: {"servers": {"name": {"type": "stdio", ...}}} ---

func vscodeInstall(path string, s Server) error {
	m, err := readJSONFile(path)
	if err != nil {
		return err
	}
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
	return writeJSONFile(path, m)
}

// --- Zed: {"context_servers": {"name": {"source": "custom", ...}}} ---

func zedInstall(path string, s Server) error {
	m, err := readJSONFile(path)
	if err != nil {
		return err
	}
	servers, _ := m["context_servers"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
	}
	cfg := serverConfig(s)
	cfg["source"] = "custom"
	servers[s.Name] = cfg
	m["context_servers"] = servers
	return writeJSONFile(path, m)
}

// --- Amazon Q: {"mcpServers": [{"name": "...", "transport": "stdio", ...}]} ---

func amazonQInstall(path string, s Server) error {
	m, err := readJSONFile(path)
	if err != nil {
		return err
	}

	// Collect existing entries, removing any with the same name.
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
	return writeJSONFile(path, m)
}

func amazonQUninstall(path string, name string) error {
	m, err := readJSONFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	existing, ok := m["mcpServers"].([]any)
	if !ok {
		return nil
	}
	var filtered []any
	for _, entry := range existing {
		if e, ok := entry.(map[string]any); ok && e["name"] == name {
			continue
		}
		filtered = append(filtered, entry)
	}
	m["mcpServers"] = filtered
	return writeJSONFile(path, m)
}

// --- Continue: writes individual files into mcpServers/ directory ---

func continueInstall(dir string, s Server) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	cfg := map[string]any{
		"mcpServers": map[string]any{
			s.Name: serverConfig(s),
		},
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, s.Name+".json"), append(out, '\n'), 0644)
}

func continueUninstall(dir string, name string) error {
	err := os.Remove(filepath.Join(dir, name+".json"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
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
