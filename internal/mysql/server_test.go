package mysql

import (
	"math"
	"strings"
	"testing"

	driver "github.com/go-sql-driver/mysql"
	"puffsql/internal/model"
)

func testProfile() model.ConnectionProfile {
	return model.ConnectionProfile{
		ConnectionConfig: model.ConnectionConfig{
			Host: "db.example.test", Database: "shop", Username: "app",
		},
		Engine: "mysql",
	}
}

func TestDSNUsesDefaultPortAndNeverEnablesMultiStatements(t *testing.T) {
	dsn, err := DSN(testProfile(), "p@ss:w/rd")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := driver.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Addr != "db.example.test:3306" || parsed.DBName != "shop" {
		t.Fatalf("addr/db = %q / %q", parsed.Addr, parsed.DBName)
	}
	if parsed.User != "app" || parsed.Passwd != "p@ss:w/rd" {
		t.Fatalf("credentials did not round-trip: %q / %q", parsed.User, parsed.Passwd)
	}
	if parsed.MultiStatements || parsed.ParseTime {
		t.Fatalf("multiStatements=%v parseTime=%v, want both off", parsed.MultiStatements, parsed.ParseTime)
	}
}

func TestDSNRejectsEmptyHost(t *testing.T) {
	profile := testProfile()
	profile.Host = " "
	if _, err := DSN(profile, ""); err == nil {
		t.Fatal("expected an error for an empty host")
	}
}

func TestTLSModeFollowsProfileSwitches(t *testing.T) {
	profile := testProfile()
	if got := tlsMode(profile); got != "false" {
		t.Fatalf("plain = %q", got)
	}
	profile.Encrypt = true
	if got := tlsMode(profile); got != "true" {
		t.Fatalf("encrypt = %q", got)
	}
	profile.TrustServerCertificate = true
	if got := tlsMode(profile); got != "skip-verify" {
		t.Fatalf("encrypt+trust = %q", got)
	}
}

func TestQuoteIdentifierDoublesBackticks(t *testing.T) {
	if got := QuoteIdentifier("we`ird"); got != "`we``ird`" {
		t.Fatalf("quoted = %s", got)
	}
	if got := QualifiedTable("shop", "order items"); got != "`shop`.`order items`" {
		t.Fatalf("qualified = %s", got)
	}
}

func TestNormalizeValueKeepsMySQLValuesExact(t *testing.T) {
	if value := normalizeValue(nil, "INT"); value != nil {
		t.Fatalf("NULL became %#v", value)
	}
	if value := normalizeValue(int64(42), "INT"); value != int64(42) {
		t.Fatalf("int = %#v", value)
	}
	if value := normalizeValue(int64(9007199254740993), "BIGINT"); value != "9007199254740993" {
		t.Fatalf("large bigint = %#v", value)
	}
	if value := normalizeValue(uint64(math.MaxUint64), "UNSIGNED BIGINT"); value != "18446744073709551615" {
		t.Fatalf("max unsigned bigint = %#v", value)
	}
	if value := normalizeValue(float32(1.1), "FLOAT"); value != 1.1 {
		t.Fatalf("float = %#v", value)
	}
	if value := normalizeValue([]byte("12.50"), "DECIMAL"); value != "12.50" {
		t.Fatalf("decimal = %#v", value)
	}
	if value := normalizeValue([]byte("0000-00-00 00:00:00"), "DATETIME"); value != "0000-00-00 00:00:00" {
		t.Fatalf("zero datetime = %#v", value)
	}
	if value := normalizeValue([]byte{1}, "BIT"); value != int64(1) {
		t.Fatalf("bit(1) = %#v", value)
	}
	if value := normalizeValue([]byte{1, 0}, "BIT"); value != int64(256) {
		t.Fatalf("bit(16) = %#v", value)
	}
	binary, ok := normalizeValue([]byte{1, 2, 3}, "BLOB").(model.BinaryCell)
	if !ok || binary.Length != 3 {
		t.Fatalf("blob = %#v", binary)
	}
	if value := normalizeValue([]byte("héllo"), "TEXT"); value != "héllo" {
		t.Fatalf("text = %#v", value)
	}
	if strings.Contains(dbengineErrorText(), "SQL Server") {
		t.Fatal("unsupported message must stay engine neutral")
	}
}
