# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.5.x   | :white_check_mark: |
| < 0.5   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in SAME, please report it responsibly:

**Email:** dev@sgx-labs.dev

**What to include:**
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested fixes (optional)

**Response timeline:**
- Acknowledgment within 48 hours
- Initial assessment within 7 days
- Fix timeline communicated based on severity

**Please do not:**
- Open public GitHub issues for security vulnerabilities
- Exploit vulnerabilities beyond proof-of-concept
- Share vulnerability details before a fix is released

## Security Model

SAME is designed with a local-first security model:

### Data Locality
- All data (embeddings, database, config) stays on your machine
- No telemetry, analytics, or external API calls from SAME itself
- The only network calls are to Ollama (localhost) or optionally OpenAI (if configured)

### Ollama URL Validation
- Ollama URL is validated to be localhost-only (`127.0.0.1`, `localhost`, `::1`)
- Prevents SSRF attacks via malicious config

### Private Content Exclusion
- Directories named `_PRIVATE` are excluded from indexing
- Private content is never surfaced to AI agents
- Configurable skip patterns via `skip_dirs`

### Prompt Injection Protection
- Surfaced snippets are scanned for prompt injection patterns before injection
- Uses [go-promptguard](https://github.com/mdombrov-33/go-promptguard) for detection
- Suspicious content is blocked from context surfacing

### Path Traversal Protection
- MCP `get_note` tool validates paths stay within vault boundary
- Relative path components (`..`) are rejected

### Input Validation
- All user inputs are validated before processing
- SQL queries use parameterized statements (no injection risk)

### Eval Data Boundary

Evaluation test fixtures (ground truth, test queries, expected results) must **never** reference real vault content:

- No real `_PRIVATE/` paths or note titles
- No real client names, project names, or business terms
- No real vault note content or snippets

Eval data must be either **entirely synthetic** or use a **purpose-built demo vault** with public sample data.

## Known Limitations

1. **Trust boundary:** Content surfaced to your AI tool is sent to that tool's API. SAME doesn't control what happens after context is injected.

2. **Embedding model:** If using OpenAI embeddings, your note content is sent to OpenAI's API. Use Ollama for fully local operation.

3. **No encryption at rest:** The SQLite database is not encrypted. Use disk encryption if needed.

## Security Checklist

Run `same doctor` to verify:
- [x] Ollama URL is localhost-only
- [x] Private directories are excluded
- [x] Database is accessible only to current user
- [x] Vector search is functioning
- [x] Context surfacing respects skip patterns
