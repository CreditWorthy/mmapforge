# Contributing to mmapforge

## Prerequisites

- Go 1.24+
- macOS or Linux (unix build tag required)

## Getting started

```bash
git clone https://github.com/CreditWorthy/mmapforge.git
cd mmapforge
go test ./...
```

## Running tests

```bash
# all tests with race detector
go test -race -count=1 ./...

# with coverage
go test -coverprofile=cover.out ./...
go tool cover -html=cover.out

# fuzz the header parser
go test -fuzz=FuzzDecodeHeader -fuzztime=30s

# benchmarks
go test ./... -bench=. -benchmem
```

## Code generation

The `mmapforge` binary is the code generator. To regenerate the example store:

```bash
go build -o mmapforge ./cmd/mmapforge
go generate ./example/...
```

## Project structure

```
mmapforge/
  common.go          - shared constants (Magic, HeaderSize, etc.)
  errors.go          - sentinel errors
  header.go          - binary header encode/decode
  layout.go          - field layout engine and schema hashing
  mmap_unix.go       - memory-mapped Region (Map, Grow, Close, Sync)
  store.go           - Store (CreateStore, OpenStore, Append, grow)
  store_seq.go       - per-record seqlock protocol
  store_read.go      - typed field readers (ReadUint64, ReadString, etc.)
  store_write.go     - typed field writers (WriteUint64, WriteString, etc.)
  cmd/mmapforge/     - code generator CLI
  internal/codegen/  - struct parser and code generator
  example/           - generated MarketCap store with tests and benchmarks
```

## Style

- Run `golangci-lint run` before submitting. The repo has a `.golangci.yml` config.
- Every exported type and function needs a godoc comment.
- Keep 100% test coverage. If you add code, add tests.
- Use the mockable function vars (`mmapFixedFunc`, `madviseFunc`, etc.) for testing syscall paths.

## Submitting a PR

1. Fork the repo and create a branch from `main`.
2. Make your changes. Add tests.
3. Run `go test -race ./...` and `golangci-lint run`.
4. Open a PR with a clear description of what changed and why.
