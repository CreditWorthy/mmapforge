package mmapforge

import "testing"

func TestApplyOptions_NoOpts(t *testing.T) {
	cfg := applyOptions(nil)
	if cfg.readOnly {
		t.Error("readOnly should be false by default")
	}
	if cfg.oneWriter {
		t.Error("oneWriter should be false by default")
	}
}

func TestApplyOptions_WithReadOnly(t *testing.T) {
	cfg := applyOptions([]StoreOption{WithReadOnly()})
	if !cfg.readOnly {
		t.Error("readOnly should be true")
	}
	if cfg.oneWriter {
		t.Error("oneWriter should be false")
	}
}

func TestApplyOptions_WithOneWriter(t *testing.T) {
	cfg := applyOptions([]StoreOption{WithOneWriter()})
	if cfg.readOnly {
		t.Error("readOnly should be false")
	}
	if !cfg.oneWriter {
		t.Error("oneWriter should be true")
	}
}

func TestApplyOptions_BothOptions(t *testing.T) {
	cfg := applyOptions([]StoreOption{WithReadOnly(), WithOneWriter()})
	if !cfg.readOnly {
		t.Error("readOnly should be true")
	}
	if !cfg.oneWriter {
		t.Error("oneWriter should be true")
	}
}
