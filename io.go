package addmcp

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// doInstall dispatches to the format-specific read-transform-write shell.
func (def agentDef) doInstall(fsys FS, path string, s Server) error {
	switch def.format {
	case formatJSON:
		return jsonRTW(fsys, path, func(m map[string]any) map[string]any {
			return def.installTransform(m, s)
		})
	case formatYAML:
		return yamlRTW(fsys, path, func(m map[string]any) map[string]any {
			return def.installTransform(m, s)
		})
	case formatTOML:
		return tomlRTW(fsys, path, func(m map[string]any) map[string]any {
			return def.installTransform(m, s)
		})
	case formatContinueDir:
		content := def.installTransform(nil, s)
		out, err := json.MarshalIndent(content, "", "  ")
		if err != nil {
			return err
		}
		if err := fsys.MkdirAll(path, 0755); err != nil {
			return err
		}
		return fsys.WriteFile(filepath.Join(path, s.Name+".json"), append(out, '\n'), 0644)
	}
	return fmt.Errorf("unsupported format")
}

// doUninstall dispatches to the format-specific read-transform-write shell.
// Returns nil if the config file doesn't exist (nothing to uninstall from).
func (def agentDef) doUninstall(fsys FS, path string, name string) error {
	if def.format == formatContinueDir {
		err := fsys.Remove(filepath.Join(path, name+".json"))
		if isNotExist(err) {
			return nil
		}
		return err
	}

	// Don't create a config file just to uninstall from it
	if _, err := fsys.Stat(path); isNotExist(err) {
		return nil
	}

	switch def.format {
	case formatJSON:
		return jsonRTW(fsys, path, func(m map[string]any) map[string]any {
			return def.uninstallTransform(m, name)
		})
	case formatYAML:
		return yamlRTW(fsys, path, func(m map[string]any) map[string]any {
			return def.uninstallTransform(m, name)
		})
	case formatTOML:
		return tomlRTW(fsys, path, func(m map[string]any) map[string]any {
			return def.uninstallTransform(m, name)
		})
	}
	return fmt.Errorf("unsupported format")
}

// --- Read-Transform-Write shells ---
// Each reads a config file, applies a pure transform, and writes it back.
// If the file doesn't exist, starts with an empty map (creating it).

func jsonRTW(fsys FS, path string, transform func(map[string]any) map[string]any) error {
	var m map[string]any
	data, err := fsys.ReadFile(path)
	if err != nil {
		if !isNotExist(err) {
			return err
		}
		m = make(map[string]any)
	} else {
		cleaned := stripJSONC(data)
		if err := json.Unmarshal(cleaned, &m); err != nil {
			return fmt.Errorf("%s: malformed JSON: %w", path, err)
		}
	}

	result := transform(m)

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := fsys.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return fsys.WriteFile(path, append(out, '\n'), 0644)
}

func yamlRTW(fsys FS, path string, transform func(map[string]any) map[string]any) error {
	var m map[string]any
	data, err := fsys.ReadFile(path)
	if err != nil {
		if !isNotExist(err) {
			return err
		}
		m = make(map[string]any)
	} else {
		if err := yaml.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("%s: malformed YAML: %w", path, err)
		}
		if m == nil {
			m = make(map[string]any)
		}
	}

	result := transform(m)

	out, err := yaml.Marshal(result)
	if err != nil {
		return err
	}
	if err := fsys.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return fsys.WriteFile(path, out, 0644)
}

func tomlRTW(fsys FS, path string, transform func(map[string]any) map[string]any) error {
	var m map[string]any
	data, err := fsys.ReadFile(path)
	if err != nil {
		if !isNotExist(err) {
			return err
		}
		m = make(map[string]any)
	} else {
		if err := toml.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("%s: malformed TOML: %w", path, err)
		}
	}

	result := transform(m)

	out, err := toml.Marshal(result)
	if err != nil {
		return err
	}
	if err := fsys.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return fsys.WriteFile(path, out, 0644)
}
