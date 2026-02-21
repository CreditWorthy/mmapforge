package mmapforge

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
)

func testLayout() *RecordLayout {
	layout, err := ComputeLayout([]FieldDef{
		{Name: "id", Type: FieldUint64},
		{Name: "value", Type: FieldFloat64},
	})
	if err != nil {
		panic(err)
	}
	return layout
}

func tempPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test.mmf")
}

func mustCreateStore(t *testing.T) *Store {
	t.Helper()
	path := tempPath(t)
	s, err := CreateStore(path, testLayout(), 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	return s
}

type savedFuncs struct {
	mf func(uintptr, int, *os.File, bool) error
	ma func(uintptr, int, int) error
	ms func(uintptr, uintptr, uintptr) error
	sf func(*os.File) (os.FileInfo, error)
	ef func([]byte, *Header) error
}

func saveFuncs() savedFuncs {
	return savedFuncs{
		mf: mmapFixedFunc,
		ma: madviseFunc,
		ms: msyncSyscall,
		sf: statFileFunc,
		ef: encodeHeaderFunc,
	}
}

func restoreAllFuncs(s savedFuncs) {
	mmapFixedFunc = s.mf
	madviseFunc = s.ma
	msyncSyscall = s.ms
	statFileFunc = s.sf
	encodeHeaderFunc = s.ef
}

func TestCreateStore(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if s.Len() != 0 {
		t.Errorf("Len = %d, want 0", s.Len())
	}
	if s.Cap() != initialCapacity {
		t.Errorf("Cap = %d, want %d", s.Cap(), initialCapacity)
	}
}

func TestCreateStore_AlreadyExists(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	s.Close()

	_, err = CreateStore(path, layout, 1)
	if err == nil {
		t.Fatal("expected error creating over existing file")
	}
}

func TestCreateStore_BadPath(t *testing.T) {
	_, err := CreateStore("/no/such/dir/test.mmf", testLayout(), 1)
	if err == nil {
		t.Fatal("expected error for bad path")
	}
}

func TestCreateStore_MapFails(t *testing.T) {
	saved := saveFuncs()
	defer restoreAllFuncs(saved)

	mmapFixedFunc = func(_ uintptr, _ int, _ *os.File, _ bool) error {
		return syscall.ENOMEM
	}

	_, err := CreateStore(tempPath(t), testLayout(), 1)
	if err == nil {
		t.Fatal("expected error when Map fails inside CreateStore")
	}
}

func TestCreateStore_EncodeHeaderFails(t *testing.T) {
	saved := saveFuncs()
	defer restoreAllFuncs(saved)

	encodeHeaderFunc = func(_ []byte, _ *Header) error {
		return fmt.Errorf("injected encode error")
	}

	_, err := CreateStore(tempPath(t), testLayout(), 1)
	if err == nil {
		t.Fatal("expected error when EncodeHeader fails inside CreateStore")
	}
}

func TestOpenStore(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	s.Close()

	s2, err := OpenStore(path, layout)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer s2.Close()

	if s2.Len() != 0 {
		t.Errorf("Len = %d, want 0", s2.Len())
	}
	if s2.Cap() != initialCapacity {
		t.Errorf("Cap = %d, want %d", s2.Cap(), initialCapacity)
	}
}

func TestOpenStore_PersistsRecordCount(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, appendErr := s.Append(); appendErr != nil {
			t.Fatalf("Append: %v", appendErr)
		}
	}
	s.Close()

	s2, err := OpenStore(path, layout)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer s2.Close()

	if s2.Len() != 5 {
		t.Errorf("Len = %d, want 5", s2.Len())
	}
}

func TestOpenStore_NonExistent(t *testing.T) {
	_, err := OpenStore(filepath.Join(t.TempDir(), "nope.mmf"), testLayout())
	if err == nil {
		t.Fatal("expected error opening non-existent file")
	}
}

