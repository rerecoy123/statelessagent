# Running SAME in Docker

## Build

```bash
docker build -t same .
```

## Usage

Mount your notes directory as `/vault`:

```bash
# Check vault status
docker run --rm -v ~/my-notes:/vault same status

# Search your notes
docker run --rm -v ~/my-notes:/vault same search "authentication"

# Reindex
docker run --rm -v ~/my-notes:/vault same reindex

# Run MCP server (for AI agent integration)
docker run --rm -i -v ~/my-notes:/vault same mcp
```

## Notes

- The container runs in keyword-only search mode by default (no local Ollama in the container).
- For semantic search, point `embedding.base_url` at an OpenAI-compatible embedding endpoint:
  ```bash
  docker run --rm -v ~/my-notes:/vault \
    -e SAME_EMBED_PROVIDER=openai-compatible \
    -e SAME_EMBED_MODEL=nomic-embed-text-v1.5 \
    -e SAME_EMBED_BASE_URL=http://host.docker.internal:1234/v1 \
    same search "authentication"
  ```
- The vault's `.same/data/` directory (containing the SQLite database) is persisted in your mounted volume.
- `same web` binds to localhost by design. For dashboard access, run `same web` directly on the host.
