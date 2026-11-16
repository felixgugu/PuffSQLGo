package mysql

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	driver "github.com/go-sql-driver/mysql"
	"puffsql/internal/dbengine"
)

func dbengineErrorText() string {
	return dbengine.ErrUnsupported.Error()
}

func TestExecuteRunsStatementsOneByOneAndTracksUse(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := &Server{DB: db, database: "shop"}

	mock.ExpectQuery(`SELECT CONNECTION_ID\(\)`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(77))
	mock.ExpectExec("USE `shop`").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT id FROM orders").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(1)).AddRow(int64(2)).AddRow(int64(3)))
	mock.ExpectQuery("USE inventory").WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectExec("ROLLBACK").WillReturnResult(sqlmock.NewResult(0, 0))

	maxRows := 2
	session := 0
	result := server.Execute(
		context.Background(), nil, "SELECT id FROM orders;\nUSE inventory;", &maxRows, nil,
		func(id int) { session = id }, nil,
	)

	if session != 77 {
		t.Fatalf("session id = %d", session)
	}
	if len(result.ResultSets) != 1 || result.ResultSets[0].RowCount != 2 || !result.ResultSets[0].IsTruncated {
		t.Fatalf("result sets = %#v", result.ResultSets)
	}
	if server.Database() != "inventory" {
		t.Fatalf("database after USE = %q", server.Database())
	}
	if last := result.Messages[len(result.Messages)-1]; last.Level != "info" {
		t.Fatalf("last message = %#v", last)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteStopsAtFirstErrorAndRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := &Server{DB: db, database: "shop"}

	mock.ExpectQuery(`SELECT CONNECTION_ID\(\)`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))
	mock.ExpectExec("USE `shop`").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("START TRANSACTION").WillReturnRows(sqlmock.NewRows(nil))
	mock.ExpectQuery("UPDATE broken").
		WillReturnError(&driver.MySQLError{Number: 1146, Message: "Table 'shop.broken' doesn't exist"})
	mock.ExpectExec("ROLLBACK").WillReturnResult(sqlmock.NewResult(0, 0))

	result := server.Execute(
		context.Background(), nil,
		"START TRANSACTION;\n\nUPDATE broken SET a = 1;\nCOMMIT;", nil, nil, nil, nil,
	)

	if len(result.Messages) != 1 {
		t.Fatalf("messages = %#v", result.Messages)
	}
	message := result.Messages[0]
	if message.Level != "error" || message.Code == nil || *message.Code != 1146 {
		t.Fatalf("error message = %#v", message)
	}
	if message.Message != "Table 'shop.broken' doesn't exist" {
		t.Fatalf("error text = %q", message.Message)
	}
	if message.LineNumber == nil || *message.LineNumber != 3 {
		t.Fatalf("error line = %v", message.LineNumber)
	}
	// COMMIT must never run after the failure, and the open transaction must be rolled back.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteContinuesWithoutDefaultDatabaseWhenItWasDropped(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := &Server{DB: db, database: "gone"}

	mock.ExpectQuery(`SELECT CONNECTION_ID\(\)`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))
	mock.ExpectExec("USE `gone`").
		WillReturnError(&driver.MySQLError{Number: 1049, Message: "Unknown database 'gone'"})
	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(int64(1)))
	mock.ExpectExec("ROLLBACK").WillReturnResult(sqlmock.NewResult(0, 0))

	result := server.Execute(context.Background(), nil, "SELECT 1", nil, nil, nil, nil)

	if len(result.ResultSets) != 1 {
		t.Fatalf("result sets = %#v", result.ResultSets)
	}
	if result.Messages[0].Level != "warning" || server.Database() != "" {
		t.Fatalf("messages = %#v, database = %q", result.Messages, server.Database())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