func TestOpenStore_FileTooSmall(t *testing.T) {
	path := tempPath(t)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, writeErr := f.Write([]byte("short")); writeErr != nil {
		t.Fatalf("Write: %v", writeErr)
	}
	f.Close()

	_, err = OpenStore(path, testLayout())
	if err == nil {
		t.Fatal("expected error for file too small")
	}
}

func TestOpenStore_EmptyFile(t *testing.T) {
	path := tempPath(t)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	f.Close()

	_, err = OpenStore(path, testLayout())
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestOpenStore_BadMagic(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	s.Close()

	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	if _, writeErr := f.WriteAt([]byte("BAAD"), 0); writeErr != nil {
		t.Fatalf("WriteAt: %v", writeErr)
	}
	f.Close()

	_, err = OpenStore(path, layout)
	if err == nil {
		t.Fatal("expected error for bad magic")
	}
}

func TestOpenStore_BadFormatVersion(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	s.Close()

	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], 999)
	if _, writeErr := f.WriteAt(buf[:], 4); writeErr != nil {
		t.Fatalf("WriteAt: %v", writeErr)
	}
	f.Close()

	_, err = OpenStore(path, layout)
	if err == nil {
		t.Fatal("expected error for bad format version")
	}
}

func TestOpenStore_SchemaMismatch(t *testing.T) {
	path := tempPath(t)

	s, err := CreateStore(path, testLayout(), 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	s.Close()

	other, err := ComputeLayout([]FieldDef{
		{Name: "x", Type: FieldUint32},
	})
	if err != nil {
		t.Fatalf("ComputeLayout: %v", err)
	}
	_, err = OpenStore(path, other)
	if err == nil {
		t.Fatal("expected schema mismatch error")
	}
}

func TestOpenStore_MapFails(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	s.Close()

	saved := saveFuncs()
	defer restoreAllFuncs(saved)

	mmapFixedFunc = func(_ uintptr, _ int, _ *os.File, _ bool) error {
		return syscall.ENOMEM
	}

	_, err = OpenStore(path, layout)
	if err == nil {
		t.Fatal("expected error when Map fails inside OpenStore")
	}
}

func TestOpenStore_StatFails(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	s.Close()

	saved := saveFuncs()
	defer restoreAllFuncs(saved)

	statFileFunc = func(_ *os.File) (os.FileInfo, error) {
		return nil, fmt.Errorf("injected stat error")
	}

	_, err = OpenStore(path, layout)
	if err == nil {
		t.Fatal("expected error when Stat fails inside OpenStore")
	}
}

func TestClose_Double(t *testing.T) {
	s := mustCreateStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := s.Close(); err == nil {
		t.Fatal("expected error on double close")
	}
}

func TestClose_FlushesHeader(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, appendErr := s.Append(); appendErr != nil {
			t.Fatalf("Append: %v", appendErr)
		}
	}
	s.Close()

	s2, err := OpenStore(path, layout)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer s2.Close()

	if s2.Len() != 5 {
		t.Errorf("Len after reopen = %d, want 5", s2.Len())
	}
}

func TestClose_SyncFails(t *testing.T) {
	saved := saveFuncs()

	s := mustCreateStore(t)

	msyncSyscall = func(_, _, _ uintptr) error {
		return syscall.EIO
	}

	err := s.Close()
	restoreAllFuncs(saved)

	if err == nil {
		t.Fatal("expected error when region.Sync fails during Close")
	}
}

func TestClose_FlushHeaderFails(t *testing.T) {
	saved := saveFuncs()

	s := mustCreateStore(t)

	encodeHeaderFunc = func(_ []byte, _ *Header) error {
		return fmt.Errorf("injected encode error")
	}

	err := s.Close()
	restoreAllFuncs(saved)

	if err == nil {
		t.Fatal("expected error when flushHeader fails during Close")
	}
}

func TestSync(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if _, err := s.Append(); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := s.Sync(); err != nil {
		t.Fatalf("Sync: %v", err)
	}
}

