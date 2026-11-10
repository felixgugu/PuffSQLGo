package mssql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestExecuteStreamsMultipleSetsAndAppliesRowLimit(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := &Server{DB: db, database: "master"}
	mock.ExpectQuery("SELECT @@SPID").WillReturnRows(sqlmock.NewRows([]string{"spid"}).AddRow(77))
	mock.ExpectExec("USE \\[master\\]").WillReturnResult(sqlmock.NewResult(0, 0))
	firstRows := sqlmock.NewRows([]string{"Id", "Name"}).
		AddRow(1, "Ada").
		AddRow(2, "Grace")
	secondRows := sqlmock.NewRows([]string{"Id", "Name"}).AddRow(10, "second")
	mock.ExpectQuery("SELECT 1").WillReturnRows(firstRows, secondRows)

	var observedSPID int
	maxRows := 1
	result := server.Execute(context.Background(), nil, "SELECT 1", &maxRows, nil, func(spid int) {
		observedSPID = spid
	}, nil)
	if observedSPID != 77 {
		t.Fatalf("SPID callback = %d, want 77", observedSPID)
	}
	if len(result.ResultSets) != 2 {
		t.Fatalf("result set count = %d, want 2", len(result.ResultSets))
	}
	first := result.ResultSets[0]
	if !first.IsTruncated || first.TotalCount != 2 || first.RowCount != 1 || first.Rows[0][1] != "Ada" {
		t.Fatalf("first set was not limited correctly: %#v", first)
	}
	if result.ResultSets[1].Rows[0][1] != "second" {
		t.Fatalf("second result set missing: %#v", result.ResultSets[1])
	}
	if result.AffectedRows != 2 || len(result.Messages) == 0 {
		t.Fatalf("result summary is incomplete: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteReportsCancellationWithoutThrowingAwayPartialResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := &Server{DB: db, database: "master"}
	mock.ExpectQuery("SELECT @@SPID").WillReturnRows(sqlmock.NewRows([]string{"spid"}).AddRow(21))
	mock.ExpectExec("USE \\[master\\]").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT 1").WillReturnError(context.Canceled)

	result := server.Execute(context.Background(), nil, "SELECT 1", nil, nil, nil, nil)
	if len(result.Messages) != 1 || result.Messages[0].Level != "warning" {
		t.Fatalf("cancellation produced %#v", result.Messages)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteReportsDriverFailureAsQueryMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := &Server{DB: db}
	mock.ExpectQuery("SELECT @@SPID").WillReturnRows(sqlmock.NewRows([]string{"spid"}).AddRow(21))
	mock.ExpectQuery("BROKEN").WillReturnError(errors.New("synthetic SQL failure"))

	result := server.Execute(context.Background(), nil, "BROKEN", nil, nil, nil, nil)
	if len(result.Messages) != 1 || result.Messages[0].Level != "error" {
		t.Fatalf("failure produced %#v", result.Messages)
	}
	if result.Messages[0].LineNumber == nil || *result.Messages[0].LineNumber != 1 {
		t.Fatalf("script line number was not preserved: %#v", result.Messages[0])
	}
}

func TestExecuteDropsDatabaseFromMasterContext(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := &Server{DB: db, database: "OldDb"}
	mock.ExpectQuery("SELECT @@SPID").WillReturnRows(sqlmock.NewRows([]string{"spid"}).AddRow(41))
	mock.ExpectExec("USE \\[OldDb\\]").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("USE master").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("DROP DATABASE \\[OldDb\\]").WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"One"}).AddRow(1))

	var phases []string
	result := server.Execute(
		context.Background(), nil,
		"DROP DATABASE [OldDb];\nGO\nUSE [Other];\nSELECT 1;",
		nil, nil, nil,
		func(name string) { phases = append(phases, name) },
	)
	if len(result.Messages) == 0 || result.Messages[0].Level == "error" {
		t.Fatalf("drop produced errors: %#v", result.Messages)
	}
	if server.Database() != "Other" {
		t.Fatalf("database after batches = %q, want Other", server.Database())
	}
	found := false
	for _, name := range phases {
		if name == "drop_database" {
			found = true
		}
	}
	if !found {
		t.Fatalf("drop phase was not reported: %#v", phases)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteDropsAlreadyMissingDatabaseThroughMaster(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := &Server{DB: db, database: "OldDb"}
	mock.ExpectQuery("SELECT @@SPID").WillReturnRows(sqlmock.NewRows([]string{"spid"}).AddRow(42))
	mock.ExpectExec("USE \\[OldDb\\]").WillReturnError(errors.New("database does not exist"))
	mock.ExpectExec("USE master").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("USE master").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("DROP DATABASE IF EXISTS \\[OldDb\\]").WillReturnRows(sqlmock.NewRows(nil))

	result := server.Execute(
		context.Background(), nil,
		"DROP DATABASE IF EXISTS [OldDb];",
		nil, nil, nil, nil,
	)
	if len(result.Messages) == 0 || result.Messages[0].Level == "error" {
		t.Fatalf("missing-database drop produced errors: %#v", result.Messages)
	}
	if server.Database() != "master" {
		t.Fatalf("database after fallback = %q, want master", server.Database())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeValueKeepsBinaryCompactAndPreservesNull(t *testing.T) {
	if value := normalizeValue([]byte{1, 2, 3}, "VARBINARY"); value == nil {
		t.Fatal("expected binary marker")
	}
	if value := normalizeValue([]byte("12.50"), "DECIMAL"); value != "12.50" {
		t.Fatalf("decimal = %#v", value)
	}
	if value := normalizeValue(nil, "INT"); value != nil {
		t.Fatalf("NULL became %#v", value)
	}
}

func TestNormalizeValueKeepsDateTimeOffsetZone(t *testing.T) {
	taipei := time.FixedZone("", 8*60*60)
	stamp := time.Date(2026, 9, 21, 8, 30, 0, 0, taipei)

	if value := normalizeValue(stamp, "DATETIMEOFFSET"); value != "2026-09-21 08:30:00 +08:00" {
		t.Fatalf("datetimeoffset = %#v", value)
	}
	if value := normalizeValue(stamp, "datetimeoffset(3)"); value != "2026-09-21 08:30:00 +08:00" {
		t.Fatalf("datetimeoffset(3) = %#v", value)
	}
	if value := normalizeValue(stamp.UTC(), "DATETIMEOFFSET"); value != "2026-09-21 00:30:00 +00:00" {
		t.Fatalf("datetimeoffset UTC = %#v", value)
	}
	if value := normalizeValue(stamp, "datetime2"); value != "2026-09-21 08:30:00" {
		t.Fatalf("datetime2 = %#v", value)
	}
}

func TestReadResultSetPreservesDateTimeOffsetZone(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	taipei := time.FixedZone("", 8*60*60)
	stamp := time.Date(2026, 11, 5, 10, 0, 0, 0, taipei)
	rows := sqlmock.NewRowsWithColumnDefinition(
		sqlmock.NewColumn("Plus8").OfType("DATETIMEOFFSET", time.Time{}).Nullable(true),
		sqlmock.NewColumn("Plain").OfType("DATETIME2", time.Time{}).Nullable(true),
	).AddRow(stamp, stamp)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	sqlRows, err := db.Query("SELECT Plus8, Plain")
	if err != nil {
		t.Fatal(err)
	}
	defer sqlRows.Close()
	names, err := sqlRows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	set, err := readResultSet(sqlRows, names, nil)
	if err != nil {
		t.Fatal(err)
	}
	if set.Columns[0].DataType != "DATETIMEOFFSET" {
		t.Fatalf("column type = %q", set.Columns[0].DataType)
	}
	if got := set.Rows[0][0]; got != "2026-11-05 10:00:00 +08:00" {
		t.Fatalf("datetimeoffset cell = %#v", got)
	}
	if got := set.Rows[0][1]; got != "2026-11-05 10:00:00" {
		t.Fatalf("datetime2 cell = %#v", got)
	}
}

func TestNormalizeValueFormatsUniqueIdentifier(t *testing.T) {
	// SQL Server wire order for FF19966F-868B-11D0-B42D-00C04FC964FF.
	raw := []byte{
		0x6F, 0x96, 0x19, 0xFF,
		0x8B, 0x86,
		0xD0, 0x11,
		0xB4, 0x2D,
		0x00, 0xC0, 0x4F, 0xC9, 0x64, 0xFF,
	}

	value := normalizeValue(raw, "uniqueidentifier")
	if value != "FF19966F-868B-11D0-B42D-00C04FC964FF" {
		t.Fatalf("uniqueidentifier = %#v", value)
	}
	if raw[0] != 0x6F {
		t.Fatal("normalization mutated the scanned value")
	}
}
