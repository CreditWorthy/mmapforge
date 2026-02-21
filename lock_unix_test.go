//go:build unix

package mmapforge

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFlockExclusive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	f, openErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if openErr != nil {
		t.Fatal(openErr)
	}
	defer f.Close()

	if flockErr := flockExclusive(f); flockErr != nil {
		t.Fatalf("flockExclusive: %v", flockErr)
	}

	if unlockErr := funlock(f); unlockErr != nil {
		t.Fatalf("funlock: %v", unlockErr)
	}
}

func TestFlockExclusive_AlreadyLocked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	f1, openErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if openErr != nil {
		t.Fatal(openErr)
	}
	defer f1.Close()

	if flockErr := flockExclusive(f1); flockErr != nil {
		t.Fatalf("first flockExclusive: %v", flockErr)
	}

	f2, openErr2 := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if openErr2 != nil {
		t.Fatal(openErr2)
	}
	defer f2.Close()

	flockErr := flockExclusive(f2)
	if flockErr == nil {
		t.Fatal("expected error for already locked file")
	}
	if !errors.Is(flockErr, ErrLocked) {
		t.Errorf("expected ErrLocked, got: %v", flockErr)
	}

	if unlockErr := funlock(f1); unlockErr != nil {
		t.Fatalf("funlock: %v", unlockErr)
	}
}

func TestFlockExclusive_RelockAfterUnlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	f1, openErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if openErr != nil {
		t.Fatal(openErr)
	}
	defer f1.Close()

	if flockErr := flockExclusive(f1); flockErr != nil {
		t.Fatalf("first lock: %v", flockErr)
	}
	if unlockErr := funlock(f1); unlockErr != nil {
		t.Fatalf("unlock: %v", unlockErr)
	}

	f2, openErr2 := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if openErr2 != nil {
		t.Fatal(openErr2)
	}
	defer f2.Close()

	if flockErr := flockExclusive(f2); flockErr != nil {
		t.Fatalf("second lock after unlock should succeed: %v", flockErr)
	}
	if unlockErr := funlock(f2); unlockErr != nil {
		t.Fatalf("funlock: %v", unlockErr)
	}
}

func TestFlockExclusive_BadFd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	f, openErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if openErr != nil {
		t.Fatal(openErr)
	}
	f.Close()

	flockErr := flockExclusive(f)
	if flockErr == nil {
		t.Fatal("expected error for closed file descriptor")
	}
	if errors.Is(flockErr, ErrLocked) {
		t.Error("should not be ErrLocked for bad fd")
	}
}

func TestFunlock_BadFd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	f, openErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if openErr != nil {
		t.Fatal(openErr)
	}
	f.Close()

	unlockErr := funlock(f)
	if unlockErr == nil {
		t.Fatal("expected error for closed file descriptor")
	}
}
