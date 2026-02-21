package mmapforge

import (
	"testing"
)

func TestSeqBeginWrite_EndWrite(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}

	seq0 := s.SeqReadBegin(idx)
	if seq0 != 0 {
		t.Fatalf("initial seq = %d, want 0", seq0)
	}

	s.SeqBeginWrite(idx)
	seq1 := s.SeqReadBegin(idx)
	if seq1&1 != 1 {
		t.Errorf("seq after BeginWrite = %d, want odd", seq1)
	}

	s.SeqEndWrite(idx)
	seq2 := s.SeqReadBegin(idx)
	if seq2&1 != 0 {
		t.Errorf("seq after EndWrite = %d, want even", seq2)
	}
	if seq2 != 2 {
		t.Errorf("seq after full cycle = %d, want 2", seq2)
	}
}

func TestSeqReadValid_EvenAndMatching(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}

	seq := s.SeqReadBegin(idx)
	if !s.SeqReadValid(idx, seq) {
		t.Error("SeqReadValid should be true for initial even seq")
	}
}

func TestSeqReadValid_OddSeq(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}

	s.SeqBeginWrite(idx)
	seq := s.SeqReadBegin(idx)
	if s.SeqReadValid(idx, seq) {
		t.Error("SeqReadValid should be false for odd seq (write in progress)")
	}
	s.SeqEndWrite(idx)
}

func TestSeqReadValid_StaleSeq(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}

	seq := s.SeqReadBegin(idx)

	s.SeqBeginWrite(idx)
	s.SeqEndWrite(idx)

	if s.SeqReadValid(idx, seq) {
		t.Error("SeqReadValid should be false when counter changed during read")
	}
}

func TestSeqMultipleCycles(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx, appendErr := s.Append()
	if appendErr != nil {
		t.Fatal(appendErr)
	}

	for i := 0; i < 10; i++ {
		s.SeqBeginWrite(idx)
		s.SeqEndWrite(idx)
	}

	seq := s.SeqReadBegin(idx)
	if seq != 20 {
		t.Errorf("seq after 10 cycles = %d, want 20", seq)
	}
	if !s.SeqReadValid(idx, seq) {
		t.Error("SeqReadValid should be true after completed cycles")
	}
}

func TestSeqMultipleRecords(t *testing.T) {
	s := mustCreateStore(t)
	defer s.Close()

	idx0, appendErr0 := s.Append()
	if appendErr0 != nil {
		t.Fatal(appendErr0)
	}
	idx1, appendErr1 := s.Append()
	if appendErr1 != nil {
		t.Fatal(appendErr1)
	}

	s.SeqBeginWrite(idx0)
	s.SeqEndWrite(idx0)

	seq0 := s.SeqReadBegin(idx0)
	seq1 := s.SeqReadBegin(idx1)

	if seq0 != 2 {
		t.Errorf("record 0 seq = %d, want 2", seq0)
	}
	if seq1 != 0 {
		t.Errorf("record 1 seq = %d, want 0 (untouched)", seq1)
	}
}

func TestSeqBeginWrite_PanicsOnReadOnly(t *testing.T) {
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
	s.Close()

	ro, err := OpenStore(path, layout, WithReadOnly())
	if err != nil {
		t.Fatalf("OpenStore ReadOnly: %v", err)
	}
	defer ro.Close()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from SeqBeginWrite on read-only store")
		}
	}()
	ro.SeqBeginWrite(idx)
}
