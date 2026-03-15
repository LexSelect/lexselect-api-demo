# LexSelect CLI

Command-line tool for the LexSelect External API.

## Install

```bash
go build -o lexselect .
```

## Usage

```bash
# Set your API key
export LEXSELECT_API_KEY=lxs_your_key_here

# Upload a document and wait for processing
./lexselect upload contract.pdf

# List your documents
./lexselect list
./lexselect list --limit 10 --sort created_at --dir desc --type file

# Get document details
./lexselect get <document_id>

# Check processing status
./lexselect status <document_id>

# Get page metadata
./lexselect pages <document_id>

# Delete a document
./lexselect delete <document_id>

# Raw JSON output (for scripting)
./lexselect list --json
./lexselect status <document_id> --json
```

## Configuration

| Flag / Env Var | Description | Default |
|---|---|---|
| `--api-key` / `LEXSELECT_API_KEY` | API key | — |
| `--api-url` / `LEXSELECT_API_URL` | API base URL | `https://api.lexselect.io/api` |
| `--json` | Output raw JSON | `false` |

The CLI also reads from a `.env` file in the current directory.
