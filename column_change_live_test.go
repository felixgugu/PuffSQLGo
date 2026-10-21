package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"puffsql/internal/model"
)

func TestLiveColumnChangeLifecycle(t *testing.T) {
	if os.Getenv("PUFFSQL_LIVE") != "1" {
		t.Skip("set PUFFSQL_LIVE=1 to run schema changes against the test database")
	}
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
	var profile model.ConnectionProfile
	for _, candidate := range profiles {
		if candidate.ID == profileID {
			profile = candidate
			break
		}
	}
	if profile.ID == "" {
		t.Fatal("test profile not found")
	}
	if profile.Database != "TestDB1" {
		t.Fatalf("refusing schema test outside TestDB1: %s", profile.Database)
	}
	if err := app.Connect(profile.ID); err != nil {
		t.Fatal(err)
	}
	defer app.Disconnect(profile.ID)

	table := fmt.Sprintf("PuffSQL_ColumnChange_%d", time.Now().UnixNano())
	qualified := "[dbo].[" + table + "]"
	if _, err := app.ExecuteQuery(profile.ID, &profile.Database,
		"CREATE TABLE "+qualified+" ([Id] int NOT NULL); INSERT INTO "+qualified+" ([Id]) VALUES (1);",
		nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	defer app.ExecuteQuery(profile.ID, &profile.Database, "DROP TABLE IF EXISTS "+qualified, nil, nil, nil)

	length := 50
	addDefault := "N'seed'"
	addDescription := "顯示名稱"
	addReq := model.ColumnChangeRequest{
		ConnectionID: profile.ID, Database: profile.Database, Schema: "dbo", Table: table,
		Operation: "add", ColumnName: "Name", DataType: "nvarchar", Length: &length,
		IsNullable: false, DefaultExpression: &addDefault, Description: &addDescription,
	}
	publish(t, app, addReq)
	assertColumnDescription(t, app, profile, table, "Name", addDescription)
	dependencyPreview, err := app.PreviewColumnChange(model.ColumnChangeRequest{
		ConnectionID: profile.ID, Database: profile.Database, Schema: "dbo", Table: table,
		Operation: "drop", OriginalName: "Name", ColumnName: "Name", DataType: "nvarchar",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(dependencyPreview.BlockingIssues) == 0 || len(dependencyPreview.Dependencies) == 0 {
		t.Fatal("dropping a defaulted column must report its dependency")
	}

	newLength := 100
	alterReq := addReq
	alterReq.Operation = "alter"
	alterReq.OriginalName = "Name"
	alterReq.ColumnName = "DisplayName"
	alterReq.Length = &newLength
	alterReq.DefaultExpression = nil
	updatedDescription := "客戶顯示名稱"
	alterReq.Description = &updatedDescription
	publish(t, app, alterReq)
	assertColumnDescription(t, app, profile, table, "DisplayName", updatedDescription)

	clearDescription := alterReq
	clearDescription.OriginalName = "DisplayName"
	clearDescription.Description = nil
	publish(t, app, clearDescription)
	assertColumnDescription(t, app, profile, table, "DisplayName", "")

	dropReq := alterReq
	dropReq.Operation = "drop"
	dropReq.OriginalName = "DisplayName"
	dropReq.ColumnName = "DisplayName"
	preview, err := app.PreviewColumnChange(dropReq)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.BlockingIssues) != 0 {
		t.Fatalf("drop unexpectedly blocked: %#v", preview.BlockingIssues)
	}
	if _, err := app.ApplyColumnChange(dropReq, preview.Fingerprint); err != nil {
		t.Fatal(err)
	}

	columns, err := app.GetColumns(profile.ID, &profile.Database, "dbo", table)
	if err != nil {
		t.Fatal(err)
	}
	if len(columns) != 1 || columns[0].Name != "Id" {
		t.Fatalf("final columns = %#v", columns)
	}
}

func assertColumnDescription(
	t *testing.T,
	app *App,
	profile model.ConnectionProfile,
	table, column, want string,
) {
	t.Helper()
	columns, err := app.GetColumns(profile.ID, &profile.Database, "dbo", table)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range columns {
		if item.Name == column {
			got := ""
			if item.Description != nil {
				got = *item.Description
			}
			if got != want {
				t.Fatalf("description for %s = %q, want %q", column, got, want)
			}
			return
		}
	}
	t.Fatalf("column %s not found", column)
}

func publish(t *testing.T, app *App, req model.ColumnChangeRequest) {
	t.Helper()
	preview, err := app.PreviewColumnChange(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.BlockingIssues) != 0 {
		t.Fatalf("%s blocked: %#v", req.Operation, preview.BlockingIssues)
	}
	if _, err := app.ApplyColumnChange(req, preview.Fingerprint); err != nil {
		t.Fatal(err)
	}
}
