// Package errlog persists PuffSQL diagnostics to a log file that lives beside the executable.
//
// The log exists so a failing or stalled SQL operation can still be diagnosed after the UI has
// lost the message. It is append-only across sessions, size rotated, and never records result
// rows, passwords or other secrets.
package errlog

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	// FileName is the log written next to PuffSQL.exe. Users can open it from the messages panel.
	FileName = "PuffSQL-error.log"
	// DefaultMaxSize rotates the file before it grows past 5 MiB.
	DefaultMaxSize = int64(5 << 20)
	// DefaultMaxBackups keeps two older files: PuffSQL-error.log.1 and .2.
	DefaultMaxBackups = 2

	maxSQLLength  = 8000
	maxStackBytes = 16000
)

// Entry is one diagnostic record. Empty fields are omitted from the rendered line.
type Entry struct {
	Level        string
	Source       string
	ConnectionID string
	Database     string
	Phase        string
	Code         *int
	Line         *int
	Elapsed      time.Duration
	SQL          string
	Message      string
	Stack        string
}

// Logger appends diagnostics to a single file. A nil logger is valid and discards records, which
// keeps tests and optional call sites simple.
type Logger struct {
	path       string
	mu         sync.Mutex
	maxSize    int64
	maxBackups int
}

// New returns a logger writing to path. An empty path logs to stderr only.
func New(path string) *Logger {
	return &Logger{path: path, maxSize: DefaultMaxSize, maxBackups: DefaultMaxBackups}
}

// Path returns the file currently in use, or an empty string when only stderr is available.
func (l *Logger) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// ResolvePath prefers a writable log beside the executable and falls back to fallbackDir when the
// executable directory is read-only (for example an installation under Program Files). The bool
// reports whether the fallback was used.
func ResolvePath(fallbackDir string) (string, bool) {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), FileName)
		if writable(candidate) {
			return candidate, false
		}
	}
	if strings.TrimSpace(fallbackDir) != "" {
		candidate := filepath.Join(fallbackDir, FileName)
		if writable(candidate) {
			return candidate, true
		}
	}
	return "", true
}

// Write appends one entry. It never panics and never blocks callers on a write failure.
func (l *Logger) Write(entry Entry) {
	if l == nil {
		return
	}
	text := formatEntry(entry)
	if l.path == "" {
		log.Print(strings.TrimSpace(text))
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.rotateLocked()
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		log.Printf("errlog: create directory: %v", err)
		return
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		log.Printf("errlog: open %s: %v", l.path, err)
		return
	}
	defer file.Close()
	if _, err := file.WriteString(text); err != nil {
		log.Printf("errlog: write %s: %v", l.path, err)
	}
}

func (l *Logger) rotateLocked() {
	info, err := os.Stat(l.path)
	if err != nil || info.Size() < l.maxSize {
		return
	}
	backups := l.maxBackups
	if backups < 1 {
		backups = 1
	}
	_ = os.Remove(fmt.Sprintf("%s.%d", l.path, backups))
	for i := backups - 1; i >= 1; i-- {
		_ = os.Rename(
			fmt.Sprintf("%s.%d", l.path, i),
			fmt.Sprintf("%s.%d", l.path, i+1),
		)
	}
	if err := os.Rename(l.path, l.path+".1"); err != nil {
		log.Printf("errlog: rotate %s: %v", l.path, err)
	}
}

func writable(path string) bool {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

func formatEntry(entry Entry) string {
	level := strings.ToUpper(strings.TrimSpace(entry.Level))
	if level == "" {
		level = "ERROR"
	}
	var body strings.Builder
	fmt.Fprintf(&body, "[%s] [%s]", time.Now().Format("2006-01-02 15:04:05.000"), level)
	if entry.Source != "" {
		fmt.Fprintf(&body, " source=%s", entry.Source)
	}
	if entry.ConnectionID != "" {
		fmt.Fprintf(&body, " conn=%s", entry.ConnectionID)
	}
	if entry.Database != "" {
		fmt.Fprintf(&body, " db=%s", entry.Database)
	}
	if entry.Phase != "" {
		fmt.Fprintf(&body, " phase=%s", entry.Phase)
	}
	if entry.Code != nil {
		fmt.Fprintf(&body, " code=%d", *entry.Code)
	}
	if entry.Line != nil {
		fmt.Fprintf(&body, " line=%d", *entry.Line)
	}
	if entry.Elapsed > 0 {
		fmt.Fprintf(&body, " elapsed=%dms", entry.Elapsed.Milliseconds())
	}
	body.WriteString("\n")
	if entry.SQL != "" {
		fmt.Fprintf(&body, "SQL:\n%s\n", truncate(Redact(entry.SQL), maxSQLLength))
	}
	if entry.Message != "" {
		fmt.Fprintf(&body, "%s\n", Redact(entry.Message))
	}
	if entry.Stack != "" {
		fmt.Fprintf(&body, "STACK:\n%s\n", truncate(entry.Stack, maxStackBytes))
	}
	body.WriteString("--------------------------------------------------------------------------------\n\n")
	return body.String()
}

// passwordPattern hides credentials that can appear in DDL or maintenance scripts. It is
// intentionally narrow so ordinary statement text stays readable in the diagnostic log.
var passwordPattern = regexp.MustCompile(`(?i)(\b(?:password|pwd|passwd)\s*=\s*(?:N)?')(?:[^']|'')*(')`)

// Redact masks password assignments without touching the rest of a statement.
func Redact(value string) string {
	return passwordPattern.ReplaceAllString(value, "${1}***${2}")
}

func truncate(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "\n... (truncated)"
}
