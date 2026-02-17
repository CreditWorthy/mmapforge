//go:build unix

package mmapforge

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"testing"
	"unsafe"
)

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

	if err := munmapAt(r.base, r.maxVA); err != nil {
		t.Fatalf("munmap: %v", err)
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
	defer munmapAt(r2.base, r2.maxVA)

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
	defer munmapAt(r.base, r.maxVA)

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
	defer munmapAt(r.base, r.maxVA)

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
	if err := f.Truncate(int64(bigSize)); err != nil {
		t.Fatal(err)
	}

	r, err := Map(f, pageSize, false, Sequential, pageSize*4)
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	defer munmapAt(r.base, r.maxVA)

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

	if err := f.Truncate(int64(pageSize)); err != nil {
		t.Fatal(err)
	}

	buf, err := syscall.Mmap(int(f.Fd()), 0, pageSize,
		syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Munmap(buf)

	addr := uintptr(unsafe.Pointer(&buf[0]))

	if err := madviseAt(addr, pageSize, syscall.MADV_SEQUENTIAL); err != nil {
		t.Errorf("madviseAt(SEQUENTIAL): %v", err)
	}
	if err := madviseAt(addr, pageSize, syscall.MADV_RANDOM); err != nil {
		t.Errorf("madviseAt(RANDOM): %v", err)
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
	if err := munmapAt(addr, pageSize); err != nil {
		t.Errorf("munmapAt: %v", err)
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

	if err := f.Truncate(int64(pageSize)); err != nil {
		t.Fatal(err)
	}

	reserved, err := syscall.Mmap(-1, 0, pageSize*2,
		syscall.PROT_NONE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Munmap(reserved)

	addr := uintptr(unsafe.Pointer(&reserved[0]))

	if err := mmapFixed(addr, pageSize, f, false); err != nil {
		t.Fatalf("mmapFixed: %v", err)
	}

	if err := mmapFixed(addr, pageSize, f, true); err != nil {
		t.Fatalf("mmapFixed writable: %v", err)
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
	defer syscall.Munmap(reserved)

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
	defer munmapAt(r.base, r.maxVA)

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
	defer munmapAt(r.base, r.maxVA)
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
	defer munmapAt(r.base, r.maxVA)

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
	defer munmapAt(r.base, r.maxVA)

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

	if err := f.Truncate(int64(pageSize)); err != nil {
		t.Fatal(err)
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

	mmapFixedFunc = func(addr uintptr, length int, f *os.File, writable bool) error {
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

	mmapFixedFunc = func(addr uintptr, length int, f *os.File, writable bool) error {
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

	madviseFunc = func(addr uintptr, length int, advise int) error {
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

	mmapSyscall = func(addr, length, prot, flags, fd, offset uintptr) (uintptr, error) {
		return 0xBADF00D, nil
	}

	f, err := os.CreateTemp(t.TempDir(), "mmapforge-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := f.Truncate(int64(pageSize)); err != nil {
		t.Fatal(err)
	}

	reserved, err := syscall.Mmap(-1, 0, pageSize*2,
		syscall.PROT_NONE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Munmap(reserved)

	addr := uintptr(unsafe.Pointer(&reserved[0]))
	err = mmapFixed(addr, pageSize, f, false)
	if err == nil {
		t.Fatal("expected error for address mismatch")
	}
}
