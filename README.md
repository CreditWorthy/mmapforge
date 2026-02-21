# mmapforge

[![CI](https://github.com/CreditWorthy/mmapforge/actions/workflows/ci.yml/badge.svg)](https://github.com/CreditWorthy/mmapforge/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/CreditWorthy/mmapforge)](https://goreportcard.com/report/github.com/CreditWorthy/mmapforge)
[![GoDoc](https://pkg.go.dev/badge/github.com/CreditWorthy/mmapforge.svg)](https://pkg.go.dev/github.com/CreditWorthy/mmapforge)
[![Coverage](https://codecov.io/gh/CreditWorthy/mmapforge/branch/main/graph/badge.svg)](https://codecov.io/gh/CreditWorthy/mmapforge)
[![License](https://img.shields.io/github/license/CreditWorthy/mmapforge)](https://github.com/CreditWorthy/mmapforge/blob/main/LICENSE)

> **Incubating - still a work in progress.**

A zero-copy, mmap-backed typed record store for Go. No serialization. No allocation on reads. No external dependencies.

You define a struct, annotate it, run the code generator, and get a fully typed store that reads and writes directly from memory-mapped files. Field access is a single memory load - ~3ns per read on Apple M4 Pro, zero heap allocations.

## Install

```bash
go install github.com/CreditWorthy/mmapforge/cmd/mmapforge@latest
```

This installs the `mmapforge` code generator. Then add the library to your project:

```bash
go get github.com/CreditWorthy/mmapforge
```

## Usage

### 1. Define your struct

Create a Go file with a struct annotated with `mmap` tags and a `mmapforge:schema` comment:

```go
package mypackage

//go:generate mmapforge -input types.go

// mmapforge:schema version=1
type Tick struct {
    Symbol    string  `mmap:"symbol,64"`
    Price     float64 `mmap:"price"`
    Volume    float64 `mmap:"volume"`
    Timestamp uint64  `mmap:"timestamp"`
}
```

String fields take a max size after the name (e.g. `symbol,64` for a 64-byte string). Numeric fields are fixed size.

### 2. Generate the store

```bash
go generate ./...
```

This creates a `tick_store.go` file with a fully typed `TickStore` that has `Get`/`Set` methods for every field, plus `Append`, `Len`, `Close`, and `Sync`.

### 3. Use it

```go
// Create a new store
store, err := NewTickStore("ticks.mmf")
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// Append a record
idx, err := store.Append()
if err != nil {
    log.Fatal(err)
}

// Write fields
store.SetSymbol(idx, "AAPL")
store.SetPrice(idx, 189.50)
store.SetVolume(idx, 52_000_000)
store.SetTimestamp(idx, uint64(time.Now().UnixNano()))

// Read fields 
price, err := store.GetPrice(idx)
```

All reads and writes go directly to the memory-mapped file. No serialization, no copies. Concurrent reads are lock-free via per-record seqlocks.

## Why

Most storage libraries serialize your data on write and deserialize on read. That costs CPU time and heap allocations. mmapforge skips all of that - your data lives in a flat binary format on disk, memory-mapped into your process. Reading a field is just pointer arithmetic into the mapped region.

This is useful for:
- **Game state** - thousands of entities updated every tick
- **Time-series data** - append-only streams of fixed-size records
- **Caches** - memory-mapped shared state between processes
- **Anything where read speed matters more than flexibility**

## Benchmarks

All benchmarks run on Apple M4 Pro, darwin/arm64, Go 1.24. Run with:

```bash
go test ./... -bench=. -benchmem
```

### Core Store — Read Path 

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| ReadUint64 | 1.79 | 0 | 0 |
| ReadFloat64 | 1.80 | 0 | 0 |
| ReadInt32 | 1.79 | 0 | 0 |
| ReadUint8 | 1.79 | 0 | 0 |
| ReadString | 2.30 | 0 | 0 |
| ReadMultiField (4 fields) | 7.52 | 0 | 0 |

### Core Store — Write Path 

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| WriteUint64 | 1.81 | 0 | 0 |
| WriteFloat64 | 1.81 | 0 | 0 |
| WriteInt32 | 2.01 | 0 | 0 |
| WriteString | 4.09 | 0 | 0 |
| WriteMultiField (4 fields) | 13.44 | 0 | 0 |

### Seqlock 

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| SeqReadBegin | 0.44 | 0 | 0 |
| SeqWriteCycle | 1.40 | 0 | 0 |

### Append

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| Append | 5.12 | 0 | 0 |

### Generated Store (MarketCap example) — Per-field Get 

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| GetID | 3.11 | 0 | 0 |
| GetPrice | 3.23 | 0 | 0 |
| GetVolume | 3.25 | 0 | 0 |
| GetMarketCap | 3.23 | 0 | 0 |
| GetStale | 3.17 | 0 | 0 |

### Generated Store (MarketCap example) — Per-field Set 

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| SetID | 3.55 | 0 | 0 |
| SetPrice | 3.62 | 0 | 0 |
| SetVolume | 3.62 | 0 | 0 |
| SetMarketCap | 3.62 | 0 | 0 |
| SetStale | 3.58 | 0 | 0 |

### Generated Store — Bulk Operations

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| BulkGet (all fields, atomic) | 18.83 | 48 | 1 |
| BulkSet (all fields, atomic) | 10.62 | 0 | 0 |

### Header

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| EncodeHeader | 2.12 | 0 | 0 |
| DecodeHeader | 13.19 | 64 | 1 |

### Layout Engine

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| ComputeLayout (2 fields) | 88.61 | 272 | 3 |
| ComputeLayout (5 fields) | 178.1 | 640 | 3 |
| ComputeLayout (10 fields) | 446.2 | 1768 | 6 |
| SchemaHash | 666.5 | 808 | 22 |

### vs. os.File + encoding/binary Baseline

| Benchmark | ns/op | Speedup |
|---|---|---|
| mmap ReadUint64 | 1.79 | — |
| os.File ReadAt | 319.6 | **179× slower** |
| mmap WriteUint64 | 1.81 | — |
| os.File WriteAt | 609.6 | **337× slower** |

## Crash Safety

mmapforge is a **datastore primitive, not a database**. It provides fast, typed, memory-mapped storage but makes no durability or transactional guarantees. Here is what happens if the process dies unexpectedly:

### What's protected

- **Seqlock recovery** - if a writer crashes mid-write, the per-record sequence counter gets stuck at an odd value. On the next `OpenStore`, all stuck counters are automatically reset so readers don't spin forever. The data in that record may be partially written (torn).

### What's not protected

- **Torn multi-field writes** - writing multiple fields is not atomic. If the process dies mid-write, some fields may have the new value and others the old value. Single aligned 8-byte writes (`WriteUint64`, `WriteFloat64`, etc.) are hardware-atomic on x86/arm64.
- **Stale header** - the on-disk header `RecordCount` is updated on `Sync()` or `Close()`. If neither is called before a crash, the header may report fewer records than were actually appended. The data is present in the file but the count is stale.
- **No fsync on write** - writes go to the kernel page cache via mmap. They are not flushed to stable storage until `Sync()` is called or the kernel decides to write back dirty pages. A power failure (not just process crash) can lose recently written data.

### Recommendations

- Call `Sync()` periodically if you need durability.
- Use mmapforge for hot in-process data (caches, game state, real-time feeds), not as a primary durable store.
- If you need crash-safe transactions, put a WAL or database in front.