package mysql

import (
	"math"
	"strconv"
	"strings"
	"time"

	"puffsql/internal/model"
	"puffsql/internal/sqlshared"
)

// normalizeValue converts one scanned MySQL value into the cell the grid displays. The driver
// already decodes integers and floats; everything else arrives as raw bytes.
func normalizeValue(value any, dataType string) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case int64:
		return sqlshared.SafeInteger(typed)
	case uint64:
		if typed > math.MaxInt64 {
			return strconv.FormatUint(typed, 10)
		}
		return sqlshared.SafeInteger(int64(typed))
	case float32:
		// Widening a float32 directly would surface binary noise (1.1 -> 1.100000023841858).
		widened, err := strconv.ParseFloat(strconv.FormatFloat(float64(typed), 'g', -1, 32), 64)
		if err != nil {
			return float64(typed)
		}
		return widened
	case float64:
		return typed
	case time.Time:
		return typed.Format("2006-01-02 15:04:05.999999999")
	case []byte:
		return normalizeBytes(typed, strings.ToUpper(dataType))
	default:
		return typed
	}
}

func normalizeBytes(value []byte, dataType string) any {
	switch dataType {
	case "BLOB", "TINYBLOB", "MEDIUMBLOB", "LONGBLOB", "BINARY", "VARBINARY", "GEOMETRY", "VECTOR":
		return model.BinaryCell{Type: "binary", Length: len(value)}
	case "BIT":
		// BIT(n) arrives big-endian; show it as the number it encodes.
		if len(value) > 8 {
			return model.BinaryCell{Type: "binary", Length: len(value)}
		}
		var number uint64
		for _, b := range value {
			number = number<<8 | uint64(b)
		}
		if number > math.MaxInt64 {
			return strconv.FormatUint(number, 10)
		}
		return sqlshared.SafeInteger(int64(number))
	default:
		return string(value)
	}
}
