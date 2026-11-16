package mysql

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"puffsql/internal/model"
)

// Live tests run against a real MySQL server and are skipped unless PUFFSQL_LIVE_MYSQL=1.
// They expect the fixture database `puff_shop`; load it from testdata/puff_shop.sql.
//
//	PUFFSQL_MYSQL_HOST (default 127.0.0.1)   PUFFSQL_MYSQL_PORT (default 3306)
//	PUFFSQL_MYSQL_USER                       PUFFSQL_MYSQL_PASSWORD
func openLive(t *testing.T) *Server {
	t.Helper()
	if os.Getenv("PUFFSQL_LIVE_MYSQL") != "1" {
		t.Skip("set PUFFSQL_LIVE_MYSQL=1 to run MySQL live tests")
	}
	port, _ := strconv.Atoi(os.Getenv("PUFFSQL_MYSQL_PORT"))
	host := os.Getenv("PUFFSQL_MYSQL_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	profile := model.ConnectionProfile{
		ConnectionConfig: model.ConnectionConfig{
			Host: host, Port: port, Username: os.Getenv("PUFFSQL_MYSQL_USER"), Database: "puff_shop",
		},
		Engine: "mysql",
	}
	server, err := Open(profile, os.Getenv("PUFFSQL_MYSQL_PASSWORD"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return server
}

func errorsOf(result model.QueryResult) []string {
	var messages []string
	for _, message := range result.Messages {
		if message.Level == "error" {
			messages = append(messages, message.Message)
		}
	}
	return messages
}

func TestLiveMetadata(t *testing.T) {
	server := openLive(t)
	ctx := context.Background()
	db := "puff_shop"

	databases, err := server.GetDatabases(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range databases {
		found = found || item.Name == db
	}
	if !found {
		t.Fatalf("puff_shop missing from %v", databases)
	}

	tables, err := server.GetTables(ctx, &db, nil)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]string{}
	for _, table := range tables {
		if table.Schema != db {
			t.Fatalf("schema must be the database name: %#v", table)
		}
		kinds[table.Name] = table.Kind
	}
	if kinds["customers"] != "BASE TABLE" || kinds["order items"] != "BASE TABLE" || kinds["v_customer_totals"] != "VIEW" {
		t.Fatalf("tables = %v", kinds)
	}

	objects, err := server.GetTableObjects(ctx, &db, db, "order items")
	if err != nil {
		t.Fatal(err)
	}
	columns := map[string]model.ColumnItem{}
	for _, column := range objects.Columns {
		columns[column.Name] = column
	}
	if !columns["order_id"].IsPrimaryKey || !columns["line_no"].IsPrimaryKey {
		t.Fatalf("composite primary key not detected: %#v", objects.Columns)
	}
	total := columns["total"]
	if !total.IsComputed || !total.IsPersisted || total.ComputedExpression == nil {
		t.Fatalf("generated column = %#v", total)
	}
	if amount := columns["amount"]; amount.DataType != "decimal" || amount.Precision == nil || *amount.Precision != 38 ||
		amount.Scale == nil || *amount.Scale != 10 {
		t.Fatalf("decimal column = %#v", amount)
	}
	constraintKinds := map[string]string{}
	for _, constraint := range objects.Constraints {
		constraintKinds[constraint.Name] = constraint.Kind
		if constraint.Kind == "FOREIGN_KEY" {
			if constraint.RefTable == nil || *constraint.RefTable != "customers" ||
				len(constraint.RefColumns) != 1 || constraint.RefColumns[0] != "id" ||
				constraint.OnDelete == nil || *constraint.OnDelete != "CASCADE" {
				t.Fatalf("foreign key = %#v", constraint)
			}
		}
	}
	if constraintKinds["PRIMARY"] != "PRIMARY_KEY" || constraintKinds["fk_items_customer"] != "FOREIGN_KEY" ||
		constraintKinds["ck_items_qty"] != "CHECK" {
		t.Fatalf("constraints = %v", constraintKinds)
	}
	var customerIndex *model.TableIndexItem
	for i := range objects.Indexes {
		if objects.Indexes[i].Name == "ix_items_customer" {
			customerIndex = &objects.Indexes[i]
		}
	}
	if customerIndex == nil || len(customerIndex.Columns) != 2 || !customerIndex.Columns[1].Descending || customerIndex.IsUnique {
		t.Fatalf("index = %#v", customerIndex)
	}
	if len(objects.Triggers) != 1 || objects.Triggers[0].Name != "trg_items_bi" || objects.Triggers[0].Events[0] != "BEFORE INSERT" {
		t.Fatalf("triggers = %#v", objects.Triggers)
	}

	customers, err := server.GetColumns(ctx, &db, db, "customers")
	if err != nil {
		t.Fatal(err)
	}
	if len(customers) != 5 || !customers[0].IsIdentity || customers[0].Description == nil || *customers[0].Description != "客戶編號" {
		t.Fatalf("customers columns = %#v", customers)
	}
	if customers[3].Scale == nil || *customers[3].Scale != 3 {
		t.Fatalf("datetime(3) precision = %#v", customers[3])
	}

	schema, err := server.GetDatabaseSchema(ctx, &db)
	if err != nil {
		t.Fatal(err)
	}
	if len(schema) != 3 {
		t.Fatalf("database schema = %#v", schema)
	}
	for _, table := range schema {
		if len(table.Columns) == 0 {
			t.Fatalf("table %s has no columns", table.Name)
		}
	}

	keys, err := server.GetForeignKeys(ctx, &db, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].FromTable != "order items" || keys[0].ToTable != "customers" || keys[0].ToSchema != db {
		t.Fatalf("foreign keys = %#v", keys)
	}
}

func TestLiveExecuteKeepsValuesExact(t *testing.T) {
	server := openLive(t)
	result := server.Execute(context.Background(), nil,
		"SELECT order_id, amount, total, flag, ratio, note, payload, shipped FROM `order items` ORDER BY order_id DESC;\n"+
			"SELECT ref_no, created_at FROM customers WHERE id = 2;\n"+
			"SELECT CAST('0000-00-00' AS CHAR) AS zero, NULL AS nothing, 1 = 1 AS truth",
		nil, nil, nil, nil)
	if failures := errorsOf(result); len(failures) > 0 {
		t.Fatal(failures)
	}
	if len(result.ResultSets) != 3 {
		t.Fatalf("result sets = %d", len(result.ResultSets))
	}
	row := result.ResultSets[0].Rows[0]
	if row[0] != "9007199254740993" {
		t.Fatalf("bigint beyond 2^53 = %#v", row[0])
	}
	if row[1] != "1234567890123456789012345678.0123456789" || row[2] != "2469135780246913578024691356.0246913578" {
		t.Fatalf("decimal(38,10) = %#v / %#v", row[1], row[2])
	}
	if row[3] != int64(1) || row[4] != 1.1 || row[5] != "auto" || row[7] != "2026-01-02" {
		t.Fatalf("bit/float/text/date = %#v", row)
	}
	if binary, ok := row[6].(model.BinaryCell); !ok || binary.Length != 3 {
		t.Fatalf("blob = %#v", row[6])
	}
	if id := result.ResultSets[1].Rows[0][0]; id != "18446744073709551615" {
		t.Fatalf("max unsigned bigint = %#v", id)
	}
	if created, ok := result.ResultSets[1].Rows[0][1].(string); !ok || len(created) != len("2026-01-02 03:04:05.678") {
		t.Fatalf("datetime(3) = %#v", result.ResultSets[1].Rows[0][1])
	}
	last := result.ResultSets[2].Rows[0]
	if last[0] != "0000-00-00" || last[1] != nil || last[2] != int64(1) {
		t.Fatalf("misc row = %#v", last)
	}
	types := []string{}
	for _, column := range result.ResultSets[0].Columns {
		types = append(types, column.DataType)
	}
	if got := strings.Join(types, ","); got != "BIGINT,DECIMAL,DECIMAL,BIT,FLOAT,TEXT,BLOB,DATE" {
		t.Fatalf("column types = %s", got)
	}
}

func TestLiveDelimiterProcedureAndMultipleResultSets(t *testing.T) {
	server := openLive(t)
	script := "DROP PROCEDURE IF EXISTS puff_two_sets;\n" +
		"DELIMITER $$\n" +
		"CREATE PROCEDURE puff_two_sets()\nBEGIN\n  SELECT 1 AS a;\n  SELECT 2 AS b, 3 AS c;\nEND$$\n" +
		"DELIMITER ;\n" +
		"CALL puff_two_sets();\n" +
		"DROP PROCEDURE puff_two_sets;"
	result := server.Execute(context.Background(), nil, script, nil, nil, nil, nil)
	if failures := errorsOf(result); len(failures) > 0 {
		t.Fatal(failures)
	}
	if len(result.ResultSets) != 2 || len(result.ResultSets[1].Columns) != 2 || result.ResultSets[1].Rows[0][1] != int64(3) {
		t.Fatalf("result sets = %#v", result.ResultSets)
	}
}

func TestLiveFailedScriptLeavesNoOpenTransaction(t *testing.T) {
	server := openLive(t)
	ctx := context.Background()
	server.DB.SetMaxOpenConns(1) // force every statement below onto the same pooled connection

	result := server.Execute(ctx, nil,
		"START TRANSACTION;\nUPDATE customers SET name = 'changed' WHERE id = 1;\nUPDATE missing_table SET a = 1;\nCOMMIT;",
		nil, nil, nil, nil)
	failures := errorsOf(result)
	if len(failures) != 1 {
		t.Fatalf("messages = %#v", result.Messages)
	}
	for _, message := range result.Messages {
		if message.Level == "error" && (message.Code == nil || *message.Code != 1146 || message.LineNumber == nil || *message.LineNumber != 3) {
			t.Fatalf("error = %#v", message)
		}
	}

	check := server.Execute(ctx, nil,
		"SELECT name FROM customers WHERE id = 1;\nSELECT COUNT(*) FROM information_schema.INNODB_TRX WHERE trx_mysql_thread_id = CONNECTION_ID()",
		nil, nil, nil, nil)
	if failures := errorsOf(check); len(failures) > 0 {
		t.Fatal(failures)
	}
	if name := check.ResultSets[0].Rows[0][0]; name != "O'Brien" {
		t.Fatalf("update was not rolled back: %#v", name)
	}
	if open := check.ResultSets[1].Rows[0][0]; open != int64(0) {
		t.Fatalf("transaction still open on the pooled connection: %#v", open)
	}
}

func TestLiveCancelWithKillQuery(t *testing.T) {
	server := openLive(t)
	killer := openLive(t)
	session := make(chan int, 1)
	done := make(chan model.QueryResult, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := time.Now()
	go func() {
		done <- server.Execute(ctx, nil, "SELECT SLEEP(30) AS slept", nil, nil, func(id int) { session <- id }, nil)
	}()
	id := <-session
	time.Sleep(500 * time.Millisecond)
	// Same order as connection.Manager.CancelQuery: cancel the context, then kill the session.
	cancel()
	killCtx, killCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer killCancel()
	if err := killer.KillSession(killCtx, id); err != nil {
		t.Logf("KILL QUERY after cancel: %v", err)
	}

	select {
	case result := <-done:
		if elapsed := time.Since(started); elapsed > 10*time.Second {
			t.Fatalf("cancel took %s", elapsed)
		}
		if len(result.Messages) == 0 || result.Messages[len(result.Messages)-1].Level != "warning" {
			t.Fatalf("messages = %#v", result.Messages)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("query was not cancelled")
	}

	// KILL QUERY only flags the session; the server ends the statement a moment later.
	running := -1
	for deadline := time.Now().Add(3 * time.Second); running != 0 && time.Now().Before(deadline); {
		if err := killer.DB.QueryRow(
			"SELECT COUNT(*) FROM information_schema.PROCESSLIST WHERE INFO LIKE 'SELECT SLEEP(30)%'").Scan(&running); err != nil {
			t.Fatal(err)
		}
		if running != 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if running != 0 {
		t.Fatalf("%d cancelled statement(s) still running on the server", running)
	}
}

func TestLiveSwitchDatabaseAndDroppedDatabase(t *testing.T) {
	server := openLive(t)
	ctx := context.Background()
	if err := server.SwitchDatabase(ctx, "no_such_database_here"); err == nil {
		t.Fatal("switching to a missing database must fail")
	}
	setup := server.Execute(ctx, nil, "CREATE DATABASE IF NOT EXISTS puff_tmp;\nUSE puff_tmp;", nil, nil, nil, nil)
	if failures := errorsOf(setup); len(failures) > 0 {
		t.Fatal(failures)
	}
	if server.Database() != "puff_tmp" {
		t.Fatalf("database after USE = %q", server.Database())
	}
	drop := server.Execute(ctx, nil, "DROP DATABASE puff_tmp;", nil, nil, nil, nil)
	if failures := errorsOf(drop); len(failures) > 0 {
		t.Fatal(failures)
	}
	after := server.Execute(ctx, nil, "SELECT 1", nil, nil, nil, nil)
	if failures := errorsOf(after); len(failures) > 0 || len(after.ResultSets) != 1 {
		t.Fatalf("query after dropping the selected database: %#v", after.Messages)
	}
	if after.Messages[0].Level != "warning" {
		t.Fatalf("expected a dropped-database warning: %#v", after.Messages)
	}
}

func TestLiveExcelExports(t *testing.T) {
	server := openLive(t)
	ctx := context.Background()
	dir := t.TempDir()

	schema, err := server.ExportDatabaseSchemaExcel(ctx, "puff_shop", filepath.Join(dir, "schema.xlsx"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if schema.TableCount != 2 || schema.ColumnCount != 16 {
		t.Fatalf("schema export = %#v", schema)
	}
	data, err := server.ExportDatabaseTablesExcel(ctx, "puff_shop",
		[]model.ExcelTableRef{{Schema: "puff_shop", Name: "order items"}, {Schema: "puff_shop", Name: "customers"}},
		filepath.Join(dir, "data.xlsx"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if data.TableCount != 2 || data.TotalRows != 4 {
		t.Fatalf("data export = %#v", data)
	}
	for _, name := range []string{"schema.xlsx", "data.xlsx"} {
		if info, err := os.Stat(filepath.Join(dir, name)); err != nil || info.Size() == 0 {
			t.Fatalf("%s was not written: %v", name, err)
		}
	}
}
