package errlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteAppendsRedactedEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	logger := New(path)
	code := 3702
	line := 3
	logger.Write(Entry{
		Level:        "ERROR",
		Source:       "query",
		ConnectionID: "conn-1",
		Database:     "Sales",
		Phase:        "batch1",
		Code:         &code,
		Line:         &line,
		Elapsed:      1500 * time.Millisecond,
		SQL:          "CREATE LOGIN app WITH PASSWORD = N's3cret'; DROP DATABASE [Sales];",
		Message:      `Cannot drop database "Sales" because it is currently in use.`,
	})
	logger.Write(Entry{Level: "INFO", Source: "app", Message: "second entry"})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, marker := range []string{
		"[ERROR]", "source=query", "conn=conn-1", "db=Sales", "phase=batch1",
		"code=3702", "line=3", "elapsed=1500ms", "DROP DATABASE [Sales]",
		"currently in use", "second entry",
	} {
		if !strings.Contains(text, marker) {
			t.Fatalf("log entry is missing %q:\n%s", marker, text)
		}
	}
	if strings.Contains(text, "s3cret") {
		t.Fatalf("password was not redacted:\n%s", text)
	}
	if !strings.Contains(text, "PASSWORD = N'***'") {
		t.Fatalf("redaction shape changed:\n%s", text)
	}
}

func TestRotationKeepsOnlyConfiguredBackups(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	logger := New(path)
	logger.maxSize = 128
	logger.maxBackups = 2

	for i := 0; i < 40; i++ {
		logger.Write(Entry{Level: "ERROR", Source: "test", Message: strings.Repeat("x", 40)})
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("current log missing: %v", err)
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("first backup missing: %v", err)
	}
	if _, err := os.Stat(path + ".3"); !os.IsNotExist(err) {
		t.Fatalf("unexpected third backup: %v", err)
	}
}

func TestResolvePathUsesWritableLocation(t *testing.T) {
	fallback := t.TempDir()
	path, _ := ResolvePath(fallback)
	if path == "" {
		t.Skip("neither the executable directory nor the fallback is writable")
	}
	if filepath.Dir(path) != fallback && !strings.EqualFold(filepath.Base(path), FileName) {
		t.Fatalf("unexpected path %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("resolved log file missing: %v", err)
	}
}

func TestNilLoggerIsSafe(t *testing.T) {
	var logger *Logger
	logger.Write(Entry{Level: "ERROR", Message: "ignored"})
	if logger.Path() != "" {
		t.Fatal("nil logger should report an empty path")
	}
}
