# LexSelect API Demo

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go 1.24+](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](cli/)
[![API Version](https://img.shields.io/badge/API%20Version-2026--03--06-green)](https://api.lexselect.io/api/docs)

CLI tool and code examples for the [LexSelect External API](https://api.lexselect.io/api/docs).

## Quick Start

1. Get an API key from the [Developer Portal](https://app.lexselect.io/developers/api/keys)
2. Copy `.env.example` to `.env` and add your key
3. Use the CLI or run an example

## Contents

### [`cli/`](cli/) — Go CLI Tool

A proper command-line tool for working with the LexSelect API. Single binary, no runtime dependencies.

```bash
cd cli && go build -o lexselect .
./lexselect upload contract.pdf
./lexselect list
./lexselect status <document_id>
./lexselect pages <document_id>
```

See [cli/README.md](cli/README.md) for full usage.

### [`examples/`](examples/) — Code Examples

Minimal, self-contained examples for learning the API.

| Example | Language | Description |
|---------|----------|-------------|
| [sync-polling.ipynb](examples/python/sync-polling.ipynb) | Python | Jupyter notebook: step-by-step upload, poll, and inspect |
| [upload-and-poll.ts](examples/typescript/upload-and-poll.ts) | TypeScript | Single file: upload a document and wait for processing |

## Configuration

All tools read configuration from environment variables or a `.env` file:

| Variable | Description | Default |
|----------|-------------|---------|
| `LEXSELECT_API_KEY` | Your API key (starts with `lxs_`) | — |
| `LEXSELECT_API_URL` | API base URL | `https://api.lexselect.io/api` |

## Links

- [API Documentation](https://api.lexselect.io/api/docs) — Interactive API reference
- [OpenAPI Spec](https://api.lexselect.io/api/openapi.yaml) — Machine-readable spec