func TestSync_Closed(t *testing.T) {
	s := mustCreateStore(t)
	s.Close()

	if err := s.Sync(); err == nil {
		t.Fatal("expected error syncing closed store")
	}
}

func TestSync_PersistsWithoutClose(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}

	for i := 0; i < 3; i++ {
		if _, appendErr := s.Append(); appendErr != nil {
			t.Fatalf("Append: %v", appendErr)
		}
	}
	if syncErr := s.Sync(); syncErr != nil {
		t.Fatalf("Sync: %v", syncErr)
	}
	s.Close()

	s2, err := OpenStore(path, layout)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer s2.Close()

	if s2.Len() != 3 {
		t.Errorf("Len = %d, want 3", s2.Len())
	}
}

func TestSync_RegionSyncFails(t *testing.T) {
	saved := saveFuncs()
	defer restoreAllFuncs(saved)

	s := mustCreateStore(t)
	defer func() {
		msyncSyscall = saved.ms
		s.Close()
	}()

	msyncSyscall = func(_, _, _ uintptr) error {
		return syscall.EIO
	}

	if err := s.Sync(); err == nil {
		t.Fatal("expected error when region.Sync fails during Sync")
	}
}

func TestSync_FlushHeaderFails(t *testing.T) {
	saved := saveFuncs()
	defer restoreAllFuncs(saved)

	s := mustCreateStore(t)
	defer func() {
		encodeHeaderFunc = saved.ef
		s.Close()
	}()

	encodeHeaderFunc = func(_ []byte, _ *Header) error {
		return fmt.Errorf("injected encode error")
	}

	if err := s.Sync(); err == nil {
		t.Fatal("expected error when flushHeader fails during Sync")
	}
}

func TestAppend_Sequential(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	for i := 0; i < 10; i++ {
		idx, err := s.Append()
		if err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
		if idx != i {
			t.Errorf("Append returned %d, want %d", idx, i)
		}
	}
	if s.Len() != 10 {
		t.Errorf("Len = %d, want 10", s.Len())
	}
}

func TestAppend_Closed(t *testing.T) {
	s := mustCreateStore(t)
	s.Close()

	_, err := s.Append()
	if err == nil {
		t.Fatal("expected error appending to closed store")
	}
}

func TestAppend_ReadOnly(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	s.writable = false
	_, err := s.Append()
	if err == nil {
		t.Fatal("expected error appending to read-only store")
	}
	s.writable = true
}

func TestAppend_TriggersGrow(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	target := initialCapacity + 10
	for i := 0; i < target; i++ {
		idx, err := s.Append()
		if err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
		if idx != i {
			t.Errorf("Append returned %d, want %d", idx, i)
		}
	}

	if s.Len() != target {
		t.Errorf("Len = %d, want %d", s.Len(), target)
	}
	if s.Cap() <= initialCapacity {
		t.Errorf("Cap = %d, expected > %d after grow", s.Cap(), initialCapacity)
	}
}

