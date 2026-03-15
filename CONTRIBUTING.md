# Contributing

Thanks for your interest in contributing!

## Development

### Go CLI

```bash
cd cli
go build -o lexselect .
go vet ./...
```

### Python Examples

```bash
cd examples/python
python -m venv .venv
source .venv/bin/activate
pip install -e .
jupyter notebook
```

### TypeScript Examples

```bash
cd examples/typescript
npm install
npx tsx upload-and-poll.ts <file.pdf>
```

## Pull Requests

1. Fork the repo and create a branch
2. Make your changes
3. Ensure `go vet` and `go build` pass
4. Open a PR with a clear description
