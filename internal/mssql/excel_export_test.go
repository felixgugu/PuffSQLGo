package mssql

import (
	"testing"

	"puffsql/internal/model"
)

func TestExcelSheetNamesAreSafeAndUnique(t *testing.T) {
	used := map[string]bool{}
	first := uniqueExcelSheetName("dbo.Sales/Archive:2026?*[]", used)
	second := uniqueExcelSheetName("dbo.Sales/Archive:2026?*[]", used)
	if first == second {
		t.Fatalf("duplicate sheet name: %s", first)
	}
	if len([]rune(first)) > 31 {
		t.Fatalf("sheet name too long: %s", first)
	}
	for _, invalid := range []string{"[", "]", ":", "*", "?", "/", "\\"} {
		if containsRune(first, invalid) {
			t.Fatalf("sheet %q still contains %q", first, invalid)
		}
	}
	if second[len(second)-2:] != "_2" {
		t.Fatalf("second sheet should have suffix: %s", second)
	}
}

func TestExcelCellValueKeepsLargeIntegersExact(t *testing.T) {
	if got := excelCellValue(int64(9_007_199_254_740_993)); got != "9007199254740993" {
		t.Fatalf("large integer = %#v", got)
	}
	if got := excelCellValue(model.BinaryCell{Type: "binary", Length: 12}); got != "[Binary 12 bytes]" {
		t.Fatalf("binary marker = %#v", got)
	}
}

func containsRune(value, needle string) bool {
	for _, r := range value {
		if string(r) == needle {
			return true
		}
	}
	return false
}
