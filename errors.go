package mmapforge

import "errors"

var (
	ErrSchemaMismatch = errors.New("mmapforge: schema hash mismatch")
	ErrOutOfBounds    = errors.New("mmapforge: index out of bounds")
	ErrCorrupted      = errors.New("mmapforge: file corrupted")
	ErrBadMagic       = errors.New("mmapforge: invalid magic bytes")
	ErrStringTooLong  = errors.New("mmapforge: string exceeds max size")
	ErrReadOnly       = errors.New("mmapforge: store is read-only")
	ErrClosed         = errors.New("mmapforge: store is closed")
	ErrInvalidBool    = errors.New("mmapforge: invalid bool value")
	ErrTypeMismatch   = errors.New("mmapforge: field type changed during migration")
)
