package mmapforge

import (
	"sync/atomic"
	"unsafe"
)

// Seqlock — per-record sequence counter for lock-free concurrent reads.
//
// Every record reserves its first 8 bytes as an atomic uint64 sequence counter.
// The protocol is a classic seqlock:
//
//	Writer:
//	  1. SeqBeginWrite  → counter becomes odd  (write in progress)
//	  2. write field(s)
//	  3. SeqEndWrite    → counter becomes even (write complete)
//
//	Reader:
//	  1. seq = SeqReadBegin  → load counter
//	  2. if seq is odd, spin (writer active)
//	  3. read field(s)
//	  4. if !SeqReadValid(seq), goto 1 (writer intervened, retry)
//
// Memory ordering guarantees:
//
// Go's sync/atomic provides sequentially consistent operations on all
// architectures (see https://pkg.go.dev/sync/atomic). On ARM64 (and other
// weakly-ordered CPUs), atomic.Uint64.Add and atomic.Uint64.Load compile
// to instructions with acquire/release semantics (LDAR/STLR on ARM64,
// implicit on x86 TSO). This ensures:
//
//   - SeqBeginWrite (Add) is visible to all cores before any subsequent
//     field writes are observed.
//   - SeqEndWrite (Add) is not reordered before the preceding field writes.
//   - SeqReadBegin (Load) establishes a happens-before edge: field reads
//     that follow cannot be speculatively satisfied with stale cache lines.
//   - SeqReadValid (Load) acts as an acquire fence that confirms no write
//     occurred during the read window.
//
// This means readers never see a torn write, even without explicit memory
// fences — the atomics provide the required ordering. The one caveat is
// process crash: if the writer dies between Begin and End, the counter stays
// odd permanently. See Store.RecoverSeqlocks for crash recovery.

// SeqBeginWrite marks the start of a write to record idx.
// Increments the 8-byte sequence counter at offset 0 of the record to an odd value.
// Caller must call SeqEndWrite when the write is complete.
func (s *Store) SeqBeginWrite(idx int) {
	if !s.writable {
		panic("mmapforge: SeqBeginWrite called on read-only store")
	}
	off := HeaderSize + idx*s.recordSize
	ptr := (*atomic.Uint64)(unsafe.Pointer(s.region.base + uintptr(off)))
	ptr.Add(1)
}

// SeqEndWrite marks the end of a write to record idx.
// Increments the sequence counter to an even value.
func (s *Store) SeqEndWrite(idx int) {
	off := HeaderSize + idx*s.recordSize
	ptr := (*atomic.Uint64)(unsafe.Pointer(s.region.base + uintptr(off)))
	ptr.Add(1)
}

// SeqReadBegin loads the sequence counter for record idx.
// If the value is odd, a write is in progress and the caller should spin.
func (s *Store) SeqReadBegin(idx int) uint64 {
	off := HeaderSize + idx*s.recordSize
	ptr := (*atomic.Uint64)(unsafe.Pointer(s.region.base + uintptr(off)))
	return ptr.Load()
}

// SeqReadValid returns true if seq is even (no write in progress) and
// the current counter still matches seq (no write happened during the read).
func (s *Store) SeqReadValid(idx int, seq uint64) bool {
	off := HeaderSize + idx*s.recordSize
	ptr := (*atomic.Uint64)(unsafe.Pointer(s.region.base + uintptr(off)))
	return seq&1 == 0 && ptr.Load() == seq
}
