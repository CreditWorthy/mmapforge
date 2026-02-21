# Changelog

## v0.1.0 (Unreleased)

Initial release.

### Features

- Zero-copy, mmap-backed typed record store
- Code generator (`mmapforge`) parses structs with `mmap` tags and generates fully typed stores
- Read/write support for all Go primitive types, strings, and byte slices
- Per-record seqlock protocol for lock-free concurrent reads
- Automatic file growth with stable base address (MAP_FIXED remapping)
- Header with magic bytes, format version, and schema hash validation
- Layout engine with proper alignment and deterministic schema hashing
- Crash recovery: stuck seqlock counters auto-reset on OpenStore

### Performance

- ~2ns per field read, ~2ns per field write (Apple M4 Pro)
- Zero heap allocations on reads
- 179x faster than os.File ReadAt, 337x faster than os.File WriteAt

### Testing

- 100% code coverage across all packages
- Race detector clean
- Fuzz testing on header parser (21M+ executions, zero crashes)

### Documentation

- README with install, usage, benchmarks, and crash safety docs
- Godoc comments on all exported types and functions
- CONTRIBUTING.md