func TestAppend_GrowDoublesCap(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	for i := 0; i < initialCapacity+2; i++ {
		if _, err := s.Append(); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	if s.Cap() != initialCapacity*2 {
		t.Errorf("Cap = %d, want %d", s.Cap(), initialCapacity*2)
	}
}

func TestAppend_MultipleGrows(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	target := initialCapacity*2 + 10
	for i := 0; i < target; i++ {
		if _, err := s.Append(); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	if s.Len() != target {
		t.Errorf("Len = %d, want %d", s.Len(), target)
	}
	if s.Cap() < target {
		t.Errorf("Cap = %d, want >= %d", s.Cap(), target)
	}
}

func TestAppend_GrowFails(t *testing.T) {
	saved := saveFuncs()
	defer restoreAllFuncs(saved)

	s := mustCreateStore(t)
	defer func() {
		mmapFixedFunc = saved.mf
		s.Close()
	}()

	for i := 0; i < initialCapacity; i++ {
		if _, err := s.Append(); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	mmapFixedFunc = func(_ uintptr, _ int, _ *os.File, _ bool) error {
		return syscall.ENOMEM
	}

	_, err := s.Append()
	if err == nil {
		t.Fatal("expected error when grow fails inside Append")
	}
}

func TestAppend_GrowPersists(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}

	target := initialCapacity + 20
	for i := 0; i < target; i++ {
		if _, appendErr := s.Append(); appendErr != nil {
			t.Fatalf("Append %d: %v", i, appendErr)
		}
	}
	s.Close()

	s2, err := OpenStore(path, layout)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer s2.Close()

	if s2.Len() != target {
		t.Errorf("Len = %d, want %d", s2.Len(), target)
	}
}

func TestGrow_ZeroCapacity(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	s.capacityPtr.Store(0)
	if err := s.grow(); err != nil {
		t.Fatalf("grow: %v", err)
	}
	if s.Cap() != initialCapacity {
		t.Errorf("Cap = %d, want %d after grow from zero", s.Cap(), initialCapacity)
	}
}

func TestGrow_RegionGrowFails(t *testing.T) {
	saved := saveFuncs()
	defer restoreAllFuncs(saved)

	s := mustCreateStore(t)
	defer func() {
		mmapFixedFunc = saved.mf
		s.Close()
	}()

	mmapFixedFunc = func(_ uintptr, _ int, _ *os.File, _ bool) error {
		return syscall.ENOMEM
	}

	err := s.grow()
	if err == nil {
		t.Fatal("expected error when region.Grow fails")
	}
}

func TestGrow_CapacityUpdatedAfterSuccess(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	before := s.Cap()
	if err := s.grow(); err != nil {
		t.Fatalf("grow: %v", err)
	}
	after := s.Cap()

	if after != before*2 {
		t.Errorf("Cap after grow = %d, want %d", after, before*2)
	}
}

func TestGrow_NegativeRecordSize(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	s.recordSize = -1
	err := s.grow()
	if err == nil {
		t.Fatal("expected error for negative record size")
	}
}

func TestGrow_OverflowsAddressSpace(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	s.capacityPtr.Store(200_000_000_000_000_000)
	err := s.grow()
	if err == nil {
		t.Fatal("expected error when newSize overflows address space")
	}
}

func TestAppend_IndexOverflowsInt(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	s.recordCountPtr.Store(uint64(math.MaxInt) + 1)
	s.capacityPtr.Store(uint64(math.MaxInt) + 2)

	_, err := s.Append()
	if err == nil {
		t.Fatal("expected error when record index overflows int")
	}
}

func TestLen_PanicsOnOverflow(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	s.recordCountPtr.Store(uint64(math.MaxInt) + 1)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from Len overflow")
		}
	}()
	s.Len()
}

func TestCap_PanicsOnOverflow(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	s.capacityPtr.Store(uint64(math.MaxInt) + 1)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from Cap overflow")
		}
	}()
	s.Cap()
}

func TestAppend_Concurrent(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	goroutines := 8
	perGoroutine := 50
	total := goroutines * perGoroutine

	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make(chan error, total)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				_, err := s.Append()
				if err != nil {
					errs <- err
				}
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("Append error: %v", err)
	}
	if s.Len() != total {
		t.Errorf("Len = %d, want %d", s.Len(), total)
	}
}

func TestAppend_ConcurrentIndicesUnique(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	goroutines := 4
	perGoroutine := 100
	total := goroutines * perGoroutine

	results := make(chan int, total)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				idx, err := s.Append()
				if err != nil {
					t.Errorf("Append: %v", err)
					return
				}
				results <- idx
			}
		}()
	}
	wg.Wait()
	close(results)

	seen := make(map[int]bool, total)
	for idx := range results {
		if seen[idx] {
			t.Errorf("duplicate index %d", idx)
		}
		seen[idx] = true
	}
	if len(seen) != total {
		t.Errorf("unique indices = %d, want %d", len(seen), total)
	}
}

