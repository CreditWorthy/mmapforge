package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeTempSchema(t *testing.T, dir, src string) string {
	t.Helper()
	path := filepath.Join(dir, "schema.go")
	if err := os.WriteFile(path, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRun_ParseFileError(t *testing.T) {
	err := run("/no/such/file.go", "")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestRun_NoSchemas(t *testing.T) {
	dir := t.TempDir()
	path := writeTempSchema(t, dir, `package x
type Foo struct { A int32 }
`)
	err := run(path, "")
	if err == nil {
		t.Fatal("expected error for no schemas")
	}
	if !strings.Contains(err.Error(), "no mmapforge:schema") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_NewGraphError(t *testing.T) {
	dir := t.TempDir()
	src := `package x
// mmapforge:schema version=1
type Dup struct {
	A int32 ` + "`mmap:\"same\"`" + `
	B uint32 ` + "`mmap:\"same\"`" + `
}
`
	path := writeTempSchema(t, dir, src)
	err := run(path, "")
	if err == nil {
		t.Fatal("expected error from NewGraph (duplicate field names)")
	}
}

func TestRun_GenError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not effective on windows")
	}

	dir := t.TempDir()
	src := `package x
// mmapforge:schema version=1
type Good struct {
	Val int32
}
`
	path := writeTempSchema(t, dir, src)

	badDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(badDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chmodErr := os.Chmod(badDir, 0755); chmodErr != nil {
			t.Log(chmodErr)
		}
	}()

	err := run(path, badDir)
	if err == nil {
		t.Fatal("expected error from Gen (unwritable output dir)")
	}
}

func TestRun_Success_DefaultOutput(t *testing.T) {
	dir := t.TempDir()
	src := `package x
// mmapforge:schema version=1
type Player struct {
	ID uint64
	Score int32
}
`
	path := writeTempSchema(t, dir, src)
	err := run(path, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if strings.Contains(e.Name(), "player") && strings.HasSuffix(e.Name(), ".go") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected generated file in default output dir")
	}
}

func TestRun_Success_ExplicitOutput(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	src := `package x
// mmapforge:schema version=2
type Item struct {
	Price float64
}
`
	path := writeTempSchema(t, dir, src)
	err := run(path, outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Error("expected generated files in explicit output dir")
	}
}

func stubMain(t *testing.T, args []string) (exitCode int, output string) {
	t.Helper()

	origExit := exitFunc
	origStderr := stderr
	origArgs := os.Args
	defer func() {
		exitFunc = origExit
		stderr = origStderr
		os.Args = origArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	exitCode = 0
	exitFunc = func(code int) { exitCode = code }

	var buf bytes.Buffer
	stderr = &buf

	os.Args = args
	flag.CommandLine = flag.NewFlagSet(args[0], flag.ContinueOnError)
	main()
	return exitCode, buf.String()
}

func TestMain_NoInput(t *testing.T) {
	code, out := stubMain(t, []string{"mmapforge"})
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out, "-input flag is required") {
		t.Errorf("stderr = %q, want input flag message", out)
	}
}

func TestMain_RunError(t *testing.T) {
	code, out := stubMain(t, []string{"mmapforge", "-input", "/no/such/file.go"})
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(out, "mmapforge:") {
		t.Errorf("stderr = %q, want error message", out)
	}
}

func TestMain_Success(t *testing.T) {
	dir := t.TempDir()
	path := writeTempSchema(t, dir, `package x
// mmapforge:schema version=1
type Z struct {
	V int32
}
`)
	code, out := stubMain(t, []string{"mmapforge", "-input", path})
	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr: %s", code, out)
	}
}
