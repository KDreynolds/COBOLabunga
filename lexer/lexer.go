package lexer

import (
	"fmt"
	"strings"
	"unicode"
)

type Lexer struct {
	input    []rune
	pos      int
	line     int
	col      int
	keywords map[string]TokenType
}

func New(input string) *Lexer {
	l := &Lexer{
		input:    []rune(input),
		pos:      0,
		line:     1,
		col:      1,
		keywords: buildKeywordMap(),
	}
	return l
}

func buildKeywordMap() map[string]TokenType {
	return map[string]TokenType{
		"IDENTIFICATION":  IDENTIFICATION,
		"DIVISION":        DIVISION,
		"PROGRAM-ID":      PROGRAM_ID,
		"DATA":            DATA,
		"WORKING-STORAGE": WORKING_STORAGE,
		"SECTION":         SECTION,
		"PROCEDURE":       PROCEDURE,
		"PIC":             PIC,
		"PICTURE":         PICTURE,
		"COMP":            COMP,
		"OCCURS":          OCCURS,
		"TIMES":           TIMES,
		"VALUE":           VALUE,
		"MOVE":            MOVE,
		"TO":              TO,
		"COMPUTE":         COMPUTE,
		"DISPLAY":         DISPLAY,
		"PERFORM":         PERFORM,
		"IF":              IF,
		"ELSE":            ELSE,
		"END-IF":          END_IF,
		"EVALUATE":        EVALUATE,
		"WHEN":            WHEN,
		"OTHER":           OTHER,
		"END-EVALUATE":    END_EVALUATE,
		"STOP":            STOP,
		"RUN":             RUN,
		"HTTP-GET":        HTTP_GET,
		"HTTP-POST":       HTTP_POST,
		"HTTP-PUT":        HTTP_PUT,
		"HTTP-PATCH":      HTTP_PATCH,
		"HTTP-DELETE":     HTTP_DELETE,
		"GIVING":          GIVING,
		"SENDING":         SENDING,
		"STATUS":          STATUS,
		"HEADERS":         HEADERS,
		"MAPPING":         MAPPING,
		"ON":              ON,
		"EXCEPTION":       EXCEPTION,
		"NOT":             NOT,
		"END-HTTP":        END_HTTP,
	}
}

func (l *Lexer) peek() (rune, bool) {
	if l.pos >= len(l.input) {
		return 0, false
	}
	return l.input[l.pos], true
}

func (l *Lexer) read() (rune, bool) {
	if l.pos >= len(l.input) {
		return 0, false
	}
	ch := l.input[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch, true
}

func (l *Lexer) skipWhitespace() {
	for {
		ch, ok := l.peek()
		if !ok {
			break
		}
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			l.read()
		} else {
			break
		}
	}
}

func (l *Lexer) skipToEndOfLine() {
	for {
		ch, ok := l.read()
		if !ok || ch == '\n' {
			break
		}
	}
}

func (l *Lexer) Next() Token {
	l.skipWhitespace()

	ch, ok := l.peek()
	if !ok {
		return Token{Type: EOF, Literal: "", Line: l.line, Column: l.col}
	}

	line := l.line
	col := l.col

	switch {
	case ch == '.':
		l.read()
		return Token{Type: PERIOD, Literal: ".", Line: line, Column: col}
	case ch == '(':
		l.read()
		return Token{Type: LPAREN, Literal: "(", Line: line, Column: col}
	case ch == ')':
		l.read()
		return Token{Type: RPAREN, Literal: ")", Line: line, Column: col}
	case ch == '=':
		l.read()
		return Token{Type: EQUALS, Literal: "=", Line: line, Column: col}
	case ch == '+':
		l.read()
		return Token{Type: PLUS, Literal: "+", Line: line, Column: col}
	case ch == '-':
		l.read()
		nextCh, ok := l.peek()
		if ok && nextCh == '>' {
			l.read()
			l.skipToEndOfLine()
			return l.Next()
		}
		return Token{Type: MINUS, Literal: "-", Line: line, Column: col}
	case ch == '*':
		l.read()
		nextCh, ok := l.peek()
		if ok && nextCh == '>' {
			l.read()
			l.skipToEndOfLine()
			return l.Next()
		}
		return Token{Type: MULTIPLY, Literal: "*", Line: line, Column: col}
	case ch == '/':
		l.read()
		return Token{Type: DIVIDE, Literal: "/", Line: line, Column: col}
	case ch == '"' || ch == '\'':
		return l.readString(ch, line, col)
	case ch >= '0' && ch <= '9':
		return l.readNumber(line, col)
	case isLetter(ch):
		return l.readWord(line, col)
	default:
		l.read()
		return Token{Type: ILLEGAL, Literal: string(ch), Line: line, Column: col}
	}
}

func (l *Lexer) readString(quote rune, line, col int) Token {
	l.read()
	var buf strings.Builder
	for {
		ch, ok := l.read()
		if !ok {
			return Token{Type: ILLEGAL, Literal: buf.String(), Line: line, Column: col}
		}
		if ch == quote {
			break
		}
		if ch == '\n' {
			return Token{Type: ILLEGAL, Literal: buf.String(), Line: line, Column: col}
		}
		buf.WriteRune(ch)
	}
	return Token{Type: STRING_LITERAL, Literal: buf.String(), Line: line, Column: col}
}

func (l *Lexer) readNumber(line, col int) Token {
	var buf strings.Builder
	for {
		ch, ok := l.peek()
		if !ok || !isDigit(ch) {
			break
		}
		buf.WriteRune(ch)
		l.read()
	}
	return Token{Type: INTEGER_LITERAL, Literal: buf.String(), Line: line, Column: col}
}

func (l *Lexer) readWord(line, col int) Token {
	var buf strings.Builder
	for {
		ch, ok := l.peek()
		if !ok || !isWordChar(ch) {
			break
		}
		buf.WriteRune(ch)
		l.read()
	}
	word := buf.String()
	upper := strings.ToUpper(word)
	if typ, ok := l.keywords[upper]; ok {
		return Token{Type: typ, Literal: word, Line: line, Column: col}
	}
	return Token{Type: IDENTIFIER, Literal: word, Line: line, Column: col}
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isWordChar(ch rune) bool {
	return isLetter(ch) || isDigit(ch) || ch == '-'
}

func (l *Lexer) Tokenize() []Token {
	var tokens []Token
	for {
		tok := l.Next()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}
	return tokens
}

func (l *Lexer) PrintTokens() {
	tokens := l.Tokenize()
	for _, tok := range tokens {
		if tok.Type == EOF {
			fmt.Println("EOF")
		} else {
			fmt.Printf("%s(%s) line=%d col=%d\n", tok.Type, tok.Literal, tok.Line, tok.Column)
		}
	}
}
