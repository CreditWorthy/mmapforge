# mmapforge

[![CI](https://github.com/CreditWorthy/mmapforge/actions/workflows/ci.yml/badge.svg)](https://github.com/CreditWorthy/mmapforge/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/CreditWorthy/mmapforge)](https://goreportcard.com/report/github.com/CreditWorthy/mmapforge)
[![GoDoc](https://pkg.go.dev/badge/github.com/CreditWorthy/mmapforge.svg)](https://pkg.go.dev/github.com/CreditWorthy/mmapforge)
[![License](https://img.shields.io/github/license/CreditWorthy/mmapforge)](https://github.com/CreditWorthy/mmapforge/blob/main/LICENSE)

> **Incubating — still a work in progress.**

A zero-copy, mmap-backed typed record store for Go. No serialization. No allocation on reads. No external dependencies.

You define a struct, annotate it, run the code generator, and get a fully typed store that reads and writes directly from memory-mapped files. Field access is a single memory load — ~3ns per read on Apple M4 Pro, zero heap allocations.

## Why

Most storage libraries serialize your data on write and deserialize on read. That costs CPU time and heap allocations. mmapforge skips all of that — your data lives in a flat binary format on disk, memory-mapped into your process. Reading a field is just pointer arithmetic into the mapped region.

This is useful for:
- **Game state** — thousands of entities updated every tick
- **Time-series data** — append-only streams of fixed-size records
- **Caches** — memory-mapped shared state between processes
- **Anything where read speed matters more than flexibility**