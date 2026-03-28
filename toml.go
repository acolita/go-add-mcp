package addmcp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// --- Codex: TOML with "mcp_servers" key ---

func codexInstall(path string, s Server) error {
	m, err := readTOMLFile(path)
	if err != nil {
		return err
	}

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
	return writeTOMLFile(path, m)
}

func codexUninstall(path string, name string) error {
	m, err := readTOMLFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	servers, _ := m["mcp_servers"].(map[string]any)
	if servers == nil {
		return nil
	}
	delete(servers, name)
	m["mcp_servers"] = servers
	return writeTOMLFile(path, m)
}

// --- TOML file I/O ---

func readTOMLFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}
	var m map[string]any
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s: malformed TOML: %w", path, err)
	}
	return m, nil
}

func writeTOMLFile(path string, data map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	out, err := toml.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}