func TestLen_And_Cap(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	if s.Len() != 0 {
		t.Errorf("initial Len = %d, want 0", s.Len())
	}
	if s.Cap() != initialCapacity {
		t.Errorf("initial Cap = %d, want %d", s.Cap(), initialCapacity)
	}

	for i := 0; i < 3; i++ {
		if _, appendErr := s.Append(); appendErr != nil {
			t.Fatalf("Append %d: %v", i, appendErr)
		}
	}

	if s.Len() != 3 {
		t.Errorf("Len = %d, want 3", s.Len())
	}
	if s.Cap() != initialCapacity {
		t.Errorf("Cap = %d, want %d (should not grow)", s.Cap(), initialCapacity)
	}
}

func TestRecoverSeqlocks_StuckOddCounter(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	s.SeqBeginWrite(idx)
	seq := s.SeqReadBegin(idx)
	if seq&1 != 1 {
		t.Fatalf("seq should be odd after BeginWrite, got %d", seq)
	}

	if closeErr := s.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	s2, openErr := OpenStore(path, layout)
	if openErr != nil {
		t.Fatalf("OpenStore: %v", openErr)
	}
	defer s2.Close()

	recovered := s2.SeqReadBegin(idx)
	if recovered&1 != 0 {
		t.Errorf("seq after recovery = %d, want even", recovered)
	}
	if recovered != seq+1 {
		t.Errorf("seq after recovery = %d, want %d", recovered, seq+1)
	}
	if !s2.SeqReadValid(idx, recovered) {
		t.Error("SeqReadValid should be true after recovery")
	}
}

func TestRecoverSeqlocks_MultipleRecords(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}

	idx0, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	idx1, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}
	idx2, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	// record 0: clean (full write cycle)
	s.SeqBeginWrite(idx0)
	s.SeqEndWrite(idx0)

	// record 1: stuck (begin write, no end)
	s.SeqBeginWrite(idx1)

	// record 2: untouched

	if closeErr := s.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	s2, err := OpenStore(path, layout)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer s2.Close()

	seq0 := s2.SeqReadBegin(idx0)
	if seq0 != 2 {
		t.Errorf("record 0 seq = %d, want 2 (was clean)", seq0)
	}

	seq1 := s2.SeqReadBegin(idx1)
	if seq1&1 != 0 {
		t.Errorf("record 1 seq = %d, want even (was stuck)", seq1)
	}

	seq2 := s2.SeqReadBegin(idx2)
	if seq2 != 0 {
		t.Errorf("record 2 seq = %d, want 0 (untouched)", seq2)
	}
}

func TestRecoverSeqlocks_NoRecords(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	if closeErr := s.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	s2, err := OpenStore(path, layout)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer s2.Close()

	if s2.Len() != 0 {
		t.Errorf("Len = %d, want 0", s2.Len())
	}
}

func TestRecoverSeqlocks_AllClean(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		idx, err := s.Append()
		if err != nil {
			t.Fatal(err)
		}
		s.SeqBeginWrite(idx)
		s.SeqEndWrite(idx)
	}

	if closeErr := s.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}

	s2, openErr := OpenStore(path, layout)
	if openErr != nil {
		t.Fatalf("OpenStore: %v", openErr)
	}
	defer s2.Close()

	for i := 0; i < 5; i++ {
		seq := s2.SeqReadBegin(i)
		if seq != 2 {
			t.Errorf("record %d seq = %d, want 2", i, seq)
		}
	}
}

// --- WithReadOnly tests ---

func TestCreateStore_WithReadOnly(t *testing.T) {
	_, err := CreateStore(tempPath(t), testLayout(), 1, WithReadOnly())
	if err == nil {
		t.Fatal("expected error creating store in read-only mode")
	}
}

