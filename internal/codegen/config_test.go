package codegen

import "testing"

func TestDefaultHeader(t *testing.T) {
	if DefaultHeader == "" {
		t.Fatal("DefaultHeader should not be empty")
	}
}

func TestConfig_header_Custom(t *testing.T) {
	c := &Config{Header: "// Custom header"}
	if got := c.header(); got != "// Custom header" {
		t.Errorf("header() = %q, want %q", got, "// Custom header")
	}
}

func TestConfig_header_Default(t *testing.T) {
	c := &Config{}
	if got := c.header(); got != DefaultHeader {
		t.Errorf("header() = %q, want %q", got, DefaultHeader)
	}
}
