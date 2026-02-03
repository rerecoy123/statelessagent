# SAME — Stateless Agent Memory Engine

SAME gives AI coding agents persistent memory across sessions. It indexes any folder of markdown files into a local SQLite database with vector embeddings, then surfaces relevant context automatically via hooks and an MCP server.

Works with Obsidian, Logseq, Foam, Dendron, or any directory of `.md` files. Designed for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) but usable with any MCP client (Cursor, Windsurf, etc.).

## Quick Start

```bash
# Install (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/sgx-labs/statelessagent/main/install.sh | bash

# Or build from source
git clone https://github.com/sgx-labs/statelessagent.git
cd statelessagent
make install

# Point at your knowledge base
export VAULT_PATH=~/Documents/my-notes

# Start Ollama (required for embeddings)
ollama pull nomic-embed-text

# Index your notes
same reindex

# Search
same search "how does authentication work"

# Check system health
same doctor
```

## How It Works

1. **Indexing** — Walks your markdown directory, chunks long notes by H2 headings, and generates 768-dim embeddings via [Ollama](https://ollama.ai) (`nomic-embed-text`). Stores everything in a local SQLite database with [sqlite-vec](https://github.com/asg017/sqlite-vec) for vector search.

2. **Search** — KNN vector search with composite scoring that blends semantic similarity, recency (exponential decay by content type), and confidence (based on content type, access frequency, and maintenance signals).

3. **Hooks** — Claude Code hooks that fire automatically:
   - **Context Surfacing** (`UserPromptSubmit`) — Embeds your prompt, searches the vault, injects relevant notes as context
   - **Decision Extraction** (`Stop`) — Extracts decisions from the conversation transcript and appends them to a decision log
   - **Handoff Generation** (`Stop`/`PreCompact`) — Generates session handoff notes for cross-session continuity
   - **Staleness Check** (`SessionStart`) — Surfaces notes that haven't been reviewed in a while

4. **MCP Server** — Exposes 6 tools over stdio for direct use by any MCP client.

## CLI Reference

```
same reindex [--force]       Index/re-index markdown files
same search <query>          Search from the command line
same related <note-path>     Find semantically similar notes
same stats                   Show index statistics
same doctor                  System health check
same mcp                     Start MCP stdio server
same watch                   Watch for changes and auto-reindex
same hook <name>             Run a hook handler
same eval-export             JSON output for eval harness
same bench                   Performance benchmarks
same budget                  Context utilization report
same vault list|add|remove   Manage vault registrations
same plugin list             List hook plugins
same version [--check]       Print version / check for updates
```

## MCP Server

SAME exposes 6 tools via MCP (Model Context Protocol):

| Tool | Description |
|------|-------------|
| `search_notes` | Semantic search across all indexed notes |
| `search_notes_filtered` | Search with domain/workstream/tag filters |
| `get_note` | Read full content of a note by path |
| `find_similar_notes` | Find notes similar to a given note |
| `reindex` | Re-index the vault (with cooldown) |
| `index_stats` | Index statistics (note count, chunks, etc.) |

### Claude Code Configuration

Add to your project's `.mcp.json`:

```json
{
  "mcpServers": {
    "same": {
      "command": "same",
      "args": ["mcp"],
      "env": {
        "VAULT_PATH": "/path/to/your/notes"
      }
    }
  }
}
```

### Claude Code Hooks

Add to `.claude/settings.json`:

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": "",
        "hooks": [{ "type": "command", "command": "same hook context-surfacing" }]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          { "type": "command", "command": "same hook decision-extractor" },
          { "type": "command", "command": "same hook handoff-generator" }
        ]
      }
    ],
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          { "type": "command", "command": "same version --check" },
          { "type": "command", "command": "same hook staleness-check" }
        ]
      }
    ]
  }
}
```

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `VAULT_PATH` | auto-detect | Path to your markdown knowledge base |
| `OLLAMA_URL` | `http://localhost:11434` | Ollama API endpoint (must be localhost) |
| `SAME_DATA_DIR` | `<vault>/.same/data` | Where to store the SQLite database |
| `SAME_HANDOFF_DIR` | `sessions` | Directory for handoff notes (relative to vault) |
| `SAME_DECISION_LOG` | `decisions.md` | Path for the decision log (relative to vault) |
| `SAME_SKIP_DIRS` | *(none)* | Extra directories to skip (comma-separated) |

SAME also supports a vault registry for managing multiple knowledge bases:

```bash
same vault add work ~/Documents/work-notes
same vault add personal ~/Documents/personal-notes
same vault default work
same --vault personal search "weekend plans"
```

## Building from Source

Requires Go 1.23+ and CGO (for SQLite):

```bash
git clone https://github.com/sgx-labs/statelessagent.git
cd statelessagent
make build        # Build for current platform
make test         # Run tests
make install      # Install to $GOPATH/bin
make cross-all    # Build for macOS (arm64/amd64), Linux, Windows
```

Cross-compilation for Linux/Windows from macOS requires [zig](https://ziglang.org/) as a C cross-compiler.

## Security

- Ollama URL is validated to be localhost-only (no remote embedding endpoints)
- `_PRIVATE/` directories are excluded from indexing and context surfacing at multiple layers
- Snippet content is scanned for prompt injection patterns before injection
- Path traversal attacks are blocked in the MCP `get_note` tool
- All data stays local — no external API calls except Ollama on localhost

## License

Business Source License 1.1 — free for personal, educational, and non-commercial use. Commercial production use requires a paid license. See [LICENSE](LICENSE) for full terms.

On 2030-02-02, the code automatically relicenses to Apache 2.0.
