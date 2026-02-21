//go:build unix

package mmapforge

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// the OS manages memory in fixed-size chunks called "pages"
// mmap requires everything to be page-aligned so we grab this once at startup
// and reuse it everywhere instead of asking the OS every time
var pageSize = os.Getpagesize()

// DefaultMaxVA is the fallback virtual address reservation when no reserveVA
// is passed to Map. Set low because callers should always provide an explicit
// value (e.g. StoreReserveVA). The actual reservation is clamped to at least
// the page-aligned file size, so this only controls headroom for future growth.
const DefaultMaxVA = 1 << 30

// functions can be overridden for testing
var mmapFixedFunc = mmapFixed
var madviseFunc = madviseAt
var regionFinalizerFunc = regionFinalizer
var mmapSyscall = func(addr, length, prot, flags, fd, offset uintptr) (uintptr, error) {
	r, _, errno := syscall.Syscall6(syscall.SYS_MMAP, addr, length, prot, flags, fd, offset)
	if errno != 0 {
		return 0, errno
	}
	return r, nil
}
var msyncSyscall = func(addr, length, flags uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_MSYNC, addr, length, flags)
	if errno != 0 {
		return errno
	}
	return nil
}

// rounds n up to the nearest page boundary; if already aligned it stays the same
// n <= 0 gets clamped to one page <- cant map zero bytes
//
// the math is ceiling division:
// (n-1)      <- subtract 1 so exact multiples dont overshoot to the next page
// / pageSize <- integer division gives us which page the last byte lands in (zero-based)
// + 1        <- convert to page count (one-based)
// * pageSize <- convert page count back to bytes
func pageAlign(n int) int {
	if n <= 0 {
		return pageSize
	}
	return ((n-1)/pageSize + 1) * pageSize
}

// hints to the kernel about how we plan to read the mapped region
// when you touch a mapped page the OS loads it from disk on demand ("page fault")
// if we tell it our pattern ahead of time it can prefetch smarter
type AccessPattern int

const (
	// Sequential <- we read front to back; kernel will aggressively prefetch ahead
	Sequential AccessPattern = iota

	// Random <- we jump around; kernel skips prefetch, keeps more pages cached instead
	Random
)

// Region is a page-aligned, memory-mapped view of a file with a stable
// base address. A large virtual address range is reserved up front with
// PROT_NONE. The file is mapped over the start of that range using
// MAP_FIXED. On Grow the file is extended and remapped at the same
// base address, so pointers and slices obtained from Slice remain valid
// as long as they fall within the previously mapped size.
//
// Owns the underlying *os.File. Safe for concurrent reads after Map returns.
type Region struct {
	file      *os.File
	base      uintptr
	maxVA     int
	size      atomic.Int64
	access    AccessPattern
	writeable bool
}

// Map opens a memory-mapped view of f starting at offset 0.
//
// A virtual address range of maxVA bytes is reserved (PROT_NONE,
// anonymous). The file is then mapped over the first `size` bytes of
// that reservation using MAP_FIXED|MAP_SHARED. If the file is smaller
// than the requested size it is extended via Truncate.
//
// reserveVA must be >= size. Pass 0 to use DefaultMaxVA.
//
// Caller must call Close when done.
func Map(f *os.File, size int, writable bool, access AccessPattern, reserveVA ...int) (*Region, error) {
	if size <= 0 {
		return nil, fmt.Errorf("mmapforge: map: invalid size %d", size)
	}

	reserveSize := DefaultMaxVA
	if len(reserveVA) > 0 && reserveVA[0] > 0 {
		reserveSize = reserveVA[0]
	}

	aligned := pageAlign(size)
	if aligned > reserveSize {
		reserveSize = aligned
	}

	reserveSize = pageAlign(reserveSize)

	// We reserve a contiguous virtual address range with PROT_NONE <- no memory consumed
	reserved, err := syscall.Mmap(-1, 0, reserveSize, syscall.PROT_NONE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		return nil, fmt.Errorf("mmapforge: reserve %d bytes VA: %w", reserveSize, err)
	}

	// unsafe.Pointer() <- required to obtain the base address from the anonymous mapping reservation
	base := uintptr(unsafe.Pointer(&reserved[0]))

	info, err := f.Stat()
	if err != nil {
		munerr := syscall.Munmap(reserved)
		return nil, errors.Join(
			fmt.Errorf("mmapforge: stat: %w", err),
			munerr,
		)
	}

	// Extend the file to cover the mapped region, otherwise SIGBUS on beyond EOF;
	if info.Size() < int64(size) {
		if err := f.Truncate(int64(size)); err != nil {
			munerr := syscall.Munmap(reserved)
			return nil, errors.Join(
				fmt.Errorf("mmapforge: truncate: %w", err),
				munerr,
			)
		}
	}

	if fixmaperr := mmapFixedFunc(base, size, f, writable); fixmaperr != nil {
		munerror := syscall.Munmap(reserved)
		return nil, errors.Join(
			fmt.Errorf("mmapforge: mmap: %w", fixmaperr),
			munerror,
		)
	}

	if madviseerr := madviseFunc(base, size, access.sysAdvice()); madviseerr != nil {
		munerror := munmapAt(base, reserveSize)
		return nil, errors.Join(
			fmt.Errorf("mmapforge: madvise: %w", madviseerr),
			munerror,
		)
	}

	r := Region{
		base:      base,
		maxVA:     reserveSize,
		file:      f,
		writeable: writable,
		access:    access,
	}

	r.size.Store(int64(size))
	rp := &r
	runtime.SetFinalizer(rp, regionFinalizerFunc)
	return rp, nil
}

