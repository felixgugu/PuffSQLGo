package mssql

import "testing"

func TestDetectDropDatabase(t *testing.T) {
	cases := map[string]string{
		"DROP DATABASE [Sales];":                      "Sales",
		"drop database Sales":                         "Sales",
		"DROP  DATABASE\n  \"Sales Archive\"":         "Sales Archive",
		"DROP DATABASE IF EXISTS [My]]Db]":            "My]Db",
		"/* DROP DATABASE [Ignored] */ SELECT 1":      "",
		"-- DROP DATABASE [Ignored]\nSELECT 1":        "",
		"SELECT 'DROP DATABASE [Ignored]' AS example": "",
		"USE master;\nGO\nDROP DATABASE [Sales];\nGO": "Sales",
		"ALTER DATABASE [Sales] SET SINGLE_USER":      "",
	}
	for statement, want := range cases {
		got, ok := DetectDropDatabase(statement)
		if want == "" {
			if ok {
				t.Fatalf("DetectDropDatabase(%q) = %q, want no match", statement, got)
			}
			continue
		}
		if !ok || got != want {
			t.Fatalf("DetectDropDatabase(%q) = %q, %v; want %q", statement, got, ok, want)
		}
	}
}
