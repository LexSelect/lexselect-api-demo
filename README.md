# LexSelect API Demo

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go 1.24+](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](cli/)
[![API Version](https://img.shields.io/badge/API%20Version-2026--09--12-green)](https://api.lexselect.io/api/docs)

CLI tool and code examples for the [LexSelect External API](https://api.lexselect.io/api/docs), targeting the `2026-09-12` response contract. Confirm that the selected server supports a version before using it.

> **Server-to-server API.** Keys (prefixed `lxs_`) must not be embedded in browser
> applications. The gateway answers preflights before authentication and permits
> only configured exact origins for Studio. Use your own backend for integrations.

## Quick Start

1. Get an API key from [Studio → API keys](https://studio.lexselect.io/keys)
2. Copy `.env.example` to `.env` and add your key
3. Build the CLI or run an example

```bash
cd cli && go build -o lexselect .
./lexselect upload example.pdf
# Then use the returned document ID to fetch results:
./lexselect parse <document_id> --include text,paragraphs --json
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
| `DELETE` | `/documents/{id}` | Queue permanent deletion of a document or folder | `delete` |
| `GET` | `/documents/{id}/processing/latest` | Get the latest processing status | `status` |
| `GET` | `/documents/{id}/processing/latest/pages` | List page metadata | `pages` |
| `GET` | `/documents/{id}/parse` | Get the parsed structure (tree, kvps, text, tables, blocks) | `parse` |
| `GET` | `/documents/{id}/render` | Render to Markdown/HTML (experimental) | `render` |

### Processing status (polling)

For the `2026-09-12` API contract, `GET /documents/{id}/processing/latest`
reports progress as facts — counts per scope — instead of a computed fraction:

```json
{
  "status": "in_progress",
  "stage": "page-processing",
  "pages_done": 7,
  "pages_total": 12,
  "total_known": true,
  "progress_details": [
    {"scope": "page", "kind": "page", "done": 7, "total": 12, "total_known": true}
  ]
}
```

When `total_known` is `false`, the total may still grow — render counts honestly
(e.g. `7/12+`) and do not derive a percentage. Pinning `X-API-Version: 2026-03-06`
keeps the legacy shape (`processing_progress` float + `page_count`). Note that the
`/parse` response's `page_count` is the response batch size. While processing,
`total_pages` counts currently successful pages and may grow; it is not proof
that the whole document has finished.

`in_progress` includes normal preparation/capacity waits. `failed` is a failed
processing run. `unknown` can represent an unmapped state such as paused/cancelled
work; it is not success. Use bounded polling with a client deadline, backoff and
jitter, and respect `Retry-After` when supplied. For many documents, use signed
webhooks and durably save results before automatic retention deletes them.

## Upload flow

Upload a document in a single request: `POST /documents/upload` with a
`multipart/form-data` body containing `name` and `file`. Omit `size`, or send zero,
for buffered mode. A positive exact byte count selects streaming; all metadata
must then precede `file`. The server stores the bytes and triggers asynchronous
processing, returning the created document rather than a finished parse.
Optionally pass `parent_id`
(defaults to your Default Project) and `content_type` (inferred from the file
name otherwise).

The accepted input extensions are `pdf`, `doc`, `docx`, `odt`, `rtf`, `eml` and
`msg`. A `pages` filter on result reads does not reduce processing or billing.

DELETE acknowledges queued, irreversible removal. Repeating it can succeed while
the deleted row still exists, then returns `404` after purge. API documents have
automatic retention; store results in your own system promptly. Deleting content
does not erase required usage/accounting records. See the API reference for the
current retention and limit rules.

## Error model

Every response (success or error) carries an `X-Request-Id` header for tracing. Errors are
[RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) `application/problem+json` with the shape
`{ type, title, status, detail }`. Note: `limit` outside `1–100` returns `400` (it is **not**
clamped), and oversized request bodies return `413` problem+json.

Fix invalid requests/credentials before retrying. Back off on rate limits and
transient read errors. An upload timeout can happen after processing has started:
reconcile a returned document ID, or list documents in the upload parent when
no ID was received, before sending a new upload. Idempotency response
caching is scoped to the API key/string for 24 hours, is not bound to endpoint/body,
and has no in-flight lock. Use a unique string per operation/payload; it is not an
exactly-once or file-content deduplication guarantee.

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

### TLS and local development

Use valid HTTPS and certificate verification for real API requests. If a local
test server uses your own CA, configure that CA in the client's trust store;
keep local credentials and configuration out of source control. These examples
do not require a presigned browser PUT or disabling TLS verification.

## Links

- [API Documentation](https://api.lexselect.io/api/docs) — Interactive API reference
- [OpenAPI Spec](https://api.lexselect.io/api/openapi.yaml) — Machine-readable spec
