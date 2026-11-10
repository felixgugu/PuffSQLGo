package connection

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// slowQueryThreshold controls when a successfully completed query also writes a diagnostic trace.
const slowQueryThreshold = 3 * time.Second

// stallThresholds are the points at which a running query reports that it is still blocked.
var stallThresholds = []time.Duration{
	10 * time.Second,
	30 * time.Second,
	60 * time.Second,
	2 * time.Minute,
	5 * time.Minute,
}

// queryTrace records the phases a statement has reached. It is written to the error log when the
// query fails, is cancelled, runs slowly, or stays blocked long enough for the watchdog to fire.
type queryTrace struct {
	started time.Time
	mu      sync.Mutex
	phases  []traceStamp
	last    string
}

type traceStamp struct {
	name string
	at   time.Duration
}

func newQueryTrace() *queryTrace {
	return &queryTrace{started: time.Now()}
}

func (t *queryTrace) Phase(name string) {
	if t == nil || strings.TrimSpace(name) == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.last = name
	t.phases = append(t.phases, traceStamp{name: name, at: time.Since(t.started)})
	// GO 1000 can produce thousands of repetitions; keep the newest window bounded.
	if len(t.phases) > 256 {
		t.phases = append([]traceStamp(nil), t.phases[len(t.phases)-256:]...)
	}
}

func (t *queryTrace) Last() string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.last
}

func (t *queryTrace) Elapsed() time.Duration {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return time.Since(t.started)
}

func (t *queryTrace) Summary() string {
	if t == nil {
		return ""
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.phases) == 0 {
		return ""
	}
	parts := make([]string, 0, len(t.phases))
	for _, stamp := range t.phases {
		parts = append(parts, fmt.Sprintf("%s=%dms", stamp.name, stamp.at.Milliseconds()))
	}
	return strings.Join(parts, " ")
}

// withTrace appends the phase timeline to a diagnostic message when one was recorded.
func withTrace(message string, trace *queryTrace) string {
	if summary := trace.Summary(); summary != "" {
		return message + "\ntrace: " + summary
	}
	return message
}
