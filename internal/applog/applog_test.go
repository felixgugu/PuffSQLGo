package applog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"puffsql/internal/model"
)

func TestLoggerLifecycleAndRedactionBoundary(t *testing.T) {
	dir := t.TempDir()
	queryPath := filepath.Join(dir, "puffsql.log")
	aiPath := filepath.Join(dir, "ai.log")
	logger := New(queryPath, aiPath)
	if err := logger.Init(); err != nil {
		t.Fatal(err)
	}
	logger.LogQuery("conn", "Sales", "SELECT SecretValue FROM dbo.Users", []model.QueryMessage{{
		Level: "info", Message: "1 row affected", Timestamp: "now",
	}}, 12, "SUCCESS")
	logger.LogAIRequest("https://example.test", "POST", `{"model":"test","messages":[{"role":"user","content":"hello"}]}`)
	status := 200
	duration := uint64(15)
	logger.LogAIResponse("https://example.test", `{"choices":[{"message":{"content":"ok"}}]}`, &status, &duration, false)

	queryData, err := os.ReadFile(queryPath)
	if err != nil {
		t.Fatal(err)
	}
	queryLog := string(queryData)
	if !strings.Contains(queryLog, "SELECT SecretValue FROM dbo.Users") ||
		!strings.Contains(queryLog, "[SUCCESS]") || !strings.Contains(queryLog, "1 row affected") {
		t.Fatalf("query log is incomplete:\n%s", queryLog)
	}
	aiData, err := os.ReadFile(aiPath)
	if err != nil {
		t.Fatal(err)
	}
	aiLog := string(aiData)
	for _, marker := range []string{"REQUEST CONTENT", "RESPONSE CONTENT", "HTTP 200", "Duration: 15ms", `"content": "hello"`} {
		if !strings.Contains(aiLog, marker) {
			t.Fatalf("AI log is missing %q:\n%s", marker, aiLog)
		}
	}

	if _, err := Clear(aiPath, "PuffSQL AI Request Log"); err != nil {
		t.Fatal(err)
	}
	cleared, _ := os.ReadFile(aiPath)
	if strings.Contains(string(cleared), "hello") || !strings.Contains(string(cleared), "PuffSQL AI Request Log") {
		t.Fatalf("clear did not reset the AI log: %s", cleared)
	}
}