// mmapFixed <- syscall wrapper that maps a file at an exact address
// we call SYS_MMAP directly instead of Go's syscall.Mmap because we need MAP_FIXED
//
// PROT_READ   <- we can read the mapped pages
// PROT_WRITE  <- we can write to them (only if writable=true)
// MAP_SHARED  <- writes are visible to other processes and flush back to the file;
//
//	MAP_PRIVATE would give us a copy-on-write fork that never touches the real file
//
// MAP_FIXED   <- "put this mapping at exactly this address" <- without this the kernel
//
//	picks whatever address it wants and our reservation breaks
//
// we verify the returned address matches what we asked for;
// if it doesnt something went seriously wrong
func mmapFixed(addr uintptr, length int, f *os.File, writable bool) error {
	prot := syscall.PROT_READ
	if writable {
		prot |= syscall.PROT_WRITE
	}

	r, err := mmapSyscall(
		addr,
		uintptr(length),
		uintptr(prot),
		uintptr(syscall.MAP_SHARED|syscall.MAP_FIXED),
		f.Fd(),
		0,
	)
	if err != nil {
		return err
	}
	if r != addr {
		return fmt.Errorf("mmapforge: mmap: expected address %#x, got %#x", addr, r)
	}
	return nil
}

// munmapAt <- tears down a mapping at addr for length bytes
// after this any access to those addresses = SIGSEGV (segfault)
// this is the cleanup of mmap; call it when youre done or youll leak VA space
// third arg to Syscall is unused by munmap but Go requires 3 args so we pass 0
func munmapAt(addr uintptr, length int) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_MUNMAP,
		addr,
		uintptr(length),
		0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// madviseAt <- tells the kernel how we plan to use this mapped region
// this is just a hint; kernel can ignore it <- if it returns ENOSYS (not implemented)
// we swallow the error instead of failing the whole Map call
func madviseAt(addr uintptr, length int, advise int) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_MADVISE,
		addr,
		uintptr(length),
		uintptr(advise),
	)

	if errno != 0 && errno != syscall.ENOSYS {
		return errno
	}
	return nil
}

// sysAdvice <- converts our AccessPattern enum to the int the kernel expects for madvise(2)
func (a AccessPattern) sysAdvice() int {
	switch a {
	case Random:
		return syscall.MADV_RANDOM
	default:
		return syscall.MADV_SEQUENTIAL
	}
}

// Sync flushes dirty pages to disk via msync. Block until the kernal confirms the write hit stable storage.
func (r *Region) Sync() error {
	sz := r.size.Load()
	if sz == 0 {
		return fmt.Errorf("mmapforge: sync: %w", ErrClosed)
	}

	err := msyncSyscall(r.base, uintptr(sz), uintptr(syscall.MS_SYNC))
	if err != nil {
		return fmt.Errorf("mmapforge: sync: %w", err)
	}

	return nil
}

// Unmap releases the entire VA reservation. Idempotent.
func (r *Region) Unmap() error {
	if r.size.Load() == 0 && r.maxVA == 0 {
		return nil
	}
	err := munmapAt(r.base, r.maxVA)
	r.size.Store(0)
	r.maxVA = 0
	if err != nil {
		return fmt.Errorf("mmapforge: unmap: %w", err)
	}
	return nil
}

func regionFinalizer(r *Region) {
	if r.size.Load() != 0 || r.maxVA != 0 {
		_, _ = fmt.Fprintf(os.Stderr, "mmapforge: Region for %s was garbage collected without Close()\n", r.file.Name())
		_ = r.Close()
	}
}

// Close unmaps the region and closes the file descriptor.
func (r *Region) Close() error {
	runtime.SetFinalizer(r, nil)
	unmapErr := r.Unmap()
	closeErr := r.file.Close()
	if unmapErr != nil {
		return fmt.Errorf("mmapforge: close: %w", unmapErr)
	}
	if closeErr != nil {
		return fmt.Errorf("mmapforge: close: %w", closeErr)
	}
	return nil
}

// Grow remaps the file to at least minSize bytes (page-aligned) at the
// same base address using MAP_FIXED. No-op if already large enough.
//
// Because the base address never changes, slices from previous Slice
// calls remain valid (they point into the same VA range). New pages
// beyond the old size become accessible after Grow returns.
//
// Must be externally serialized (Store.appendMu).
func (r *Region) Grow(minSize int) error {
	cur := int(r.size.Load())
	if minSize <= cur {
		return nil
	}

	aligned := pageAlign(minSize)
	if aligned > r.maxVA {
		return fmt.Errorf("mmapforge: grow %d exceeds max VA reservation %d", aligned, r.maxVA)
	}

	if err := r.file.Truncate(int64(aligned)); err != nil {
		return fmt.Errorf("mmapforge: grow truncate: %w", err)
	}

	if err := mmapFixedFunc(r.base, aligned, r.file, r.writeable); err != nil {
		return fmt.Errorf("mmapforge: grow mmap: %w", err)
	}

	if err := madviseFunc(r.base, aligned, r.access.sysAdvice()); err != nil {
		return fmt.Errorf("mmapforge: grow madvise: %w", err)
	}

	r.size.Store(int64(aligned))
	return nil
}

// Mapped returns the size of the mapped region in bytes.
func (r *Region) Mapped() int {
	return int(r.size.Load())
}

// Slice returns the mmap byte range [off, off+n) from the stable base.
// Out-of-range panics on purpose so layout bugs surface fast.
// Valid for the lifetime of the Region (base address never changes).
func (r *Region) Slice(offset, n int) []byte {
	// unsafe.Slice: creates a Go slice header over the stable mmap VA range.
	// Safe because the base address is fixed for the Region's lifetime, and
	// the caller is responsible for not accessing beyond the mapped size.
	return unsafe.Slice((*byte)(unsafe.Pointer(r.base+uintptr(offset))), n)
}
