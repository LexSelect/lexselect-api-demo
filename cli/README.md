# LexSelect CLI

Command-line tool for the LexSelect External API. Single binary, no runtime dependencies.

## Install

```bash
go build -o lexselect .
```

## Configuration

Provide your key via env var, a `.env` file, or the `--api-key` flag:

```bash
export LEXSELECT_API_KEY=lxs_your_key_here
# or put it in a .env file (found in the CWD or any parent directory):
#   LEXSELECT_API_KEY=lxs_...
#   LEXSELECT_API_URL=https://api.lexselect.io/api
```

| Flag | Env var | Description | Default |
|------|---------|-------------|---------|
| `--api-key` | `LEXSELECT_API_KEY` | API key (`lxs_...`) | — |
| `--api-url` | `LEXSELECT_API_URL` | API base URL | `https://api.lexselect.io/api` |
| `--json` | — | Print raw JSON | `false` |

A flag value overrides the env/`.env` value. The CLI searches for `.env` in the
current directory and walks up to parent directories, so a repo-root `.env` works
when running from a subdirectory. The API key is **never** printed in `--help` or
error output.

## Commands

| Command | Endpoint(s) | Description |
|---------|-------------|-------------|
| `upload <file>` | `POST /documents/upload` | Upload a file (one multipart request), wait for processing, print the result |
| `list` | `GET /documents` | List documents and folders |
| `get <id>` | `GET /documents/{id}` | Show a document's metadata |
| `delete <id>` | `DELETE /documents/{id}` | Soft-delete a document or folder |
| `status <id>` | `GET /documents/{id}/processing/latest` | Show processing status |
| `pages <id>` | `GET /documents/{id}/processing/latest/pages` | List page metadata |
| `parse <id>` | `GET /documents/{id}/parse` | Show the parsed structure (use `--json` for the full tree) |
| `render <id>` | `GET /documents/{id}/render` | Render to Markdown or HTML (experimental) |
| `version` | — | Show CLI and API version |

Add `--json` to any command for raw JSON output.

### Processing status fields

`status` (and the polling loop in `upload`) read the latest API version
(`2026-06-07`), where progress is reported as facts rather than a fraction:
`stage` (e.g. `preparing`, `page-processing`, `done`), `pages_done`,
`pages_total`, `total_known`, and a per-scope `progress_details` array:

```json
{"stage": "page-processing", "pages_done": 7, "pages_total": 12, "total_known": true, "progress_details": [{"scope": "page", "kind": "page", "done": 7, "total": 12, "total_known": true}]}
```

When `total_known` is `false` the total may still grow, so the CLI shows
`7/12+ (total not final)` instead of a percentage. Pinning
`X-API-Version: 2026-03-06` keeps the legacy `processing_progress` +
`page_count` shape.

### Examples

```bash
# Upload and wait for processing
./lexselect upload contract.pdf

# List (all flags)
./lexselect list --limit 10 --sort created_at --dir desc --type file
./lexselect list --parent <project_id> --cursor <next_cursor> --flat

# Inspect a processed document
./lexselect status <id>
./lexselect pages <id>
./lexselect parse <id> --include tree,kvps --pages 1-5
./lexselect parse <id> --json
./lexselect render <id> --format markdown
./lexselect render <id> --format html --pages 1

# Manage
./lexselect get <id>
./lexselect delete <id>
```

### `list` flags

| Flag | Description | Default |
|------|-------------|---------|
| `--limit` | Items per page (1–100; out of range → `400`) | `25` |
| `--sort` | Sort field: `name`, `size`, `created_at`, `modified_at` | `name` |
| `--dir` | Sort direction: `asc` or `desc` | `asc` |
| `--type` | Filter by type: `file`, `folder`, `project` | — |
| `--parent` | Parent folder/project ID (omit for root-level projects) | — |
| `--cursor` | Pagination cursor from a previous `next_cursor` | — |
| `--flat` | Return a flat listing instead of one tree level | `false` |

## Errors

API errors are surfaced as `<status> <title>: <detail>` (from the RFC 9457
`application/problem+json` body). Rate-limit responses (`429`) are retried up to 3 times.
