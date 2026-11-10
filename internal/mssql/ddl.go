package mssql

import "strings"

// DetectDropDatabase returns the database named by a DROP DATABASE statement in one batch.
//
// The parser walks SQL tokens instead of searching raw text so keywords inside comments, string
// literals or bracketed identifiers cannot trigger the detection. It deliberately stops at the
// first DROP DATABASE, which is enough to route the batch through a master-context connection.
func DetectDropDatabase(statement string) (string, bool) {
	tokens := tokenizeSQL(statement)
	for i := 0; i+1 < len(tokens); i++ {
		if !tokens[i].keyword("DROP") || !tokens[i+1].keyword("DATABASE") {
			continue
		}
		next := i + 2
		if next+1 < len(tokens) && tokens[next].keyword("IF") && tokens[next+1].keyword("EXISTS") {
			next += 2
		}
		if next >= len(tokens) {
			return "", false
		}
		name := strings.TrimSpace(tokens[next].text)
		if name == "" || tokens[next].literal {
			return "", false
		}
		return name, true
	}
	return "", false
}

type sqlToken struct {
	text    string
	quoted  bool
	literal bool
}

func (t sqlToken) keyword(value string) bool {
	return !t.quoted && !t.literal && strings.EqualFold(t.text, value)
}

func tokenizeSQL(text string) []sqlToken {
	tokens := make([]sqlToken, 0, 32)
	for i := 0; i < len(text); {
		current := text[i]
		switch {
		case current == '-' && i+1 < len(text) && text[i+1] == '-':
			for i < len(text) && text[i] != '\n' {
				i++
			}
		case current == '/' && i+1 < len(text) && text[i+1] == '*':
			depth := 1
			i += 2
			for i < len(text) && depth > 0 {
				if text[i] == '/' && i+1 < len(text) && text[i+1] == '*' {
					depth++
					i += 2
					continue
				}
				if text[i] == '*' && i+1 < len(text) && text[i+1] == '/' {
					depth--
					i += 2
					continue
				}
				i++
			}
		case current == '\'':
			value, next := readDelimited(text, i+1, '\'')
			tokens = append(tokens, sqlToken{text: value, literal: true})
			i = next
		case current == '[':
			value, next := readDelimited(text, i+1, ']')
			tokens = append(tokens, sqlToken{text: value, quoted: true})
			i = next
		case current == '"':
			value, next := readDelimited(text, i+1, '"')
			tokens = append(tokens, sqlToken{text: value, quoted: true})
			i = next
		case isSQLSpace(current):
			i++
		case isSQLPunctuation(current):
			tokens = append(tokens, sqlToken{text: string(current)})
			i++
		default:
			start := i
			for i < len(text) && !isSQLSpace(text[i]) && !isSQLPunctuation(text[i]) &&
				!startsComment(text, i) {
				i++
			}
			tokens = append(tokens, sqlToken{text: text[start:i]})
		}
	}
	return tokens
}

// readDelimited reads an identifier or literal body with doubled closing characters as escapes.
func readDelimited(text string, start int, closing byte) (string, int) {
	var value strings.Builder
	for i := start; i < len(text); {
		if text[i] == closing {
			if i+1 < len(text) && text[i+1] == closing {
				value.WriteByte(closing)
				i += 2
				continue
			}
			return value.String(), i + 1
		}
		value.WriteByte(text[i])
		i++
	}
	return value.String(), len(text)
}

func startsComment(text string, index int) bool {
	if index+1 >= len(text) {
		return false
	}
	return (text[index] == '-' && text[index+1] == '-') ||
		(text[index] == '/' && text[index+1] == '*')
}

func isSQLSpace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func isSQLPunctuation(value byte) bool {
	switch value {
	case '(', ')', ',', ';', '.', '=', '+', '-', '*', '/', '%', '<', '>', '!', ':', '&', '|', '^', '~':
		return true
	default:
		return false
	}
}
