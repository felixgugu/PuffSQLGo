package mysql

import (
	"reflect"
	"testing"
)

func statementTexts(script string) []string {
	texts := []string{}
	for _, statement := range SplitStatements(script) {
		texts = append(texts, statement.SQL)
	}
	return texts
}

func TestSplitStatementsOnSemicolonsOutsideQuotesAndComments(t *testing.T) {
	script := "SELECT ';' AS a, \"x;y\" AS b, `odd;name` FROM t; -- trailing; comment\n" +
		"# hash; comment\n" +
		"/* block; comment */\n" +
		"INSERT INTO t VALUES ('it''s', 'a\\'b;c');\n" +
		"SELECT 1"
	got := statementTexts(script)
	want := []string{
		"SELECT ';' AS a, \"x;y\" AS b, `odd;name` FROM t",
		"INSERT INTO t VALUES ('it''s', 'a\\'b;c')",
		"SELECT 1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("statements = %#v", got)
	}
}

func TestSplitStatementsReportsStartLinesAndSkipsCommentOnlyParts(t *testing.T) {
	statements := SplitStatements("-- header\n\nSELECT 1;\n\n/* note */\n  SELECT\n 2;\n-- tail only\n;")
	if len(statements) != 2 {
		t.Fatalf("statements = %#v", statements)
	}
	if statements[0].StartLine != 3 || statements[1].StartLine != 6 {
		t.Fatalf("start lines = %d, %d", statements[0].StartLine, statements[1].StartLine)
	}
}

func TestSplitStatementsHonoursDelimiterCommand(t *testing.T) {
	script := "DELIMITER $$\n" +
		"CREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\n  SELECT 2;\nEND$$\n" +
		"delimiter ;\n" +
		"CALL p();"
	got := statementTexts(script)
	want := []string{
		"CREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\n  SELECT 2;\nEND",
		"CALL p()",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("statements = %#v", got)
	}
}

func TestSplitStatementsKeepsExecutableCommentsAndMinusMinus(t *testing.T) {
	got := statementTexts("/*!40101 SET NAMES utf8mb4 */;\nSELECT 5--2;")
	want := []string{"/*!40101 SET NAMES utf8mb4 */", "SELECT 5--2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("statements = %#v", got)
	}
}

func TestDetectUseDatabase(t *testing.T) {
	cases := map[string]string{
		"USE shop":            "shop",
		"use `my``db`":        "my`db",
		"  USE\n  inventory ": "inventory",
		"SELECT 'USE shop'":   "",
		"USE shop extra":      "",
	}
	for statement, want := range cases {
		if got := DetectUseDatabase(statement); got != want {
			t.Errorf("DetectUseDatabase(%q) = %q, want %q", statement, got, want)
		}
	}
}
