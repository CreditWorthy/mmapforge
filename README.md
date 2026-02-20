# mmapforge

[![CI](https://github.com/CreditWorthy/mmapforge/actions/workflows/ci.yml/badge.svg)](https://github.com/CreditWorthy/mmapforge/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/CreditWorthy/mmapforge)](https://goreportcard.com/report/github.com/CreditWorthy/mmapforge)
[![GoDoc](https://pkg.go.dev/badge/github.com/CreditWorthy/mmapforge.svg)](https://pkg.go.dev/github.com/CreditWorthy/mmapforge)
[![Coverage](https://codecov.io/gh/CreditWorthy/mmapforge/branch/main/graph/badge.svg)](https://codecov.io/gh/CreditWorthy/mmapforge)
[![License](https://img.shields.io/github/license/CreditWorthy/mmapforge)](https://github.com/CreditWorthy/mmapforge/blob/main/LICENSE)

> **Incubating — still a work in progress.**

A zero-copy, mmap-backed typed record store for Go. No serialization. No allocation on reads. No external dependencies.

You define a struct, annotate it, run the code generator, and get a fully typed store that reads and writes directly from memory-mapped files. Field access is a single memory load - ~3ns per read on Apple M4 Pro, zero heap allocations.

## Why

Most storage libraries serialize your data on write and deserialize on read. That costs CPU time and heap allocations. mmapforge skips all of that - your data lives in a flat binary format on disk, memory-mapped into your process. Reading a field is just pointer arithmetic into the mapped region.

This is useful for:
- **Game state** — thousands of entities updated every tick
- **Time-series data** — append-only streams of fixed-size records
- **Caches** — memory-mapped shared state between processes
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