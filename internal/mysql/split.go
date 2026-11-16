package mysql

import (
	"regexp"
	"strings"
)

// Statement is one executable statement of a script and the 1-based line it starts on.
type Statement struct {
	SQL       string
	StartLine int
}

var (
	delimiterCommand = regexp.MustCompile(`(?i)^delimiter[ \t]+(\S+)`)
	useCommand       = regexp.MustCompile("(?is)^use\\s+(`(?:[^`]|``)+`|[^\\s;]+)\\s*$")
)

// SplitStatements splits a script the way the mysql client does: on the current delimiter
// (";" until a DELIMITER command changes it), ignoring delimiters inside quoted strings,
// backtick identifiers and comments. Comments before a statement are dropped; statements that
// contain nothing but comments are skipped.
func SplitStatements(script string) []Statement {
	script = strings.ReplaceAll(script, "\r\n", "\n")
	var statements []Statement
	var current strings.Builder
	delimiter := ";"
	line := 1
	startLine := 0
	hasContent := false

	flush := func() {
		text := strings.TrimSpace(current.String())
		if hasContent && text != "" {
			statements = append(statements, Statement{SQL: text, StartLine: startLine})
		}
		current.Reset()
		hasContent = false
	}
	// emit copies script[from:to] into the current statement, or drops it while no statement
	// has started (leading comments and whitespace).
	emit := func(text string, content bool) {
		if content && !hasContent {
			hasContent = true
			startLine = line
		}
		if hasContent {
			current.WriteString(text)
		}
		line += strings.Count(text, "\n")
	}

	for i := 0; i < len(script); {
		c := script[i]
		// DELIMITER is a client command: it is only recognised at the start of a line, between
		// statements.
		if !hasContent && strings.TrimSpace(lineHead(script, i)) == "" {
			if match := delimiterCommand.FindStringSubmatch(lineTail(script, i)); match != nil {
				delimiter = match[1]
				end := i + len(lineTail(script, i))
				line += strings.Count(script[i:end], "\n")
				i = end
				continue
			}
		}
		if strings.HasPrefix(script[i:], delimiter) {
			flush()
			i += len(delimiter)
			continue
		}
		switch {
		case c == '\'' || c == '"' || c == '`':
			end := quotedEnd(script, i)
			emit(script[i:end], true)
			i = end
		case c == '#' || (c == '-' && isDashComment(script, i)):
			end := i + len(lineTail(script, i))
			emit(script[i:end], false)
			i = end
		case c == '/' && i+1 < len(script) && script[i+1] == '*':
			end := strings.Index(script[i+2:], "*/")
			if end < 0 {
				end = len(script)
			} else {
				end += i + 4
			}
			// "/*!" and "/*+" are executable version comments and optimizer hints.
			executable := i+2 < len(script) && (script[i+2] == '!' || script[i+2] == '+')
			emit(script[i:end], executable)
			i = end
		default:
			emit(script[i:i+1], c != ' ' && c != '\t' && c != '\n' && c != '\r')
			i++
		}
	}
	flush()
	return statements
}

// DetectUseDatabase returns the database named by a "USE db" statement, or "".
func DetectUseDatabase(statement string) string {
	match := useCommand.FindStringSubmatch(strings.TrimSpace(statement))
	if match == nil {
		return ""
	}
	name := match[1]
	if strings.HasPrefix(name, "`") {
		name = strings.ReplaceAll(name[1:len(name)-1], "``", "`")
	}
	return name
}

// quotedEnd returns the index just past the quoted run starting at start. Strings honour
// backslash escapes and doubled quotes; backtick identifiers honour doubled backticks only.
func quotedEnd(script string, start int) int {
	quote := script[start]
	for i := start + 1; i < len(script); i++ {
		switch {
		case script[i] == '\\' && quote != '`':
			i++
		case script[i] == quote:
			if i+1 < len(script) && script[i+1] == quote {
				i++
				continue
			}
			return i + 1
		}
	}
	return len(script)
}

// isDashComment reports whether "--" at i starts a comment: MySQL requires whitespace (or the
// end of the script) after the two dashes.
func isDashComment(script string, i int) bool {
	if i+1 >= len(script) || script[i+1] != '-' {
		return false
	}
	if i+2 >= len(script) {
		return true
	}
	next := script[i+2]
	return next == ' ' || next == '\t' || next == '\n' || next == '\r'
}

// lineTail is the rest of the current line from i, without the line break.
func lineTail(script string, i int) string {
	if end := strings.IndexByte(script[i:], '\n'); end >= 0 {
		return script[i : i+end]
	}
	return script[i:]
}

// lineHead is the part of the current line before i.
func lineHead(script string, i int) string {
	start := strings.LastIndexByte(script[:i], '\n') + 1
	return script[start:i]
}
