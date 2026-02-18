package mmapforge

import (
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"
)

const initialCapacity = 64

var statFileFunc = func(f *os.File) (os.FileInfo, error) { return f.Stat() }
var encodeHeaderFunc = EncodeHeader

type Store struct {
	region         *Region
	layout         *RecordLayout
	header         *Header
	recordCountPtr *atomic.Uint64
	capacityPtr    *atomic.Uint64
	path           string
	recordSize     int
	appendMu       sync.Mutex
	writable       bool
}

func CreateStore(path string, layout *RecordLayout, schemaVersion uint32) (*Store, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, fmt.Errorf("mmapforge: create %s: %w", path, err)
	}

	hash := SchemaHash(layout.Descriptors())
	h := &Header{
		Magic:         Magic,
		FormatVersion: Version,
		SchemaHash:    hash,
		SchemaVersion: schemaVersion,
		RecordSize:    layout.RecordSize,
		RecordCount:   0,
		Capacity:      uint64(initialCapacity),
	}

	fileSize := HeaderSize + int(layout.RecordSize)*initialCapacity
	region, err := Map(f, fileSize, true, Random, DefaultMaxVA)
	if err != nil {
		closeErr := f.Close()
		return nil, errors.Join(
			fmt.Errorf("mmapforge: map %s: %w", path, err),
			fmt.Errorf("mmapforge: close %s: %w", path, closeErr),
		)
	}

	if encodeErr := encodeHeaderFunc(region.Slice(0, HeaderSize), h); encodeErr != nil {
		closeErr := region.Close()
		return nil, errors.Join(
			fmt.Errorf("mmapforge: encode header: %w", encodeErr),
			fmt.Errorf("mmapforge: close %s: %w", path, closeErr),
		)
	}

	s := &Store{
		region:     region,
		layout:     layout,
		header:     h,
		path:       path,
		writable:   true,
		recordSize: int(layout.RecordSize),
	}

	s.recordCountPtr = (*atomic.Uint64)(unsafe.Pointer(s.region.base + 48))
	s.capacityPtr = (*atomic.Uint64)(unsafe.Pointer(s.region.base + 56))
	return s, nil
}

func OpenStore(path string, layout *RecordLayout) (*Store, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("mmapforge: open %s: %w", path, err)
	}

	info, err := statFileFunc(f)
	if err != nil {
		closeErr := f.Close()
		return nil, errors.Join(
			fmt.Errorf("mmapforge: stat %s: %w", path, err),
			fmt.Errorf("mmapforge: close %s: %w", path, closeErr),
		)
	}

	fileSize := int(info.Size())
	if fileSize < HeaderSize {
		closeErr := f.Close()
		return nil, errors.Join(
			fmt.Errorf("mmapforge: file %s is too small (%d bytes)", path, fileSize),
			fmt.Errorf("mmapforge: close %s: %w", path, closeErr),
		)
	}

	region, err := Map(f, fileSize, true, Random, DefaultMaxVA)
	if err != nil {
		closeErr := f.Close()
		return nil, errors.Join(
			fmt.Errorf("mmapforge: map %s: %w", path, err),
			fmt.Errorf("mmapforge: close %s: %w", path, closeErr),
		)
	}

	h, err := DecodeHeader(region.Slice(0, HeaderSize))
	if err != nil {
		closeErr := region.Close()
		return nil, errors.Join(
			fmt.Errorf("mmapforge: decode header: %w", err),
			fmt.Errorf("mmapforge: close %s: %w", path, closeErr),
		)
	}

	expectedHash := SchemaHash(layout.Descriptors())
	if h.SchemaHash != expectedHash {
		closeErr := region.Close()
		return nil, errors.Join(
			fmt.Errorf("mmapforge: schema hash mismatch: expected %x, got %x", expectedHash, h.SchemaHash),
			fmt.Errorf("mmapforge: close %s: %w", path, closeErr),
		)
	}

	s := &Store{
		region:     region,
		layout:     layout,
		header:     h,
		path:       path,
		writable:   true,
		recordSize: int(layout.RecordSize),
	}

	s.recordCountPtr = (*atomic.Uint64)(unsafe.Pointer(s.region.base + 48))
	s.capacityPtr = (*atomic.Uint64)(unsafe.Pointer(s.region.base + 56))
	return s, nil
}

func (s *Store) Close() error {
	if s.region == nil {
		return fmt.Errorf("mmapforge: close %s: %w", s.path, ErrClosed)
	}

	if s.writable {
		if err := s.flushHeader(); err != nil {
			closeErr := s.region.Close()
			return errors.Join(
				fmt.Errorf("mmapforge: flush header: %w", err),
				fmt.Errorf("mmapforge: close %s: %w", s.path, closeErr),
			)
		}

		if syncErr := s.region.Sync(); syncErr != nil {
			closeErr := s.region.Close()
			return errors.Join(
				fmt.Errorf("mmapforge: sync: %w", syncErr),
				fmt.Errorf("mmapforge: close %s: %w", s.path, closeErr),
			)
		}
	}

	err := s.region.Close()
	s.region = nil
	return err
}

func (s *Store) Sync() error {
	if s.region == nil {
		return fmt.Errorf("mmapforge: sync %s: %w", s.path, ErrClosed)
	}
	if err := s.flushHeader(); err != nil {
		return err
	}
	return s.region.Sync()
}

func (s *Store) Len() int {
	v := s.recordCountPtr.Load()
	if v > uint64(math.MaxInt) {
		panic("mmapforge: record count overflows int")
	}
	return int(v)
}

func (s *Store) Cap() int {
	v := s.capacityPtr.Load()
	if v > uint64(math.MaxInt) {
		panic("mmapforge: capacity overflows int")
	}
	return int(v)
}

func (s *Store) Append() (int, error) {
	if s.region == nil {
		return 0, fmt.Errorf("mmapforge: append %s: %w", s.path, ErrClosed)
	}
	if !s.writable {
		return 0, fmt.Errorf("mmapforge: append %s: %w", s.path, ErrReadOnly)
	}

	s.appendMu.Lock()
	defer s.appendMu.Unlock()

	idx := s.recordCountPtr.Load()
	if idx > s.capacityPtr.Load() {
		if err := s.grow(); err != nil {
			return 0, err
		}
	}

	s.recordCountPtr.CompareAndSwap(idx, idx+1)
	if idx > uint64(math.MaxInt) {
		return 0, fmt.Errorf("mmapforge: append %s: record index %d overflows int", s.path, idx)
	}
	return int(idx), nil
}

func (s *Store) flushHeader() error {
	s.header.RecordCount = s.recordCountPtr.Load()
	return encodeHeaderFunc(s.region.Slice(0, HeaderSize), s.header)
}

func (s *Store) grow() error {
	newCap := s.capacityPtr.Load() * 2
	if newCap == 0 {
		newCap = uint64(initialCapacity)
	}

	recSize := s.recordSize
	if recSize < 0 {
		return fmt.Errorf("mmapforge: grow %s: negative record size", s.path)
	}
	newSize := uint64(HeaderSize) + newCap*uint64(recSize)
	if newSize > uint64(math.MaxInt) {
		return fmt.Errorf("mmapforge: grow %s: size %d overflows address space", s.path, newSize)
	}
	if err := s.region.Grow(int(newSize)); err != nil {
		return fmt.Errorf("mmapforge: grow %s: %w", s.path, err)
	}
	s.capacityPtr.Store(newCap)
	return nil
}
