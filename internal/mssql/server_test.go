package mssql

import (
	"net/url"
	"testing"

	"puffsql/internal/model"
)

func TestDSNEscapesCredentialsAndCarriesSecurityOptions(t *testing.T) {
	profile := model.ConnectionProfile{
		ConnectionConfig: model.ConnectionConfig{
			Host: "2001:db8::1", Port: 14330, Database: "Sales DB",
			Username: "user@example.com", Encrypt: true, TrustServerCertificate: true,
		},
	}
	dsn, err := DSN(profile, "p@ss:/?#[]")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "sqlserver" || parsed.Hostname() != "2001:db8::1" || parsed.Port() != "14330" {
		t.Fatalf("unexpected endpoint in %q", dsn)
	}
	password, _ := parsed.User.Password()
	if parsed.User.Username() != "user@example.com" || password != "p@ss:/?#[]" {
		t.Fatalf("credentials were not round-tripped: %q / %q", parsed.User.Username(), password)
	}
	query := parsed.Query()
	if query.Get("database") != "Sales DB" || query.Get("encrypt") != "true" ||
		query.Get("TrustServerCertificate") != "true" || query.Get("app name") != "PuffSQL" {
		t.Fatalf("security options missing from %q", dsn)
	}
}

func TestDSNUsesStandardPortAndRejectsMissingHost(t *testing.T) {
	profile := model.ConnectionProfile{ConnectionConfig: model.ConnectionConfig{Host: "localhost"}}
	dsn, err := DSN(profile, "")
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(dsn)
	if parsed.Port() != "1433" {
		t.Fatalf("want default port 1433, got %q", parsed.Port())
	}
	if _, err := DSN(model.ConnectionProfile{}, ""); err == nil {
		t.Fatal("an empty host must be rejected")
	}
}

func TestQuoteIdentifier(t *testing.T) {
	if got := QuoteIdentifier("we]ird"); got != "[we]]ird]" {
		t.Fatalf("QuoteIdentifier() = %q", got)
	}
	if got := QualifiedTable("dbo", "Order Details"); got != "[dbo].[Order Details]" {
		t.Fatalf("QualifiedTable() = %q", got)
	}
}
