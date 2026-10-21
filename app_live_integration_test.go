package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveAppFacade exercises the same exported methods Wails invokes from Vue.
func TestLiveAppFacade(t *testing.T) {
	if os.Getenv("PUFFSQL_LIVE") != "1" {
		t.Skip("set PUFFSQL_LIVE=1 to run against the configured SQL Server")
	}
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	app.Startup(ctx)
	profiles, err := app.GetConnections()
	if err != nil || len(profiles) == 0 {
		t.Fatalf("profiles: %v, %v", profiles, err)
	}
	selected := profiles[0]
	if wanted := os.Getenv("PUFFSQL_TEST_PROFILE"); wanted != "" {
		found := false
		for _, profile := range profiles {
			if profile.ID == wanted {
				selected = profile
				found = true
			}
		}
		if !found {
			t.Fatalf("profile %q not found", wanted)
		}
	}
	if err := app.Connect(selected.ID); err != nil {
		t.Fatal(err)
	}
	defer app.Disconnect(selected.ID)

	database := selected.Database
	requestID := "app-live"
	maxRows := 5
	timeout := 15
	result, err := app.ExecuteQuery(
		selected.ID, &database,
		"SELECT DB_NAME() AS DatabaseName, SUSER_SNAME() AS LoginName;\nGO\nSELECT 2 AS Probe;",
		&maxRows, &requestID, &timeout,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ResultSets) != 2 || len(result.Messages) == 0 ||
		!strings.Contains(result.Messages[len(result.Messages)-1].Message, "Executed 2 batch") {
		t.Fatalf("App facade query result = %#v", result)
	}
	databases, err := app.GetDatabases(selected.ID)
	if err != nil || len(databases) == 0 {
		t.Fatalf("facade databases: %d, %v", len(databases), err)
	}
	tables, err := app.GetTables(selected.ID, &database, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) > 0 {
		table := tables[0]
		objects, err := app.GetTableObjects(selected.ID, &database, table.Schema, table.Name)
		if err != nil {
			t.Fatal(err)
		}
		if len(objects.Columns) == 0 {
			t.Fatalf("facade returned no columns for %s.%s", table.Schema, table.Name)
		}
	}
	t.Logf("App facade: database=%v login=%v sets=%d databases=%d tables=%d",
		result.ResultSets[0].Rows[0][0], result.ResultSets[0].Rows[0][1],
		len(result.ResultSets), len(databases), len(tables),
	)
}

func TestLiveExplorerLoadsEveryTable(t *testing.T) {
	if os.Getenv("PUFFSQL_LIVE") != "1" {
		t.Skip("set PUFFSQL_LIVE=1 to run against the configured SQL Server")
	}
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	app.Startup(ctx)
	profiles, err := app.GetConnections()
	if err != nil {
		t.Fatal(err)
	}
	profileID := os.Getenv("PUFFSQL_TEST_PROFILE")
	if profileID == "" {
		t.Fatal("PUFFSQL_TEST_PROFILE is required")
	}
	var database string
	for _, profile := range profiles {
		if profile.ID == profileID {
			database = profile.Database
			break
		}
	}
	if database != "TestDB1" {
		t.Fatalf("refusing broad Explorer test outside TestDB1: %s", database)
	}
	if err := app.Connect(profileID); err != nil {
		t.Fatal(err)
	}
	defer app.Disconnect(profileID)
	tables, err := app.GetTables(profileID, &database, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range tables {
		started := time.Now()
		objects, err := app.GetTableObjects(profileID, &database, table.Schema, table.Name)
		if err != nil {
			t.Fatalf("%s.%s: %v", table.Schema, table.Name, err)
		}
		if len(objects.Columns) == 0 {
			t.Fatalf("%s.%s returned no columns", table.Schema, table.Name)
		}
		if elapsed := time.Since(started); elapsed > 20*time.Second {
			t.Fatalf("%s.%s exceeded metadata timeout: %v", table.Schema, table.Name, elapsed)
		}
	}
	t.Logf("loaded metadata for %d tables under the bounded App facade", len(tables))
}
