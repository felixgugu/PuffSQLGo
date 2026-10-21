package mssql

import (
	"reflect"
	"testing"
)

func TestSplitSQLBatches(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want []SQLBatch
	}{
		{
			name: "single statement",
			sql:  "SELECT 1;",
			want: []SQLBatch{{SQL: "SELECT 1;", StartLine: 1, RepeatCount: 1}},
		},
		{
			name: "repeat and original line numbers",
			sql:  "SELECT 1;\nGO 3 -- repeat\n\nSELECT 2;\nGO",
			want: []SQLBatch{
				{SQL: "SELECT 1;", StartLine: 1, RepeatCount: 3},
				{SQL: "SELECT 2;", StartLine: 4, RepeatCount: 1},
			},
		},
		{
			name: "GO inside literals identifiers and nested comments",
			sql: "SELECT 'line 1\nGO\nline 3', [GO], \"go\"\n" +
				"/* outer /* GO */ still comment */;\nGO\nSELECT 2;",
			want: []SQLBatch{
				{SQL: "SELECT 'line 1\nGO\nline 3', [GO], \"go\"\n/* outer /* GO */ still comment */;", StartLine: 1, RepeatCount: 1},
				{SQL: "SELECT 2;", StartLine: 6, RepeatCount: 1},
			},
		},
		{
			name: "multibyte text does not panic",
			sql:  "無線滑鼠\nSELECT 1;\ngo\n資料表",
			want: []SQLBatch{
				{SQL: "無線滑鼠\nSELECT 1;", StartLine: 1, RepeatCount: 1},
				{SQL: "資料表", StartLine: 4, RepeatCount: 1},
			},
		},
		{
			name: "empty boundaries are pruned",
			sql:  "SELECT 1;\nGO\n\nGO\nGO\nSELECT 2;",
			want: []SQLBatch{
				{SQL: "SELECT 1;", StartLine: 1, RepeatCount: 1},
				{SQL: "SELECT 2;", StartLine: 6, RepeatCount: 1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SplitSQLBatches(tt.sql); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SplitSQLBatches()\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func TestGODetectionOnlyAcceptsTheClientCommand(t *testing.T) {
	for _, value := range []string{"GO", "go 5", "Go 12", "gO"} {
		if !isGOCommand(value) {
			t.Errorf("%q should be a GO boundary", value)
		}
	}
	for _, value := range []string{"GOTO", "g", "GO x", "GO 5 extra", "ｇｏ", "無線滑鼠"} {
		if isGOCommand(value) {
			t.Errorf("%q should not be a GO boundary", value)
		}
	}
}

func TestDetectUseDatabase(t *testing.T) {
	tests := map[string]string{
		"USE master;":               "master",
		"use [AdventureWorks2022];": "AdventureWorks2022",
		"USE [My DB] ; -- comment":  "My DB",
		"USER_NAME()":               "",
		"use":                       "",
		"SELECT * FROM dbo.Users":   "",
	}
	for sql, want := range tests {
		if got := DetectUseDatabase(sql); got != want {
			t.Errorf("DetectUseDatabase(%q) = %q, want %q", sql, got, want)
		}
	}
}
