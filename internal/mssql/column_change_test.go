package mssql

import (
	"strings"
	"testing"

	"puffsql/internal/model"
)

func TestValidatedColumnType(t *testing.T) {
	length := 120
	scalar := validatedType(t, model.ColumnChangeRequest{DataType: "nvarchar", Length: &length})
	if scalar != "[nvarchar](120)" {
		t.Fatalf("nvarchar type = %q", scalar)
	}
	max := -1
	if value, err := validatedColumnType(model.ColumnChangeRequest{DataType: "varbinary", Length: &max}); err != nil || value != "[varbinary](MAX)" {
		t.Fatalf("varbinary(MAX) = %q, %v", value, err)
	}
	precision, scale := 18, 2
	if value, err := validatedColumnType(model.ColumnChangeRequest{DataType: "decimal", Precision: &precision, Scale: &scale}); err != nil || value != "[decimal](18, 2)" {
		t.Fatalf("decimal = %q, %v", value, err)
	}
	if _, err := validatedColumnType(model.ColumnChangeRequest{DataType: "int", Length: &length}); err != nil {
		t.Fatal(err)
	}
	if _, err := validatedColumnType(model.ColumnChangeRequest{DataType: "int; DROP TABLE Users"}); err == nil {
		t.Fatal("unknown type must be rejected")
	}
}

func validatedType(t *testing.T, req model.ColumnChangeRequest) string {
	t.Helper()
	value, err := validatedColumnType(req)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestDefaultExpressionValidation(t *testing.T) {
	value := "((0))"
	if got, err := cleanDefaultExpression(&value); err != nil || got != "((0))" {
		t.Fatalf("default = %q, %v", got, err)
	}
	bad := "0; DROP TABLE [dbo].[Users]"
	if _, err := cleanDefaultExpression(&bad); err == nil {
		t.Fatal("multi-statement default must be rejected")
	}
}

func TestColumnDescriptionStatements(t *testing.T) {
	added := "客戶名稱"
	statements := descriptionStatements("dbo", "Customers", "Name", nil, &added)
	if len(statements) != 1 || !strings.Contains(statements[0], "sp_addextendedproperty") ||
		!strings.Contains(statements[0], "N'客戶名稱'") {
		t.Fatalf("add description = %#v", statements)
	}
	updated := "顯示名稱"
	statements = descriptionStatements("dbo", "Customers", "Name", &added, &updated)
	if len(statements) != 1 || !strings.Contains(statements[0], "sp_updateextendedproperty") {
		t.Fatalf("update description = %#v", statements)
	}
	statements = descriptionStatements("dbo", "Customers", "Name", &updated, nil)
	if len(statements) != 1 || !strings.Contains(statements[0], "sp_dropextendedproperty") {
		t.Fatalf("drop description = %#v", statements)
	}
}

func TestDependencyDetectionAndFingerprint(t *testing.T) {
	column := "CustomerId"
	objects := model.TableObjects{
		Constraints: []model.TableConstraintItem{{
			Name: "FK_Orders_Customers", Kind: "FOREIGN_KEY",
			KeyColumns: []model.KeyColumnRef{{Name: column}},
			Definition: nil, DefaultColumn: nil, RefColumns: []string{},
		}},
		Indexes: []model.TableIndexItem{{
			Name: "IX_Orders_Customer", TypeDesc: "NONCLUSTERED",
			Columns: []model.IndexColumnRef{{Name: column}},
		}},
		Columns: []model.ColumnItem{{
			Name: "CustomerLabel", ComputedExpression: stringPtr("[CustomerId] + 1"),
		}},
	}
	dependencies := collectColumnDependencies(objects, column)
	if len(dependencies) != 3 {
		t.Fatalf("dependencies = %#v", dependencies)
	}
	preview := model.ColumnChangePreview{SQL: "ALTER TABLE [dbo].[Orders] DROP COLUMN [CustomerId];", Dependencies: dependencies}
	first := columnPreviewFingerprint(preview)
	second := columnPreviewFingerprint(preview)
	if first == "" || first != second {
		t.Fatalf("fingerprint is not stable: %q / %q", first, second)
	}
	preview.SQL += "\nGO"
	if columnPreviewFingerprint(preview) == first {
		t.Fatal("fingerprint must change with the SQL")
	}
}

func TestGeneratedDefaultNameAvoidsCollisions(t *testing.T) {
	existing := []model.TableConstraintItem{{Name: "DF_dbo_Users_Name"}}
	name := generatedDefaultName(existing, "dbo", "Users", "Name")
	if !strings.HasSuffix(name, "_2") {
		t.Fatalf("collision suffix missing: %s", name)
	}
}

func stringPtr(value string) *string { return &value }
