package addmcp

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// --- Goose: YAML with "extensions" key, "cmd"/"envs" fields ---

func gooseInstall(path string, s Server) error {
	m, err := readYAMLFile(path)
	if err != nil {
		return err
	}

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
	return writeYAMLFile(path, m)
}

func gooseUninstall(path string, name string) error {
	m, err := readYAMLFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	extensions, _ := m["extensions"].(map[string]any)
	if extensions == nil {
		return nil
	}
	delete(extensions, name)
	m["extensions"] = extensions
	return writeYAMLFile(path, m)
}

// --- YAML file I/O ---

func readYAMLFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s: malformed YAML: %w", path, err)
	}
	if m == nil {
		m = make(map[string]any)
	}
	return m, nil
}

func writeYAMLFile(path string, data map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}
