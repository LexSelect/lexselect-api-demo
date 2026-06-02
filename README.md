# LexSelect API Demo

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go 1.24+](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](cli/)
[![API Version](https://img.shields.io/badge/API%20Version-2026--03--06-green)](https://api.lexselect.io/api/docs)

CLI tool and code examples for the [LexSelect External API](https://api.lexselect.io/api/docs).

> **Server-to-server API.** Keys (prefixed `lxs_`) must not be embedded in browser
> applications, and cross-origin browser requests are not supported — a preflight
> `OPTIONS` returns `401` by design.

## Quick Start

1. Get an API key from the [Developer Portal](https://app.lexselect.io/developers/api/keys)
2. Copy `.env.example` to `.env` and add your key
3. Build the CLI or run an example

```bash
cd cli && go build -o lexselect .
./lexselect upload contract.pdf
```

## API Endpoints

All endpoints live under the base URL (`https://api.lexselect.io/api`) and require
`Authorization: Bearer lxs_...` with `ProcessingAPI` scope. Full, interactive
reference: the [API docs](https://api.lexselect.io/api/docs).

| Method | Path | Description | CLI command |
|--------|------|-------------|-------------|
| `POST` | `/documents/upload` | Upload a file in one multipart request | `upload` |
| `GET` | `/documents` | List files/folders within a parent (paginated) | `list` |
| `GET` | `/documents/{id}` | Get a document's metadata | `get` |
| `DELETE` | `/documents/{id}` | Soft-delete a document or folder | `delete` |
| `GET` | `/documents/{id}/processing/latest` | Get the latest processing status | `status` |
| `GET` | `/documents/{id}/processing/latest/pages` | List page metadata | `pages` |
| `GET` | `/documents/{id}/parse` | Get the parsed structure (tree, kvps, text, tables, blocks) | `parse` |
| `GET` | `/documents/{id}/render` | Render to Markdown/HTML (experimental) | `render` |

## Upload flow

Upload a document in a single request: `POST /documents/upload` with a
`multipart/form-data` body containing `name`, `size`, and the `file` part (put
`name` before `file`). The server stores the bytes, verifies the hash, and
triggers processing, returning the created document. Optionally pass `parent_id`
(defaults to your Default Project) and `content_type` (inferred from the file
name otherwise).

## Error model

Every response (success or error) carries an `X-Request-Id` header for tracing. Errors are
[RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) `application/problem+json` with the shape
`{ type, title, status, detail }`. Note: `limit` outside `1–100` returns `400` (it is **not**
clamped), and oversized request bodies return `413` problem+json.

## Examples

Minimal, self-contained examples for learning the API.

| Example | Language | Description |
|---------|----------|-------------|
| [upload-and-poll.ts](examples/typescript/upload-and-poll.ts) | TypeScript | Single file: upload → poll → fetch the parsed result |
| [sync-polling.ipynb](examples/python/sync-polling.ipynb) | Python | Notebook: upload → poll → pages → parse → render |

## Configuration

| Variable / Flag | Description | Default |
|-----------------|-------------|---------|
| `LEXSELECT_API_KEY` / `--api-key` | Your API key (starts with `lxs_`) | — |
| `LEXSELECT_API_URL` / `--api-url` | API base URL | `https://api.lexselect.io/api` |

The CLI looks for a `.env` in the working directory **and walks up to parent directories**,
so a repo-root `.env` is picked up even when you run the binary from `cli/`.

### Local development

If you point the tools at a local stack whose S3 (LocalStack) serves a self-signed
certificate, a client that doesn't trust it will fail the presigned `PUT`. For the
TypeScript example, run with `NODE_TLS_REJECT_UNAUTHORIZED=0` (local testing only).

## Links

- [API Documentation](https://api.lexselect.io/api/docs) — Interactive API reference
- [OpenAPI Spec](https://api.lexselect.io/api/openapi.yaml) — Machine-readable spec
