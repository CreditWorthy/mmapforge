package mmapforge

import (
	"sync/atomic"
	"unsafe"
)

// SeqBeginWrite marks the start of a write to record idx.
// Increments the 8-byte sequence counter at offset 0 of the record to an odd value.
// Caller must call SeqEndWrite when the write is complete.
func (s *Store) SeqBeginWrite(idx int) {
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
