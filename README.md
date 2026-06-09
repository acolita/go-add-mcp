# go-add-mcp

Install and remove MCP (Model Context Protocol) server configurations across
all major AI agent clients from Go.

Inspired by [neondatabase/add-mcp](https://github.com/neondatabase/add-mcp).

## Supported agents

Claude Code, Claude Desktop, Cursor, Windsurf, VS Code, Zed, JetBrains (Junie),
Cline (VS Code extension), Cline CLI, Roo Code, Gemini CLI, Amazon Q, Codex,
Goose, Continue, Antigravity, OpenCode, MCPorter, GitHub Copilot CLI.

Each agent is handled in its native format (JSON, TOML, YAML, or a directory
of JSON files) with the correct config key (`mcpServers`, `servers`,
`context_servers`, `mcp_servers`, …) and per-agent transport quirks (e.g.
Windsurf/Antigravity use `serverUrl`, Cline adds `disabled`/`type`).

Claude Desktop only accepts stdio (local) servers via its config file —
installing a remote (`URL`) server to it returns an error, since remotes must
be added through the app.

## Install

```sh
go get github.com/acolita/go-add-mcp
```

## Usage

```go
package main

import (
    "fmt"

    addmcp "github.com/acolita/go-add-mcp"
)

func main() {
    server := addmcp.Server{
        Name:    "my-server",
        Command: "npx",
        Args:    []string{"-y", "@example/mcp-server"},
        Env:     map[string]string{"API_KEY": "xxx"},
    }

    results := addmcp.Install(server, []addmcp.Agent{
        addmcp.ClaudeCode,
        addmcp.Cursor,
        addmcp.VSCode,
    })

    for _, r := range results {
        if r.OK() {
            fmt.Printf("%s: %s\n", r.Agent, r.Path)
        } else {
            fmt.Printf("%s: %v\n", r.Agent, r.Err)
        }
    }
}
```

### HTTP/SSE transport

Set `URL` (and optional `Headers`) instead of `Command`/`Args`:

```go
server := addmcp.Server{
    Name:    "remote",
    URL:     "https://mcp.example.com/sse",
    Headers: map[string]string{"Authorization": "Bearer xxx"},
}
```

### Scope

Default is user-global. Use `WithScope(Project)` for per-project configs:

```go
addmcp.Install(server, agents,
    addmcp.WithScope(addmcp.Project),
    addmcp.WithProjectDir("/path/to/project"),
)
```

### Other operations

```go
addmcp.Uninstall("my-server", agents)          // remove
addmcp.Detect()                                // agents present on the system
addmcp.Resolve(agents)                         // dry-run: paths that would be written
addmcp.Agents()                                // all supported agents
```

Errors are per-agent (on `Result.Err`) and non-fatal — one broken config
doesn't stop the rest.

## Design

Pure config transformation is separated from filesystem IO:

- `transform.go` — format-agnostic map transformations (100% test coverage)
- `io.go` — read/parse/write per format
- `registry.go` — per-agent paths, detection, format, and transform wiring
- `platform.go` — injected OS/env for testability

Golden-file tests cover all 19 agents, for both stdio and remote transports.

## License

See repository for license details.
