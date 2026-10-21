package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"puffsql/internal/applog"
	"puffsql/internal/model"
	"puffsql/internal/storage"
)

func testApp(t *testing.T) *App {
	t.Helper()
	store := storage.NewAt(t.TempDir())
	logger := applog.New(store.QueryLogPath(), store.AILogPath())
	if err := logger.Init(); err != nil {
		t.Fatal(err)
	}
	return &App{store: store, logger: logger}
}

func TestTemplateFacadeRoundTrip(t *testing.T) {
	app := testApp(t)
	initial, err := app.LoadCustomTemplates()
	if err != nil {
		t.Fatal(err)
	}
	if len(initial.Templates) != 1 || initial.FilePath == "" {
		t.Fatalf("initial templates = %#v", initial)
	}
	custom := []model.CustomTemplateItem{{
		ID: "team-query", Title: "Team query", Category: "custom",
		Tags: []string{"team"}, Description: "Shared", Code: "SELECT 1;",
	}}
	path, err := app.SaveCustomTemplates(custom)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := app.LoadCustomTemplates()
	if err != nil {
		t.Fatal(err)
	}
	if path != reloaded.FilePath || len(reloaded.Templates) != 1 || reloaded.Templates[0].ID != "team-query" {
		t.Fatalf("reloaded templates = %#v from %s", reloaded, path)
	}
}

func TestSQLFolderFacadeScanReadWriteAndRename(t *testing.T) {
	app := testApp(t)
	root := t.TempDir()
	nested := filepath.Join(root, "Migrations")
	empty := filepath.Join(root, "Empty")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(empty, 0o700); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(nested, "001.sql")
	if err := os.WriteFile(original, []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignore.txt"), []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}

	tree, err := app.ScanSQLFolder(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Children) != 1 || tree.Children[0].Name != "Migrations" {
		t.Fatalf("scan did not include only SQL-bearing folders: %#v", tree.Children)
	}
	content, err := app.ReadSQLFile(original)
	if err != nil || content != "SELECT 1;" {
		t.Fatalf("read = %q, %v", content, err)
	}
	if err := app.WriteSQLFile(original, "SELECT 2;"); err != nil {
		t.Fatal(err)
	}
	renamed := filepath.Join(nested, "002.sql")
	if err := app.RenameSQLPath(original, renamed); err != nil {
		t.Fatal(err)
	}
	content, err = app.ReadSQLFile(renamed)
	if err != nil || content != "SELECT 2;" {
		t.Fatalf("renamed content = %q, %v", content, err)
	}
}

func TestSQLFileReaderHandlesUTF16LittleEndian(t *testing.T) {
	app := testApp(t)
	path := filepath.Join(t.TempDir(), "utf16.sql")
	units := utf16.Encode([]rune("SELECT '中文';"))
	data := []byte{0xFF, 0xFE}
	for _, unit := range units {
		data = binary.LittleEndian.AppendUint16(data, unit)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	content, err := app.ReadSQLFile(path)
	if err != nil || content != "SELECT '中文';" {
		t.Fatalf("UTF-16 read = %q, %v", content, err)
	}
}

func TestLogFacadeLifecycle(t *testing.T) {
	app := testApp(t)
	method := "POST"
	status := 201
	duration := uint64(20)
	failed := false
	if err := app.LogAIRequest("https://example.test", &method, `{"prompt":"hello"}`); err != nil {
		t.Fatal(err)
	}
	if err := app.LogAIResponse("https://example.test", `{"answer":"ok"}`, &status, &duration, &failed); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(app.GetAILogPath())
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "REQUEST CONTENT") || !strings.Contains(text, "HTTP 201") ||
		!strings.Contains(text, `"answer": "ok"`) {
		t.Fatalf("AI log is incomplete:\n%s", text)
	}
	if _, err := app.ClearAILog(); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(app.GetAILogPath())
	if strings.Contains(string(data), "answer") {
		t.Fatal("ClearAILog did not clear the previous request")
	}
	if !strings.HasSuffix(app.GetQueryLogPath(), "puffsql.log") {
		t.Fatalf("query log path = %s", app.GetQueryLogPath())
	}
}

func TestExecuteEmptySQLReturnsEmptyContracts(t *testing.T) {
	app := testApp(t)
	result, err := app.ExecuteQuery("missing", nil, "  \r\n ", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.ResultSets == nil || result.Messages == nil ||
		len(result.ResultSets) != 0 || len(result.Messages) != 0 {
		t.Fatalf("empty query contract = %#v", result)
	}
}
