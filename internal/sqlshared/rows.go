// Package sqlshared holds the engine-neutral database/sql plumbing that every supported
// database engine reuses: result-set streaming, completion messages and the Excel writers.
package sqlshared

import (
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"time"

	"puffsql/internal/model"
)

// Normalizer converts one scanned driver value into the JSON-safe cell the grid displays.
// typeName is the driver's DatabaseTypeName for the column ("" when unknown).
type Normalizer func(value any, typeName string) any

// maxSafeJSInteger is JavaScript's Number.MAX_SAFE_INTEGER (2^53 - 1).
const maxSafeJSInteger = 1<<53 - 1

// SafeInteger keeps integers the frontend can represent exactly as JSON numbers and sends larger
// ones (bigint beyond ±2^53-1) as decimal text, because a JavaScript Number would round them.
func SafeInteger(value int64) any {
	if value > maxSafeJSInteger || value < -maxSafeJSInteger {
		return strconv.FormatInt(value, 10)
	}
	return value
}

// CollectRows drains every result set of one statement into result.
func CollectRows(rows *sql.Rows, maxRows *int, result *model.QueryResult, normalize Normalizer) error {
	for {
		columns, err := rows.Columns()
		if err != nil {
			return err
		}
		if len(columns) > 0 {
			set, err := ReadResultSet(rows, columns, maxRows, normalize)
			if err != nil {
				return err
			}
			result.ResultSets = append(result.ResultSets, set)
		} else {
			for rows.Next() {
				// Drain non-tabular result sets so the driver can reach the next response.
			}
			if err := rows.Err(); err != nil {
				return err
			}
		}
		if !rows.NextResultSet() {
			break
		}
	}
	return rows.Err()
}

// ReadResultSet reads the current result set, keeping at most maxRows rows while still counting
// the rest so the UI can report truncation.
func ReadResultSet(
	rows *sql.Rows,
	names []string,
	maxRows *int,
	normalize Normalizer,
) (model.ResultSet, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return model.ResultSet{}, err
	}
	limit := math.MaxInt
	if maxRows != nil && *maxRows >= 0 {
		limit = *maxRows
	}
	set := model.ResultSet{
		Columns: make([]model.ColumnDef, len(names)),
		Rows:    [][]any{},
	}
	typeNames := make([]string, len(names))
	for i, name := range names {
		dataType := "unknown"
		if i < len(types) {
			dataType = types[i].DatabaseTypeName()
			typeNames[i] = dataType
		}
		set.Columns[i] = model.ColumnDef{Name: name, DataType: dataType, Nullable: true, Ordinal: i}
	}

	scans := make([]any, len(names))
	values := make([]any, len(names))
	for i := range scans {
		scans[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(scans...); err != nil {
			return model.ResultSet{}, err
		}
		set.TotalCount++
		if len(set.Rows) >= limit {
			set.IsTruncated = true
			continue
		}
		row := make([]any, len(values))
		for i, value := range values {
			row[i] = normalize(value, typeNames[i])
		}
		set.Rows = append(set.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return model.ResultSet{}, err
	}
	set.RowCount = len(set.Rows)
	return set, nil
}

// CancelledMessage is the warning every engine reports when the user cancels a script.
func CancelledMessage() model.QueryMessage {
	return model.QueryMessage{
		Level:     "warning",
		Message:   "Query was cancelled by user",
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// AppendCompletion fills the row totals and appends the truncation warning and the success
// summary once every batch of a script has run.
func AppendCompletion(result *model.QueryResult, executedBatches int, maxRows *int, started time.Time) {
	var rows uint64
	hasTruncated := false
	for _, set := range result.ResultSets {
		rows += uint64(set.RowCount)
		hasTruncated = hasTruncated || set.IsTruncated
	}
	result.AffectedRows = rows
	if hasTruncated && maxRows != nil {
		result.Messages = append(result.Messages, model.QueryMessage{
			Level: "warning",
			Message: fmt.Sprintf(
				"查詢結果已達最大限制 %d 筆，其餘資料已自動截斷以保護系統效能。", *maxRows,
			),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}
	message := fmt.Sprintf(
		"Query completed successfully. %d result set(s), %d rows returned.",
		len(result.ResultSets), rows,
	)
	if executedBatches > 1 {
		message = fmt.Sprintf(
			"Query completed successfully. Executed %d batch(es), %d result set(s), %d rows returned.",
			executedBatches, len(result.ResultSets), rows,
		)
	}
	result.Messages = append(result.Messages, model.QueryMessage{
		Level:     "info",
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
	})
	result.ExecutionTime = ElapsedMillis(started)
}

// ElapsedMillis reports the whole milliseconds elapsed since start.
func ElapsedMillis(start time.Time) uint64 {
	elapsed := time.Since(start)
	if elapsed < 0 {
		return 0
	}
	return uint64(elapsed.Milliseconds())
}
