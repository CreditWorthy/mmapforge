package mmapforge

import "errors"

// Sentinel errors returned by Store and Region operations.
var (
	ErrSchemaMismatch = errors.New("mmapforge: schema hash mismatch")
	ErrOutOfBounds    = errors.New("mmapforge: index out of bounds")
	ErrCorrupted      = errors.New("mmapforge: file corrupted")
	ErrBadMagic       = errors.New("mmapforge: invalid magic bytes")
	ErrStringTooLong  = errors.New("mmapforge: string exceeds max size")
	ErrBytesTooLong   = errors.New("mmapforge: bytes exceeds max size")
	ErrReadOnly       = errors.New("mmapforge: store is read-only")
	ErrClosed         = errors.New("mmapforge: store is closed")
	ErrInvalidBool    = errors.New("mmapforge: invalid bool value")
	ErrTypeMismatch   = errors.New("mmapforge: field type changed during migration")
	ErrLocked         = errors.New("mmapforge: store is locked by another writer")
)
