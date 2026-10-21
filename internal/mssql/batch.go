package mssql

import (
	"strconv"
	"strings"
)

type SQLBatch struct {
	SQL         string
	StartLine   int
	RepeatCount int
}

// SplitSQLBatches follows the client-side GO rules used by SSMS and sqlcmd while ignoring
// separators inside strings, identifiers and nested block comments.
func SplitSQLBatches(text string) []SQLBatch {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	var batches []SQLBatch
	var current []numberedLine
	var single, bracket, double bool
	var blockDepth int

	for index, raw := range lines {
		line := strings.TrimSuffix(raw, "\r")
		initialFree := !single && !bracket && !double && blockDepth == 0
		code, endSingle, endBracket, endDouble, endDepth := maskLine(line, single, bracket, double, blockDepth)
		freeAtEnd := !endSingle && !endBracket && !endDouble && endDepth == 0
		trimmed := strings.TrimSpace(code)
		if initialFree && freeAtEnd && isGOCommand(trimmed) {
			if batch, ok := flushBatch(current, parseGORepeat(trimmed)); ok {
				batches = append(batches, batch)
			}
			current = nil
		} else {
			current = append(current, numberedLine{number: index + 1, text: line})
		}
		single, bracket, double, blockDepth = endSingle, endBracket, endDouble, endDepth
	}
	if batch, ok := flushBatch(current, 1); ok {
		batches = append(batches, batch)
	}
	return batches
}

type numberedLine struct {
	number int
	text   string
}

func maskLine(
	line string,
	single, bracket, double bool,
	blockDepth int,
) (string, bool, bool, bool, int) {
	runes := []rune(line)
	var code strings.Builder
	inLineComment := false
	for i := 0; i < len(runes); i++ {
		current := runes[i]
		var next rune
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		if inLineComment {
			break
		}
		if blockDepth > 0 {
			if current == '/' && next == '*' {
				blockDepth++
				i++
			} else if current == '*' && next == '/' {
				blockDepth--
				i++
			}
			continue
		}
		if single {
			if current == '\'' {
				if next == '\'' {
					i++
				} else {
					single = false
				}
			}
			continue
		}
		if bracket {
			if current == ']' {
				if next == ']' {
					i++
				} else {
					bracket = false
				}
			}
			continue
		}
		if double {
			if current == '"' {
				if next == '"' {
					i++
				} else {
					double = false
				}
			}
			continue
		}
		if current == '-' && next == '-' {
			inLineComment = true
			i++
		} else if current == '/' && next == '*' {
			blockDepth++
			i++
		} else if current == '\'' {
			single = true
		} else if current == '[' {
			bracket = true
		} else if current == '"' {
			double = true
		} else {
			code.WriteRune(current)
		}
	}
	return code.String(), single, bracket, double, blockDepth
}

func isGOCommand(value string) bool {
	if len(value) < 2 || !strings.EqualFold(value[:2], "go") {
		return false
	}
	rest := strings.TrimSpace(value[2:])
	if rest == "" {
		return true
	}
	for _, r := range rest {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func parseGORepeat(value string) int {
	rest := strings.TrimSpace(value[2:])
	if rest == "" {
		return 1
	}
	count, err := strconv.Atoi(rest)
	if err != nil || count < 1 {
		return 1
	}
	return count
}

func flushBatch(lines []numberedLine, repeat int) (SQLBatch, bool) {
	first := -1
	last := -1
	for i, line := range lines {
		if strings.TrimSpace(line.text) != "" {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return SQLBatch{}, false
	}
	selected := make([]string, 0, last-first+1)
	for _, line := range lines[first : last+1] {
		selected = append(selected, line.text)
	}
	text := strings.TrimSpace(strings.Join(selected, "\n"))
	if text == "" {
		return SQLBatch{}, false
	}
	if repeat < 1 {
		repeat = 1
	}
	return SQLBatch{SQL: text, StartLine: lines[first].number, RepeatCount: repeat}, true
}

func DetectUseDatabase(statement string) string {
	trimmed := strings.TrimSpace(statement)
	if len(trimmed) < 4 || !strings.EqualFold(trimmed[:3], "use") {
		return ""
	}
	if trimmed[3] != ' ' && trimmed[3] != '\t' && trimmed[3] != '\r' && trimmed[3] != '\n' {
		return ""
	}
	rest := strings.TrimSpace(trimmed[3:])
	if strings.HasPrefix(rest, "[") {
		end := strings.Index(rest, "]")
		if end > 1 {
			return strings.TrimSpace(rest[1:end])
		}
		return ""
	}
	end := len(rest)
	for i, r := range rest {
		if r == ';' || r == '-' || r == ' ' || r == '\t' || r == '\r' || r == '\n' {
			end = i
			break
		}
	}
	return strings.TrimSpace(rest[:end])
}
