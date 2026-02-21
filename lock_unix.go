//go:build unix

package mmapforge

import (
	"fmt"
	"os"
	"syscall"
)

// flockExclusive acquires a non-blocking exclusive lock on f.
// Returns ErrLocked if the lock is already held.
func flockExclusive(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		if err == syscall.EWOULDBLOCK {
			return fmt.Errorf("mmapforge: %w", ErrLocked)
		}
		return fmt.Errorf("mmapforge: flock exclusive: %w", err)
	}
	return nil
}

// funlock releases the flock on f.
func funlock(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	if err != nil {
		return fmt.Errorf("mmapforge: funlock: %w", err)
	}
	return nil
}
