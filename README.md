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
| `POST` | `/documents` | Create an upload session (step 1 of the 3-step upload) | `upload` |
| `POST` | `/documents/upload` | Single-request multipart upload (all 3 steps server-side) | `upload --single-request` |
| `PUT` | `/documents/{id}` | Complete an upload (step 3; requires `content_hash_sha256`) | `upload` |
| `GET` | `/documents` | List files/folders within a parent (paginated) | `list` |
| `GET` | `/documents/{id}` | Get a document's metadata | `get` |
| `DELETE` | `/documents/{id}` | Soft-delete a document or folder | `delete` |
| `GET` | `/documents/{id}/processing/latest` | Get the latest processing status | `status` |
| `GET` | `/documents/{id}/processing/latest/pages` | List page metadata | `pages` |
| `GET` | `/documents/{id}/parse` | Get the parsed structure (tree, kvps, text, tables, blocks) | `parse` |
| `GET` | `/documents/{id}/render` | Render to Markdown/HTML (experimental) | `render` |

## Upload flow

There are two ways to upload a document.

**3-step flow** (used by the CLI and the examples):

1. **Create** — `POST /documents` with `{ name, size }` → `{ id, upload_url, expires_in }`.
   The `id` is an *upload-session* id (pass it to step 3), not the final document id.
2. **Upload to S3** — `PUT` the bytes to `upload_url`. Set `Content-Type` to the type
   inferred from the file extension (it must match step 1), and send
   `x-amz-checksum-sha256` (base64-encoded SHA-256 of the bytes) so the server can
   verify integrity from S3 metadata without re-downloading the file.
3. **Complete** — `PUT /documents/{id}` with `{ status: "uploaded", content_hash_sha256 }`,
   where `content_hash_sha256` is the **hex** SHA-256 digest. The server verifies it
   against the uploaded bytes; a mismatch, or a missing/malformed digest, returns `400`.

**Single request** — `POST /documents/upload` with a `multipart/form-data` body. Put the
`name` field *before* the `file` part. The server runs all three steps internally and
computes the hash itself.

## Error model

Every response (success or error) carries an `X-Request-Id` header for tracing. Errors are
[RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) `application/problem+json` with the shape
`{ type, title, status, detail }`. Note: `limit` outside `1–100` returns `400` (it is **not**
clamped), and oversized request bodies return `413` problem+json.

## Examples

Minimal, self-contained examples for learning the API.

| Example | Language | Description |
|---------|----------|-------------|
| [upload-and-poll.ts](examples/typescript/upload-and-poll.ts) | TypeScript | Single file: 3-step upload → poll → fetch the parsed result |
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
