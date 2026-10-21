package connection

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"puffsql/internal/model"
	"puffsql/internal/storage"
)

// TestLiveSQLServerSmoke is opt-in because it uses a real profile and OS keychain entry.
// Run it with PUFFSQL_LIVE=1 and optionally PUFFSQL_TEST_PROFILE=<connection id>.
func TestLiveSQLServerSmoke(t *testing.T) {
	if os.Getenv("PUFFSQL_LIVE") != "1" {
		t.Skip("set PUFFSQL_LIVE=1 to run against the configured SQL Server")
	}
	store, err := storage.New()
	if err != nil {
		t.Fatal(err)
	}
	manager := NewManager(store)
	profiles, err := manager.Profiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) == 0 {
		t.Fatal("no saved PuffSQL profiles")
	}
	selected := profiles[0]
	if wanted := os.Getenv("PUFFSQL_TEST_PROFILE"); wanted != "" {
		found := false
		for _, profile := range profiles {
			if profile.ID == wanted {
				selected = profile
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("profile %q was not found", wanted)
		}
	}
	if password := os.Getenv("PUFFSQL_TEST_PASSWORD"); password != "" {
		if err := manager.credentials.Set(selected.ID, password); err != nil {
			t.Fatalf("update test profile credential: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := manager.Connect(ctx, selected.ID); err != nil {
		t.Fatalf("connect %s: %v", selected.Name, err)
	}
	defer manager.Disconnect(selected.ID)

	requestID := "live-smoke"
	maxRows := 10
	timeout := 20
	result, err := manager.ExecuteQuery(
		ctx, selected.ID, nil,
		"SELECT DB_NAME() AS DatabaseName, CAST(1 AS int) AS Probe",
		&maxRows, &requestID, &timeout, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ResultSets) != 1 || len(result.ResultSets[0].Rows) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.ResultSets[0].Rows[0][1] != int64(1) {
		t.Fatalf("probe value = %#v", result.ResultSets[0].Rows[0][1])
	}
	if len(result.Messages) == 0 || !strings.Contains(result.Messages[len(result.Messages)-1].Message, "completed successfully") {
		t.Fatalf("success message missing: %#v", result.Messages)
	}

	multi, err := manager.ExecuteQuery(
		ctx, selected.ID, nil,
		"SELECT 1 AS Number;\nGO\nSELECT 2 AS Number;",
		&maxRows, nil, &timeout, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(multi.ResultSets) != 2 {
		t.Fatalf("GO batch returned %d result sets: %#v", len(multi.ResultSets), multi)
	}
	cancelled := make(chan struct {
		result model.QueryResult
		err    error
	}, 1)
	cancelRequest := "live-cancel"
	go func() {
		value, err := manager.ExecuteQuery(
			context.Background(), selected.ID, nil,
			"WAITFOR DELAY '00:00:10'; SELECT 1 AS TooLate;",
			nil, &cancelRequest, nil, nil,
		)
		cancelled <- struct {
			result model.QueryResult
			err    error
		}{result: value, err: err}
	}()
	time.Sleep(300 * time.Millisecond)
	if err := manager.CancelQuery(selected.ID, &cancelRequest); err != nil {
		t.Fatal(err)
	}
	select {
	case outcome := <-cancelled:
		if outcome.err != nil {
			t.Fatalf("cancelled query returned an error: %v", outcome.err)
		}
		found := false
		for _, message := range outcome.result.Messages {
			if message.Level == "warning" && strings.Contains(message.Message, "cancelled") {
				found = true
			}
		}
		if !found {
			t.Fatalf("cancelled query did not report a warning: %#v", outcome.result.Messages)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled query did not settle")
	}

	server, err := manager.Server(ctx, selected.ID)
	if err != nil {
		t.Fatal(err)
	}
	databases, err := server.GetDatabases(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(databases) == 0 {
		t.Fatal("SQL Server returned no visible databases")
	}
	tables, err := server.GetTables(ctx, &selected.Database, nil)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := server.GetDatabaseSchema(ctx, &selected.Database)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) > 0 {
		table := tables[0]
		columns, err := server.GetColumns(ctx, &selected.Database, table.Schema, table.Name)
		if err != nil {
			t.Fatalf("columns for %s.%s: %v", table.Schema, table.Name, err)
		}
		objects, err := server.GetTableObjects(ctx, &selected.Database, table.Schema, table.Name)
		if err != nil {
			t.Fatalf("objects for %s.%s: %v", table.Schema, table.Name, err)
		}
		if len(columns) == 0 || len(objects.Columns) == 0 {
			t.Fatalf("metadata was empty for %s.%s", table.Schema, table.Name)
		}
		if _, err := server.GetForeignKeys(ctx, &selected.Database, &table.Schema, &table.Name); err != nil {
			t.Fatalf("foreign keys for %s.%s: %v", table.Schema, table.Name, err)
		}
	}
	t.Logf(
		"connected to %s; database=%v; probe=%v; visible_databases=%d; tables=%d; schema_objects=%d; multi_sets=%d",
		selected.Host, result.ResultSets[0].Rows[0][0], result.ResultSets[0].Rows[0][1],
		len(databases), len(tables), len(schema), len(multi.ResultSets),
	)
}
