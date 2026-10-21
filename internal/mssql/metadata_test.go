package mssql

import "testing"

func TestParseColumnsCarriesDDLAttributes(t *testing.T) {
	set := rowSet{Columns: []string{"ignored"}, Rows: [][]any{{
		"Id", "int", int64(4), int64(10), int64(0), int64(0), int64(1), int64(1),
		int64(0), int64(0), "", "", "", int64(0), int64(100), int64(2), "主鍵識別碼",
	}}}
	columns := parseColumns(set)
	if len(columns) != 1 {
		t.Fatalf("columns = %#v", columns)
	}
	column := columns[0]
	if !column.IsIdentity || !column.IsPrimaryKey || column.IsNullable {
		t.Fatalf("flags were parsed incorrectly: %#v", column)
	}
	if column.MaxLength == nil || *column.MaxLength != 4 {
		t.Fatalf("max length = %#v", column.MaxLength)
	}
	if column.IdentitySeed == nil || *column.IdentitySeed != 100 ||
		column.IdentityIncrement == nil || *column.IdentityIncrement != 2 {
		t.Fatalf("identity attributes = %#v", column)
	}
	if column.Description == nil || *column.Description != "主鍵識別碼" {
		t.Fatalf("description = %#v", column.Description)
	}
}

func TestTableObjectParsersBuildRelationshipsInServerOrder(t *testing.T) {
	constraints := parseConstraints([]rowSet{
		{},
		{Rows: [][]any{
			{"PK_Orders", "PRIMARY_KEY", int64(0), int64(1), "Id", int64(0)},
			{"PK_Orders", "PRIMARY_KEY", int64(0), int64(1), "TenantId", int64(0)},
		}},
		{Rows: [][]any{{"CK_Total", int64(0), int64(0), "([Total]>=(0))"}}},
		{Rows: [][]any{{"DF_Status", "('open')", "Status"}}},
		{Rows: [][]any{{
			"FK_Customer", int64(0), int64(1), "NO_ACTION", "NO_ACTION",
			"dbo", "Customers", "CustomerId", "Id",
		}}},
	})
	if len(constraints) != 4 {
		t.Fatalf("constraints = %#v", constraints)
	}
	if len(constraints[0].KeyColumns) != 2 || constraints[0].KeyColumns[1].Name != "TenantId" {
		t.Fatalf("composite key missing: %#v", constraints[0])
	}
	if constraints[3].Kind != "FOREIGN_KEY" || constraints[3].RefTable == nil ||
		*constraints[3].RefTable != "Customers" {
		t.Fatalf("foreign key missing: %#v", constraints[3])
	}

	indexes := parseIndexes(rowSet{Rows: [][]any{
		{"IX_Orders", "NONCLUSTERED", int64(0), int64(0), int64(0), int64(0), "", "CustomerId", int64(0), int64(0)},
		{"IX_Orders", "NONCLUSTERED", int64(0), int64(0), int64(0), int64(0), "", "Total", int64(1), int64(1)},
	}})
	if len(indexes) != 1 || len(indexes[0].Columns) != 2 || !indexes[0].Columns[1].IsIncluded {
		t.Fatalf("index columns = %#v", indexes)
	}

	triggers := parseTriggers(rowSet{Rows: [][]any{{
		"trg_Orders_Audit", int64(0), int64(0), "CREATE TRIGGER ...", "INSERT, UPDATE",
	}}})
	if len(triggers) != 1 || len(triggers[0].Events) != 2 || triggers[0].Events[1] != "UPDATE" {
		t.Fatalf("trigger events = %#v", triggers)
	}
}

func TestParseColumnsReturnsEmptySliceNotNull(t *testing.T) {
	columns := parseColumns(rowSet{Rows: [][]any{}})
	if columns == nil || len(columns) != 0 {
		t.Fatalf("want an empty slice, got %#v", columns)
	}
}
