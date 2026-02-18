//go:build unix

package mmapforge

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

func deferMunmap(t *testing.T, addr uintptr, length int) {
	t.Helper()
	if err := munmapAt(addr, length); err != nil {
		t.Errorf("munmapAt cleanup: %v", err)
	}
}

func deferSysMunmap(t *testing.T, b []byte) {
	t.Helper()
	if err := syscall.Munmap(b); err != nil {
		t.Errorf("syscall.Munmap cleanup: %v", err)
	}
}

func TestPageAlign(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"zero", 0, pageSize},
		{"negative", -1, pageSize},
		{"one", 1, pageSize},
		{"exact page", pageSize, pageSize},
		{"page plus one", pageSize + 1, pageSize * 2},
		{"three pages", pageSize * 3, pageSize * 3},
		{"mid second page", pageSize + 500, pageSize * 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pageAlign(tt.in)
			if got != tt.want {
				t.Errorf("pageAlign(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestMapRoundTrip(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	size := pageSize
	r, err := Map(f, size, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	buf := unsafe.Slice((*byte)(unsafe.Pointer(r.base)), size)
	binary.LittleEndian.PutUint64(buf[0:8], 0xDEADBEEF)

	got := binary.LittleEndian.Uint64(buf[0:8])
	if got != 0xDEADBEEF {
		t.Fatalf("read back %#x, want 0xDEADBEEF", got)
	}

	unmapErr := munmapAt(r.base, r.maxVA)
	if unmapErr != nil {
		t.Fatalf("munmap: %v", unmapErr)
	}

	f2, err := os.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	r2, err := Map(f2, size, false, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map reopen: %v", err)
	}
	defer func() { deferMunmap(t, r2.base, r2.maxVA) }()

	buf2 := unsafe.Slice((*byte)(unsafe.Pointer(r2.base)), size)
	got2 := binary.LittleEndian.Uint64(buf2[0:8])
	if got2 != 0xDEADBEEF {
		t.Fatalf("after reopen got %#x, want 0xDEADBEEF", got2)
	}
}

func TestMapInvalidSize(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = Map(f, 0, true, Sequential)
	if err == nil {
		t.Fatal("expected error for size=0, got nil")
	}

	_, err = Map(f, -1, true, Sequential)
	if err == nil {
		t.Fatal("expected error for size=-1, got nil")
	}
}

func TestSysAdvice(t *testing.T) {
	if Sequential.sysAdvice() != syscall.MADV_SEQUENTIAL {
		t.Errorf("Sequential.sysAdvice() = %d, want MADV_SEQUENTIAL (%d)", Sequential.sysAdvice(), syscall.MADV_SEQUENTIAL)
	}
	if Random.sysAdvice() != syscall.MADV_RANDOM {
		t.Errorf("Random.sysAdvice() = %d, want MADV_RANDOM (%d)", Random.sysAdvice(), syscall.MADV_RANDOM)
	}
}

func TestMapReserveVAAutoGrow(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	size := pageSize * 4
	tinyReserve := pageSize

	r, err := Map(f, size, false, Sequential, tinyReserve)
	if err != nil {
		t.Fatalf("Map should succeed when reserveVA < size: %v", err)
	}
	defer func() { deferMunmap(t, r.base, r.maxVA) }()

	if r.maxVA < size {
		t.Errorf("maxVA = %d, want >= %d", r.maxVA, size)
	}
}

func TestMapTruncatesSmallFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() { deferMunmap(t, r.base, r.maxVA) }()

	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() < int64(pageSize) {
		t.Errorf("file size = %d after Map, want >= %d", info.Size(), pageSize)
	}
}

func TestMapSkipsTruncateWhenFileAlreadyLargeEnough(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	bigSize := pageSize * 3
	truncErr := f.Truncate(int64(bigSize))
	if truncErr != nil {
		t.Fatal(truncErr)
	}

	r, err := Map(f, pageSize, false, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() { deferMunmap(t, r.base, r.maxVA) }()

	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != int64(bigSize) {
		t.Errorf("file size changed to %d, want %d (untouched)", info.Size(), bigSize)
	}
}

func TestMapStatError(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	f.Close()

	f2, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	f2.Close()

	_, err = Map(f2, pageSize, false, Sequential, pageSize*4)
	if err == nil {
		t.Fatal("expected error when Stat fails on closed fd")
	}
}

func TestMapTruncateError(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	f.Close()

	f2, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	_, err = Map(f2, pageSize, true, Sequential, pageSize*4)
	if err == nil {
		t.Fatal("expected error when Truncate fails on read-only fd")
	}
}

func TestMadviseAtSuccess(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	truncErr := f.Truncate(int64(pageSize))
	if truncErr != nil {
		t.Fatal(truncErr)
	}

	buf, err := syscall.Mmap(int(f.Fd()), 0, pageSize,
		syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { deferSysMunmap(t, buf) }()

	addr := uintptr(unsafe.Pointer(&buf[0]))

	if advErr := madviseAt(addr, pageSize, syscall.MADV_SEQUENTIAL); advErr != nil {
		t.Errorf("madviseAt(SEQUENTIAL): %v", advErr)
	}
	if advErr := madviseAt(addr, pageSize, syscall.MADV_RANDOM); advErr != nil {
		t.Errorf("madviseAt(RANDOM): %v", advErr)
	}
}

func TestMadviseAtInvalidAddr(t *testing.T) {
	err := madviseAt(0xDEAD0000, pageSize, syscall.MADV_SEQUENTIAL)
	if err == nil {
		t.Error("expected error for madvise on unmapped address")
	}
}

func TestMunmapAtSuccess(t *testing.T) {
	buf, err := syscall.Mmap(-1, 0, pageSize,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Fatal(err)
	}
	addr := uintptr(unsafe.Pointer(&buf[0]))
	if unmapErr := munmapAt(addr, pageSize); unmapErr != nil {
		t.Errorf("munmapAt: %v", unmapErr)
	}
}

func TestMunmapAtInvalidAddr(t *testing.T) {
	err := munmapAt(1, pageSize)
	if err == nil {
		t.Error("expected error for munmap on unmapped address")
	}
}

func TestMmapFixedSuccess(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	truncErr := f.Truncate(int64(pageSize))
	if truncErr != nil {
		t.Fatal(truncErr)
	}

	reserved, err := syscall.Mmap(-1, 0, pageSize*2,
		syscall.PROT_NONE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { deferSysMunmap(t, reserved) }()

	addr := uintptr(unsafe.Pointer(&reserved[0]))

	if fixErr := mmapFixed(addr, pageSize, f, false); fixErr != nil {
		t.Fatalf("mmapFixed: %v", fixErr)
	}

	if fixErr := mmapFixed(addr, pageSize, f, true); fixErr != nil {
		t.Fatalf("mmapFixed writable: %v", fixErr)
	}
}

func TestMmapFixedBadFd(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	reserved, err := syscall.Mmap(-1, 0, pageSize,
		syscall.PROT_NONE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { deferSysMunmap(t, reserved) }()

	addr := uintptr(unsafe.Pointer(&reserved[0]))
	err = mmapFixed(addr, pageSize, f, false)
	if err == nil {
		t.Fatal("expected error for mmapFixed with closed fd")
	}
}

func TestMapDefaultReserveVA(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, 1, true, Sequential)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() { deferMunmap(t, r.base, r.maxVA) }()

	expected := pageAlign(DefaultMaxVA)
	if pageAlign(1) > expected {
		expected = pageAlign(1)
	}
	if r.maxVA != expected {
		t.Errorf("maxVA = %d, want %d", r.maxVA, expected)
	}
}

func TestMapZeroReserveVAUsesDefault(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, 1, true, Sequential, 0)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() { deferMunmap(t, r.base, r.maxVA) }()
}

func TestMapWithRandomAccess(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, false, Random, pageSize*4)
	if err != nil {
		t.Fatalf("Map with Random access: %v", err)
	}
	defer func() { deferMunmap(t, r.base, r.maxVA) }()

	if r.access != Random {
		t.Errorf("access = %d, want Random (%d)", r.access, Random)
	}
}

func TestRegionSizeStored(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	size := pageSize * 2
	r, err := Map(f, size, false, Sequential, pageSize*8)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() { deferMunmap(t, r.base, r.maxVA) }()

	if r.size.Load() != int64(size) {
		t.Errorf("size = %d, want %d", r.size.Load(), size)
	}
}

func TestMapMmapFixedError(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	truncErr := f.Truncate(int64(pageSize))
	if truncErr != nil {
		t.Fatal(truncErr)
	}

	f2, err := os.Open(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	_, err = Map(f2, pageSize, true, Sequential, pageSize*4)
	if err == nil {
		t.Fatal("expected error: PROT_WRITE on read-only fd should fail mmap")
	}
}

func TestMapReserveVAError(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = Map(f, pageSize, false, Sequential, 1<<48)
	if err == nil {
		t.Fatal("expected error reserving impossibly large VA range")
	}
}

func restoreFuncs(oldMmap func(uintptr, int, *os.File, bool) error, oldMadvise func(uintptr, int, int) error) {
	mmapFixedFunc = oldMmap
	madviseFunc = oldMadvise
}

func TestMapMmapFixedErrorCleanup(t *testing.T) {
	oldMmap, oldMadvise := mmapFixedFunc, madviseFunc
	defer restoreFuncs(oldMmap, oldMadvise)

	mmapFixedFunc = func(_ uintptr, _ int, _ *os.File, _ bool) error {
		return syscall.ENOMEM
	}

	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = Map(f, pageSize, true, Sequential, pageSize*4)
	if err == nil {
		t.Fatal("expected error when mmapFixed fails")
	}
}

func TestMapMmapFixedAddressMismatch(t *testing.T) {
	oldMmap, oldMadvise := mmapFixedFunc, madviseFunc
	defer restoreFuncs(oldMmap, oldMadvise)

	mmapFixedFunc = func(addr uintptr, _ int, _ *os.File, _ bool) error {
		return fmt.Errorf("mmapforge: mmap: expected address %#x, got %#x", addr, uintptr(0xBADF00D))
	}

	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = Map(f, pageSize, true, Sequential, pageSize*4)
	if err == nil {
		t.Fatal("expected error when mmapFixed returns wrong address")
	}
}

func TestMapMadviseErrorCleanup(t *testing.T) {
	oldMmap, oldMadvise := mmapFixedFunc, madviseFunc
	defer restoreFuncs(oldMmap, oldMadvise)

	madviseFunc = func(_ uintptr, _ int, _ int) error {
		return syscall.EINVAL
	}

	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = Map(f, pageSize, false, Sequential, pageSize*4)
	if err == nil {
		t.Fatal("expected error when madvise fails")
	}
}

func TestMmapFixedAddressMismatchReal(t *testing.T) {
	old := mmapSyscall
	defer func() { mmapSyscall = old }()

	mmapSyscall = func(_, _, _, _, _, _ uintptr) (uintptr, error) {
		return 0xBADF00D, nil
	}

	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	truncErr := f.Truncate(int64(pageSize))
	if truncErr != nil {
		t.Fatal(truncErr)
	}

	reserved, err := syscall.Mmap(-1, 0, pageSize*2,
		syscall.PROT_NONE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { deferSysMunmap(t, reserved) }()

	addr := uintptr(unsafe.Pointer(&reserved[0]))
	err = mmapFixed(addr, pageSize, f, false)
	if err == nil {
		t.Fatal("expected error for address mismatch")
	}
}

func TestSliceReadWrite(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	size := pageSize
	r, err := Map(f, size, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	s := r.Slice(0, 8)
	binary.LittleEndian.PutUint64(s, 0xCAFEBABE)

	s2 := r.Slice(0, 8)
	got := binary.LittleEndian.Uint64(s2)
	if got != 0xCAFEBABE {
		t.Fatalf("Slice read back %#x, want 0xCAFEBABE", got)
	}
}

func TestSliceAtOffset(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	size := pageSize
	r, maperr := Map(f, size, true, Sequential, pageSize*4)
	if maperr != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	offset := 128
	s := r.Slice(offset, 8)
	binary.LittleEndian.PutUint64(s, 0x1234567890ABCDEF)

	raw := unsafe.Slice((*byte)(unsafe.Pointer(r.base+uintptr(offset))), 8)
	got := binary.LittleEndian.Uint64(raw)
	if got != 0x1234567890ABCDEF {
		t.Fatalf("got %#x at offset %d, want 0x1234567890ABCDEF", got, offset)
	}
}

func TestSliceZeroLength(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	s := r.Slice(0, 0)
	if len(s) != 0 {
		t.Fatalf("Slice(0,0) len = %d, want 0", len(s))
	}
}

func TestMapped(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	size := pageSize * 2
	r, err := Map(f, size, false, Sequential, pageSize*8)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	if r.Mapped() != size {
		t.Errorf("Mapped() = %d, want %d", r.Mapped(), size)
	}
}

func TestMappedAfterUnmap(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, false, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	unmaperr := r.Unmap()
	if unmaperr != nil {
		t.Errorf("Unmap failed: %v", unmaperr)
	}

	if r.Mapped() != 0 {
		t.Errorf("Mapped() after Unmap = %d, want 0", r.Mapped())
	}
}

func TestSyncFlushesToDisk(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	s := r.Slice(0, 8)
	binary.LittleEndian.PutUint64(s, 0xFEEDFACE)

	if err := r.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	buf := make([]byte, 8)
	if _, err := f.ReadAt(buf, 0); err != nil {
		t.Fatal(err)
	}
	got := binary.LittleEndian.Uint64(buf)
	if got != 0xFEEDFACE {
		t.Fatalf("file read back %#x after Sync, want 0xFEEDFACE", got)
	}
}

func TestSyncAfterUnmapReturnsError(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	unmaperr := r.Unmap()
	if unmaperr != nil {
		t.Errorf("Unmap failed: %v", unmaperr)
	}

	if err := r.Sync(); err == nil {
		t.Fatal("expected error from Sync after Unmap")
	}
}

func TestSyncMsyncError(t *testing.T) {
	old := msyncSyscall
	defer func() { msyncSyscall = old }()

	msyncSyscall = func(_, _, _ uintptr) error {
		return syscall.EIO
	}

	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	if err := r.Sync(); err == nil {
		t.Fatal("expected error when msync fails")
	}
}

func TestUnmapIdempotent(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	if err := r.Unmap(); err != nil {
		t.Fatalf("first Unmap: %v", err)
	}

	if err := r.Unmap(); err != nil {
		t.Fatalf("second Unmap (idempotent): %v", err)
	}
}

func TestUnmapClearsSizeAndMaxVA(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, false, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	unmaperr := r.Unmap()
	if unmaperr != nil {
		t.Errorf("Unmap failed: %v", unmaperr)
	}

	if r.size.Load() != 0 {
		t.Errorf("size after Unmap = %d, want 0", r.size.Load())
	}
	if r.maxVA != 0 {
		t.Errorf("maxVA after Unmap = %d, want 0", r.maxVA)
	}
}

func TestCloseUnmapsAndClosesFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}

	r, err := Map(f, pageSize, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if r.Mapped() != 0 {
		t.Errorf("Mapped() after Close = %d, want 0", r.Mapped())
	}

	_, writeErr := f.Write([]byte("x"))
	if writeErr == nil {
		t.Error("expected error writing to closed file")
	}
}

func TestCloseIdempotentUnmap(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}

	r, err := Map(f, pageSize, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	unmaperr := r.Unmap()
	if unmaperr != nil {
		t.Errorf("Unmap failed: %v", unmaperr)
	}

	if err := r.Close(); err != nil {
		t.Fatalf("Close after Unmap: %v", err)
	}
}

func TestGrowExpandsRegion(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	reserveSize := pageSize * 8
	r, err := Map(f, pageSize, true, Sequential, reserveSize)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	s := r.Slice(0, 8)
	binary.LittleEndian.PutUint64(s, 0xAAAA)

	newSize := pageSize * 3
	if err := r.Grow(newSize); err != nil {
		t.Fatalf("Grow: %v", err)
	}

	if r.Mapped() < newSize {
		t.Errorf("Mapped() = %d after Grow, want >= %d", r.Mapped(), newSize)
	}

	s2 := r.Slice(0, 8)
	got := binary.LittleEndian.Uint64(s2)
	if got != 0xAAAA {
		t.Fatalf("data after Grow = %#x, want 0xAAAA", got)
	}

	s3 := r.Slice(pageSize*2, 8)
	binary.LittleEndian.PutUint64(s3, 0xBBBB)
	got2 := binary.LittleEndian.Uint64(r.Slice(pageSize*2, 8))
	if got2 != 0xBBBB {
		t.Fatalf("new region data = %#x, want 0xBBBB", got2)
	}
}

func TestGrowNoOpWhenAlreadyLargeEnough(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	size := pageSize * 4
	r, err := Map(f, size, true, Sequential, pageSize*8)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	before := r.Mapped()
	if err := r.Grow(pageSize); err != nil {
		t.Fatalf("Grow (no-op): %v", err)
	}
	if r.Mapped() != before {
		t.Errorf("Mapped changed from %d to %d on no-op Grow", before, r.Mapped())
	}
}

func TestGrowExceedsMaxVA(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	reserveSize := pageSize * 2
	r, err := Map(f, pageSize, true, Sequential, reserveSize)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	err = r.Grow(reserveSize + pageSize)
	if err == nil {
		t.Fatal("expected error when Grow exceeds maxVA")
	}
}

func TestGrowTruncateError(t *testing.T) {
	dir := t.TempDir()
	f, err := os.CreateTemp(dir, "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	f.Close()

	f2, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	reserveSize := pageSize * 4
	reserved, err := syscall.Mmap(-1, 0, reserveSize, syscall.PROT_NONE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Fatal(err)
	}
	base := uintptr(unsafe.Pointer(&reserved[0]))

	r := &Region{
		file:      f2,
		base:      base,
		maxVA:     reserveSize,
		writeable: true,
	}
	r.size.Store(int64(pageSize))
	defer func() {
		munerr := munmapAt(base, reserveSize)
		if munerr != nil {
			t.Errorf("munmapAt failed: %v", munerr)
		}
	}()

	err = r.Grow(pageSize * 2)
	if err == nil {
		t.Fatal("expected error when Truncate fails in Grow")
	}
}

func TestGrowMmapError(t *testing.T) {
	oldMmap, oldMadvise := mmapFixedFunc, madviseFunc
	defer restoreFuncs(oldMmap, oldMadvise)

	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		mmapFixedFunc = oldMmap
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	mmapFixedFunc = func(_ uintptr, _ int, _ *os.File, _ bool) error {
		return fmt.Errorf("injected mmap error")
	}

	err = r.Grow(pageSize * 2)
	if err == nil {
		t.Fatal("expected error when mmapFixed fails in Grow")
	}
}

func TestGrowMadviseError(t *testing.T) {
	oldMmap, oldMadvise := mmapFixedFunc, madviseFunc
	defer restoreFuncs(oldMmap, oldMadvise)

	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		madviseFunc = oldMadvise
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	madviseFunc = func(_ uintptr, _ int, _ int) error {
		return syscall.EINVAL
	}

	err = r.Grow(pageSize * 2)
	if err == nil {
		t.Fatal("expected error when madvise fails in Grow")
	}
}

func TestGrowPageAligns(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, true, Sequential, pageSize*8)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	if err := r.Grow(pageSize + 1); err != nil {
		t.Fatalf("Grow: %v", err)
	}

	mapped := r.Mapped()
	if mapped%pageSize != 0 {
		t.Errorf("Mapped() = %d after Grow, not page-aligned", mapped)
	}
	if mapped < pageSize+1 {
		t.Errorf("Mapped() = %d, want >= %d", mapped, pageSize+1)
	}
}

func TestGrowPreservesSliceValidity(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, true, Sequential, pageSize*8)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	pre := r.Slice(0, 8)
	binary.LittleEndian.PutUint64(pre, 0xDEADC0DE)

	if err := r.Grow(pageSize * 4); err != nil {
		t.Fatalf("Grow: %v", err)
	}

	got := binary.LittleEndian.Uint64(pre)
	if got != 0xDEADC0DE {
		t.Fatalf("pre-grow slice data = %#x after Grow, want 0xDEADC0DE", got)
	}
}

func TestGrowFileExtended(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r, err := Map(f, pageSize, true, Sequential, pageSize*8)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer func() {
		unmaperr := r.Unmap()
		if unmaperr != nil {
			t.Errorf("Unmap failed: %v", unmaperr)
		}
	}()

	newSize := pageSize * 3
	if growerr := r.Grow(newSize); err != nil {
		t.Fatalf("Grow: %v", growerr)
	}

	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() < int64(newSize) {
		t.Errorf("file size = %d after Grow, want >= %d", info.Size(), newSize)
	}
}

func TestUnmapReturnsError(t *testing.T) {
	r := &Region{
		base:  1,
		maxVA: pageSize,
	}
	r.size.Store(int64(pageSize))

	err := r.Unmap()
	if err == nil {
		t.Fatal("expected error from Unmap with invalid base address")
	}
	if r.size.Load() != 0 {
		t.Errorf("size after failed Unmap = %d, want 0", r.size.Load())
	}
	if r.maxVA != 0 {
		t.Errorf("maxVA after failed Unmap = %d, want 0", r.maxVA)
	}
}

func TestCloseReturnsUnmapError(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}

	r := &Region{
		file:  f,
		base:  1,
		maxVA: pageSize,
	}
	r.size.Store(int64(pageSize))

	err = r.Close()
	if err == nil {
		t.Fatal("expected error from Close when Unmap fails")
	}
	if got := err.Error(); !strings.Contains(got, "unmap") {
		t.Errorf("Close error = %q, want it to mention 'unmap'", got)
	}
}

func TestCloseReturnsFileCloseError(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}

	r, err := Map(f, pageSize, true, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	syscall.Close(int(f.Fd()))

	err = r.Close()
	if err == nil {
		t.Fatal("expected error from Close when file.Close fails")
	}
	if got := err.Error(); !strings.Contains(got, "close") {
		t.Errorf("Close error = %q, want it to mention 'close'", got)
	}
}

func TestMsyncSyscallSuccess(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if truncateErr := f.Truncate(int64(pageSize)); truncateErr != nil {
		t.Fatal(truncateErr)
	}

	buf, err := syscall.Mmap(int(f.Fd()), 0, pageSize, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { deferSysMunmap(t, buf) }()

	addr := uintptr(unsafe.Pointer(&buf[0]))
	if err := msyncSyscall(addr, uintptr(pageSize), uintptr(syscall.MS_SYNC)); err != nil {
		t.Errorf("msyncSyscall on valid mapping: %v", err)
	}
}

func TestMsyncSyscallError(t *testing.T) {
	err := msyncSyscall(0xDEAD0000, uintptr(pageSize), uintptr(syscall.MS_SYNC))
	if err == nil {
		t.Fatal("expected error from msyncSyscall on unmapped address")
	}
}
