package connection

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"puffsql/internal/model"
	"puffsql/internal/storage"
)

// Exercises the manager against a real MySQL server (see internal/mysql/live_integration_test.go
// for the environment variables and the puff_shop fixture).
func TestLiveMySQLThroughManager(t *testing.T) {
	if os.Getenv("PUFFSQL_LIVE_MYSQL") != "1" {
		t.Skip("set PUFFSQL_LIVE_MYSQL=1 to run MySQL live tests")
	}
	host := os.Getenv("PUFFSQL_MYSQL_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port, _ := strconv.Atoi(os.Getenv("PUFFSQL_MYSQL_PORT"))
	password := os.Getenv("PUFFSQL_MYSQL_PASSWORD")

	manager := NewManager(storage.NewAt(t.TempDir()), nil)
	manager.credentials = &fakeCredentials{values: map[string]string{}}
	request := model.SaveConnectionRequest{
		ConnectionConfig: model.ConnectionConfig{
			Name: "Live MySQL", Host: host, Port: port, Username: os.Getenv("PUFFSQL_MYSQL_USER"),
		},
		Engine:   "mysql",
		Password: &password,
	}
	if err := manager.TestConnection(request); err != nil {
		t.Fatalf("test connection: %v", err)
	}
	wrong := "definitely-wrong"
	bad := request
	bad.Password = &wrong
	if err := manager.TestConnection(bad); err == nil {
		t.Fatal("a wrong password must fail the connection test")
	}

	profile, err := manager.SaveProfile(request)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Engine != "mysql" || profile.Port != 3306 || profile.Database != "" {
		t.Fatalf("saved profile = %#v", profile)
	}
	ctx := context.Background()
	if err := manager.Connect(ctx, profile.ID); err != nil {
		t.Fatal(err)
	}
	defer manager.Disconnect(profile.ID)

	database := "puff_shop"
	run := func(statement string) model.QueryResult {
		t.Helper()
		result, err := manager.ExecuteQuery(ctx, profile.ID, &database, statement, nil, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, message := range result.Messages {
			if message.Level == "error" {
				t.Fatalf("%s\n-- failed: %s", statement, message.Message)
			}
		}
		return result
	}

	// The statements below are the shapes the frontend MySQL dialect generates.
	limited := run("SELECT * FROM `puff_shop`.`order items` LIMIT 500;")
	if limited.ResultSets[0].RowCount != 2 {
		t.Fatalf("data viewer rows = %d", limited.ResultSets[0].RowCount)
	}
	if spid := manager.SPID(profile.ID); spid == nil || *spid <= 0 {
		t.Fatalf("session id not recorded: %v", spid)
	}

	routines := run("SELECT ROUTINE_SCHEMA, ROUTINE_NAME, ROUTINE_TYPE FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = 'puff_shop' ORDER BY ROUTINE_NAME;")
	if len(routines.ResultSets) != 1 {
		t.Fatalf("routine list = %#v", routines.ResultSets)
	}

	for kind, name := range map[string]string{"TABLE": "order items", "VIEW": "v_customer_totals"} {
		definition := run("SHOW CREATE " + kind + " `puff_shop`.`" + name + "`;").ResultSets[0]
		column := -1
		for i, def := range definition.Columns {
			if strings.HasPrefix(strings.ToLower(def.Name), "create ") {
				column = i
			}
		}
		if column < 0 {
			t.Fatalf("SHOW CREATE %s has no 'Create ...' column: %#v", kind, definition.Columns)
		}
		if text, _ := definition.Rows[0][column].(string); !strings.Contains(text, "CREATE") {
			t.Fatalf("SHOW CREATE %s = %#v", kind, definition.Rows[0][column])
		}
	}

	// Grid commit: DELETE -> INSERT -> UPDATE in one transaction, keyed statements capped by
	// LIMIT 1, a BIGINT key beyond 2^53 and a DECIMAL written as bare numbers.
	run("START TRANSACTION;\n" +
		"  -- [刪除 1] 依主鍵刪除 1 列\n" +
		"  DELETE FROM `puff_shop`.`order items`\n  WHERE `order_id` = 2\n  AND `line_no` = 1\n  LIMIT 1;\n\n" +
		"  -- [新增 1] 寫入 4 個欄位\n" +
		"  INSERT INTO `puff_shop`.`order items` (`order_id`, `line_no`, `customer_id`, `note`)\n  VALUES (77, 1, 1, 'it''s a C:\\\\path');\n\n" +
		"  -- [列 1] 更新 2 個欄位\n" +
		"  UPDATE `puff_shop`.`order items`\n  SET `amount` = 0.0000000001, `note` = NULL\n  WHERE `order_id` = 9007199254740993\n  AND `line_no` = 1\n  LIMIT 1;\n\n" +
		"COMMIT;")
	after := run("SELECT order_id, amount, note FROM `order items` ORDER BY order_id").ResultSets[0]
	if after.RowCount != 2 {
		t.Fatalf("rows after commit = %#v", after.Rows)
	}
	if after.Rows[0][0] != int64(77) || after.Rows[0][2] != `it's a C:\path` {
		t.Fatalf("inserted row = %#v", after.Rows[0])
	}
	if after.Rows[1][0] != "9007199254740993" || after.Rows[1][1] != "0.0000000001" || after.Rows[1][2] != nil {
		t.Fatalf("updated row = %#v", after.Rows[1])
	}
	// Restore the fixture for the next run.
	run("DELETE FROM `order items` WHERE order_id = 77;\n" +
		"UPDATE `order items` SET amount = 1234567890123456789012345678.0123456789, note = 'auto' WHERE order_id = 9007199254740993;\n" +
		"INSERT INTO `order items` (order_id, line_no, customer_id, amount, qty) VALUES (2, 1, 2, 12.50, 1);")

	// Cancel through the manager: context cancel, pool drop and KILL QUERY on a side connection.
	requestID := "live-cancel"
	done := make(chan model.QueryResult, 1)
	ready := make(chan struct{}, 1)
	started := time.Now()
	go func() {
		result, _ := manager.ExecuteQuery(ctx, profile.ID, &database, "SELECT SLEEP(30)", nil, &requestID, nil,
			func(int, string) { ready <- struct{}{} })
		done <- result
	}()
	<-ready
	time.Sleep(500 * time.Millisecond)
	if err := manager.CancelQuery(profile.ID, &requestID); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-done:
		if time.Since(started) > 10*time.Second {
			t.Fatalf("cancel took %s", time.Since(started))
		}
		if len(result.Messages) == 0 || !strings.Contains(result.Messages[len(result.Messages)-1].Message, "cancelled") {
			t.Fatalf("messages after cancel = %#v", result.Messages)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("query was not cancelled")
	}
	// The pool was dropped by the cancel; the next query must reconnect on its own, and the
	// cancelled statement must be gone from the server.
	leftover := run("SELECT COUNT(*) FROM information_schema.PROCESSLIST WHERE INFO = 'SELECT SLEEP(30)'").ResultSets[0]
	if leftover.Rows[0][0] != int64(0) {
		t.Fatalf("cancelled statement still running: %#v", leftover.Rows[0][0])
	}
}
