# @sgx-labs/same

NPM wrapper for [SAME (Stateless Agent Memory Engine)](https://github.com/sgx-labs/statelessagent) — persistent memory for AI coding agents.

This package downloads the prebuilt SAME binary for your platform at install time. No Go toolchain required.

## MCP Configuration

Add to your MCP client config (Claude Desktop, Cursor, etc.):

```json
{
  "mcpServers": {
    "same": {
      "command": "npx",
      "args": ["-y", "@sgx-labs/same", "mcp", "--vault", "/path/to/your/notes"]
    }
  }
}
```

## CLI Usage

```bash
# Initialize SAME in your project
npx @sgx-labs/same init

# Search your notes
npx @sgx-labs/same search "authentication approach"

# Ask a question with cited answers
npx @sgx-labs/same ask "what did we decide about the database schema?"

# Check version
npx @sgx-labs/same version
```

## Platform Support

| Platform | Architecture | Status |
|----------|-------------|--------|
| macOS | Apple Silicon (arm64) | Supported |
| macOS | Intel (x64) | Via Rosetta |
| Linux | x64 | Supported |
| Windows | x64 | Supported |

## Links

- [Main repository](https://github.com/sgx-labs/statelessagent)
- [Documentation](https://github.com/sgx-labs/statelessagent#readme)
- [Releases](https://github.com/sgx-labs/statelessagent/releases)