func TestOpenStore_WithReadOnly(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	if _, appendErr := s.Append(); appendErr != nil {
		t.Fatalf("Append: %v", appendErr)
	}
	s.Close()

	ro, err := OpenStore(path, layout, WithReadOnly())
	if err != nil {
		t.Fatalf("OpenStore ReadOnly: %v", err)
	}
	defer ro.Close()

	if ro.Len() != 1 {
		t.Errorf("Len = %d, want 1", ro.Len())
	}
}

func TestOpenStore_WithReadOnly_AppendFails(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	s.Close()

	ro, err := OpenStore(path, layout, WithReadOnly())
	if err != nil {
		t.Fatalf("OpenStore ReadOnly: %v", err)
	}
	defer ro.Close()

	_, appendErr := ro.Append()
	if appendErr == nil {
		t.Fatal("expected error appending to read-only store")
	}
}

func TestOpenStore_WithReadOnly_SkipsRecoverSeqlocks(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}

	idx, err := s.Append()
	if err != nil {
		t.Fatal(err)
	}

	// Leave seqlock stuck (odd)
	s.SeqBeginWrite(idx)
	s.Close()

	// Open read-only — should NOT recover the stuck seqlock
	ro, err := OpenStore(path, layout, WithReadOnly())
	if err != nil {
		t.Fatalf("OpenStore ReadOnly: %v", err)
	}
	defer ro.Close()

	seq := ro.SeqReadBegin(idx)
	if seq&1 != 1 {
		t.Errorf("seq = %d, want odd (read-only should not recover seqlocks)", seq)
	}
}

func TestOpenStore_WithReadOnly_CloseSkipsFlushSync(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	saved := saveFuncs()

	ro, err := OpenStore(path, layout, WithReadOnly())
	if err != nil {
		t.Fatalf("OpenStore ReadOnly: %v", err)
	}

	// Inject failures — should not matter for read-only close
	encodeHeaderFunc = func(_ []byte, _ *Header) error {
		return fmt.Errorf("injected encode error")
	}
	msyncSyscall = func(_, _, _ uintptr) error {
		return syscall.EIO
	}

	err = ro.Close()
	restoreAllFuncs(saved)

	if err != nil {
		t.Fatalf("read-only Close should not flush/sync, but got error: %v", err)
	}
}

func TestOpenStore_WithReadOnly_ReadWorks(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, createErr := CreateStore(path, layout, 1)
	if createErr != nil {
		t.Fatal(createErr)
	}
	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}
	if writeErr := s.WriteUint64(idx, 8, 42); writeErr != nil {
		t.Fatal(writeErr)
	}
	s.Close()

	ro, openErr := OpenStore(path, layout, WithReadOnly())
	if openErr != nil {
		t.Fatalf("OpenStore ReadOnly: %v", openErr)
	}
	defer ro.Close()

	val, readErr := ro.ReadUint64(idx, 8)
	if readErr != nil {
		t.Fatalf("ReadUint64: %v", readErr)
	}
	if val != 42 {
		t.Errorf("ReadUint64 = %d, want 42", val)
	}
}

// --- WithOneWriter tests ---

func TestCreateStore_WithOneWriter(t *testing.T) {
	path := tempPath(t)
	s, err := CreateStore(path, testLayout(), 1, WithOneWriter())
	if err != nil {
		t.Fatalf("CreateStore WithOneWriter: %v", err)
	}
	defer s.Close()

	if s.lockFile == nil {
		t.Fatal("lockFile should not be nil with WithOneWriter")
	}
}

func TestCreateStore_WithOneWriter_SecondFails(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s1, err := CreateStore(path, layout, 1, WithOneWriter())
	if err != nil {
		t.Fatalf("first CreateStore: %v", err)
	}
	defer s1.Close()

	// Second open with OneWriter on the same path should fail
	_, err = OpenStore(path, layout, WithOneWriter())
	if err == nil {
		t.Fatal("expected error for second one-writer open")
	}
}

func TestOpenStore_WithOneWriter(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	s2, err := OpenStore(path, layout, WithOneWriter())
	if err != nil {
		t.Fatalf("OpenStore WithOneWriter: %v", err)
	}
	defer s2.Close()

	if s2.lockFile == nil {
		t.Fatal("lockFile should not be nil with WithOneWriter")
	}
}

func TestOpenStore_WithOneWriter_SecondFails(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	s1, err := OpenStore(path, layout, WithOneWriter())
	if err != nil {
		t.Fatalf("first OpenStore: %v", err)
	}
	defer s1.Close()

	_, err = OpenStore(path, layout, WithOneWriter())
	if err == nil {
		t.Fatal("expected error for second one-writer open")
	}
}

func TestOpenStore_WithOneWriter_AndReadOnly_MutuallyExclusive(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, err := CreateStore(path, layout, 1)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	_, err = OpenStore(path, layout, WithOneWriter(), WithReadOnly())
	if err == nil {
		t.Fatal("expected error for mutually exclusive options")
	}
}

func TestOneWriter_LockReleasedOnClose(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s1, err := CreateStore(path, layout, 1, WithOneWriter())
	if err != nil {
		t.Fatalf("CreateStore: %v", err)
	}
	s1.Close()

	// After close, another writer should succeed
	s2, err := OpenStore(path, layout, WithOneWriter())
	if err != nil {
		t.Fatalf("OpenStore after close should succeed: %v", err)
	}
	s2.Close()
}

func TestClose_WithLock_FlushHeaderFails(t *testing.T) {
	saved := saveFuncs()

	path := tempPath(t)
	s, err := CreateStore(path, testLayout(), 1, WithOneWriter())
	if err != nil {
		t.Fatal(err)
	}

	encodeHeaderFunc = func(_ []byte, _ *Header) error {
		return fmt.Errorf("injected encode error")
	}

	err = s.Close()
	restoreAllFuncs(saved)

	if err == nil {
		t.Fatal("expected error when flushHeader fails during Close with lock")
	}
}

func TestClose_WithLock_SyncFails(t *testing.T) {
	saved := saveFuncs()

	path := tempPath(t)
	s, err := CreateStore(path, testLayout(), 1, WithOneWriter())
	if err != nil {
		t.Fatal(err)
	}

	msyncSyscall = func(_, _, _ uintptr) error {
		return syscall.EIO
	}

	err = s.Close()
	restoreAllFuncs(saved)

	if err == nil {
		t.Fatal("expected error when sync fails during Close with lock")
	}
}

func TestReleaseLock_NilLockFile(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	// lockFile is nil by default (no WithOneWriter)
	if err := s.releaseLock(); err != nil {
		t.Fatalf("releaseLock with nil lockFile should be no-op: %v", err)
	}
}

func TestAcquireLock_BadPath(t *testing.T) {
	s := &Store{path: "/no/such/dir/test.mmf"}
	if err := s.acquireLock(); err == nil {
		t.Fatal("expected error for bad lock path")
	}
}

func TestCreateStore_WithOneWriter_AcquireLockFails(t *testing.T) {
	// Use a path where the .lock file can't be created
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.mmf.lock")

	// Create a directory where the lock file should go, so OpenFile fails
	if err := os.MkdirAll(lockPath, 0755); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "test.mmf")
	_, err := CreateStore(path, testLayout(), 1, WithOneWriter())
	if err == nil {
		t.Fatal("expected error when lock file cannot be created")
	}
}

func TestOpenStore_WithOneWriter_AcquireLockFails(t *testing.T) {
	path := tempPath(t)
	layout := testLayout()

	s, createErr := CreateStore(path, layout, 1)
	if createErr != nil {
		t.Fatal(createErr)
	}
	s.Close()

	// Create a directory where the lock file should go
	lockPath := path + ".lock"
	if mkdirErr := os.MkdirAll(lockPath, 0755); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}

	_, openErr := OpenStore(path, layout, WithOneWriter())
	if openErr == nil {
		t.Fatal("expected error when lock file cannot be created")
	}
}
