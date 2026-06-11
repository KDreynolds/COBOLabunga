package parser

import (
	"fmt"
	"strconv"
	"unicode"

	"github.com/KDreynolds/COBOLabunga/lexer"
)

type ParseError struct {
	Line   int
	Column int
	Msg    string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d:%d: %s", e.Line, e.Column, e.Msg)
}

type Parser struct {
	tokens []lexer.Token
	pos    int
	errs   []*ParseError
}

func New(tokens []lexer.Token) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
	}
}

func (p *Parser) Parse() (*Program, []*ParseError) {
	prog := &Program{}

	for !p.atEnd() {
		switch p.peek().Type {
		case lexer.IDENTIFICATION:
			p.parseIdentificationDivision(prog)
		case lexer.DATA:
			p.parseDataDivision(prog)
		case lexer.PROCEDURE:
			p.parseProcedureDivision(prog)
		default:
			if p.peek().Type == lexer.EOF {
				break
			}
			p.error("unexpected token: " + p.peek().Literal)
			p.advance()
		}
	}

	return prog, p.errs
}

// --- Parser helpers ---

func (p *Parser) atEnd() bool {
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Type == lexer.EOF
}

func (p *Parser) peek() lexer.Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return lexer.Token{Type: lexer.EOF}
}

func (p *Parser) previous() lexer.Token {
	if p.pos > 0 {
		return p.tokens[p.pos-1]
	}
	return lexer.Token{}
}

func (p *Parser) advance() lexer.Token {
	tok := p.peek()
	p.pos++
	return tok
}

func (p *Parser) match(types ...lexer.TokenType) bool {
	for _, typ := range types {
		if p.peek().Type == typ {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) expect(typ lexer.TokenType) (lexer.Token, *ParseError) {
	if p.peek().Type == typ {
		return p.advance(), nil
	}
	tok := p.peek()
	err := &ParseError{
		Line:   tok.Line,
		Column: tok.Column,
		Msg:    fmt.Sprintf("expected %s, got %s(%s)", typ, tok.Type, tok.Literal),
	}
	p.errs = append(p.errs, err)
	return tok, err
}

func (p *Parser) error(msg string) {
	tok := p.peek()
	p.errs = append(p.errs, &ParseError{
		Line:   tok.Line,
		Column: tok.Column,
		Msg:    msg,
	})
}

func (p *Parser) errorAt(tok lexer.Token, msg string) {
	p.errs = append(p.errs, &ParseError{
		Line:   tok.Line,
		Column: tok.Column,
		Msg:    msg,
	})
}

// --- Division-level parsing ---

func (p *Parser) parseIdentificationDivision(prog *Program) {
	p.expect(lexer.IDENTIFICATION)
	p.expect(lexer.DIVISION)
	p.match(lexer.PERIOD)

	if p.match(lexer.PROGRAM_ID) {
		p.match(lexer.PERIOD)
		if p.peek().Type == lexer.IDENTIFIER {
			prog.Name = p.advance().Literal
		}
		p.match(lexer.PERIOD)
	}
}

func (p *Parser) parseDataDivision(prog *Program) {
	p.expect(lexer.DATA)
	p.expect(lexer.DIVISION)
	p.match(lexer.PERIOD)

	if p.match(lexer.WORKING_STORAGE) {
		p.expect(lexer.SECTION)
		p.match(lexer.PERIOD)
		ws := &WorkingStorage{}
		p.parseWorkingStorageItems(ws)
		prog.WorkingStorage = ws
	}
}

func (p *Parser) parseWorkingStorageItems(ws *WorkingStorage) {
	// collect all items in order; build hierarchy
	var items []*DataItem
	for !p.atEnd() {
		if p.peek().Type != lexer.INTEGER_LITERAL {
			// not a level number, end of working storage
			break
		}
		item := p.parseDataItem()
		if item != nil {
			items = append(items, item)
		} else {
			break
		}
	}
	// build hierarchy from flat list
	ws.Items = buildDataHierarchy(items)
}

func buildDataHierarchy(items []*DataItem) []*DataItem {
	if len(items) == 0 {
		return nil
	}
	// maintain stack with sentinel root at level 0
	stack := []*DataItem{{Level: 0, Name: "<root>"}}
	var roots []*DataItem

	for _, item := range items {
		// pop stack until we find parent
		for len(stack) > 0 && stack[len(stack)-1].Level >= item.Level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, item)
		}
		stack = append(stack, item)
		if item.Level == 1 {
			roots = append(roots, item)
		}
	}
	return roots
}

func (p *Parser) parseDataItem() *DataItem {
	// Level number — capture its position
	tok := p.advance()
	item := &DataItem{Line: tok.Line, Column: tok.Column}

	level, err := strconv.Atoi(tok.Literal)
	if err != nil {
		p.errorAt(tok, "invalid level number: "+tok.Literal)
		return nil
	}
	item.Level = level

	// Name — accept any word-like token (COBOL allows reserved words as data names)
	tok = p.peek()
	if tok.Type == lexer.IDENTIFIER || (tok.Type != lexer.INTEGER_LITERAL &&
		tok.Type != lexer.PERIOD && tok.Type != lexer.EOF && tok.Literal != "" &&
		unicode.IsLetter(rune(tok.Literal[0]))) {
		item.Name = p.advance().Literal
	} else {
		p.error("expected data item name")
		return item
	}

	// Parse clauses until period or next level number
	for !p.atEnd() {
		if p.peek().Type == lexer.PERIOD {
			p.advance()
			break
		}
		if p.peek().Type == lexer.INTEGER_LITERAL {
			// next data item, stop (don't consume period yet — let caller handle)
			break
		}

		switch p.peek().Type {
		case lexer.PIC, lexer.PICTURE:
			item.Picture = p.parsePictureClause()
		case lexer.OCCURS:
			item.Occurs = p.parseOccursClause()
		case lexer.VALUE:
			item.Value = p.parseValueClause()
		default:
			// skip unknown clause tokens (e.g., REDEFINES for future)
			p.advance()
		}
	}

	return item
}

func (p *Parser) parsePictureClause() *Picture {
	p.advance() // consume PIC/PICTURE
	pic := &Picture{}

	tok := p.peek()
	if tok.Type == lexer.IDENTIFIER && tok.Literal == "X" {
		pic.Type = PicX
		p.advance()
	} else if tok.Type == lexer.INTEGER_LITERAL && tok.Literal == "9" {
		pic.Type = Pic9
		p.advance()
	} else {
		p.errorAt(tok, "expected picture character (X or 9)")
		return pic
	}

	// Optional size in parentheses
	if p.match(lexer.LPAREN) {
		sizeTok, err := p.expect(lexer.INTEGER_LITERAL)
		if err == nil {
			pic.Size, _ = strconv.Atoi(sizeTok.Literal)
		}
		p.expect(lexer.RPAREN)
	} else {
		pic.Size = 1
	}

	// Optional COMP
	if p.match(lexer.COMP) {
		pic.IsComp = true
	}

	return pic
}

func (p *Parser) parseOccursClause() int {
	p.advance() // consume OCCURS
	count := 1
	if p.peek().Type == lexer.INTEGER_LITERAL {
		tok := p.advance()
		count, _ = strconv.Atoi(tok.Literal)
	}
	p.match(lexer.TIMES)
	return count
}

func (p *Parser) parseValueClause() string {
	p.advance() // consume VALUE
	// Could be string literal or integer literal
	if p.peek().Type == lexer.STRING_LITERAL {
		return p.advance().Literal
	}
	if p.peek().Type == lexer.INTEGER_LITERAL {
		return p.advance().Literal
	}
	p.error("expected value literal")
	return ""
}

// --- Procedure Division ---

func (p *Parser) parseProcedureDivision(prog *Program) {
	p.expect(lexer.PROCEDURE)
	p.expect(lexer.DIVISION)
	p.match(lexer.PERIOD)

	div := &ProcedureDivision{}
	for !p.atEnd() {
		if p.peek().Type == lexer.IDENTIFICATION || p.peek().Type == lexer.DATA ||
			p.peek().Type == lexer.EOF {
			break
		}
		para := p.parseParagraph()
		if para != nil {
			div.Paragraphs = append(div.Paragraphs, para)
		} else {
			break
		}
	}
	prog.ProcedureDivision = div
}

func (p *Parser) parseParagraph() *Paragraph {
	if p.peek().Type != lexer.IDENTIFIER {
		// Might be a bare statement without a paragraph name
		// Create an anonymous paragraph
		stmts := p.parseStatements()
		if len(stmts) > 0 {
			return &Paragraph{Name: "<anonymous>", Statements: stmts}
		}
		return nil
	}

	// Check if this IDENTIFIER is a paragraph name (followed by period)
	nameTok := p.peek()
	// Save position to try paragraph name
	// If next token after IDENTIFIER is PERIOD, it's a paragraph name
	if p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == lexer.PERIOD {
		p.advance() // consume IDENTIFIER
		p.advance() // consume PERIOD
		para := &Paragraph{Name: nameTok.Literal}
		para.Statements = p.parseStatements()
		return para
	}

	// Not a paragraph name, maybe statements without paragraph header
	stmts := p.parseStatements()
	if len(stmts) > 0 {
		return &Paragraph{Name: "<anonymous>", Statements: stmts}
	}
	return nil
}

// --- Statement parsing ---

func (p *Parser) parseStatements() []Statement {
	var stmts []Statement
	for !p.atEnd() {
		if p.isStatementEnd() || p.isScopeTerminator() {
			break
		}
		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
		// Consume trailing period if present
		if p.match(lexer.PERIOD) {
			// In COBOL, a period can terminate one or more statements.
			// If the next token is a new paragraph (IDENTIFIER followed by PERIOD), stop.
			// Otherwise, if it's another statement, continue in the same paragraph.
			if !p.atEnd() && p.peek().Type == lexer.IDENTIFIER &&
				p.pos+1 < len(p.tokens) && p.tokens[p.pos+1].Type == lexer.PERIOD {
				break
			}
			if p.atEnd() || p.peek().Type == lexer.EOF {
				break
			}
			// Continue parsing statements in the same paragraph
		}
	}
	return stmts
}

func (p *Parser) isStatementEnd() bool {
	if p.atEnd() {
		return true
	}
	switch p.peek().Type {
	case lexer.PERIOD, lexer.EOF:
		return true
	}
	return false
}

func (p *Parser) isScopeTerminator() bool {
	switch p.peek().Type {
	case lexer.ELSE, lexer.END_IF, lexer.WHEN, lexer.END_EVALUATE,
		lexer.END_HTTP, lexer.END_PERFORM,
		lexer.END_STRING, lexer.END_UNSTRING:
		return true
	}
	return false
}

func (p *Parser) isStatementStart() bool {
	switch p.peek().Type {
	case lexer.MOVE, lexer.COMPUTE, lexer.DISPLAY, lexer.PERFORM,
		lexer.IF, lexer.EVALUATE, lexer.STOP, lexer.INITIALIZE, lexer.ACCEPT,
		lexer.STRING, lexer.UNSTRING,
		lexer.HTTP_GET, lexer.HTTP_POST, lexer.HTTP_PUT,
		lexer.HTTP_PATCH, lexer.HTTP_DELETE,
		lexer.HTTP_LISTEN, lexer.HTTP_RESPOND:
		return true
	}
	return false
}

func (p *Parser) parseStatement() Statement {
	if p.atEnd() || p.isScopeTerminator() {
		return nil
	}

	switch p.peek().Type {
	case lexer.MOVE:
		return p.parseMoveStatement()
	case lexer.COMPUTE:
		return p.parseComputeStatement()
	case lexer.DISPLAY:
		return p.parseDisplayStatement()
	case lexer.PERFORM:
		return p.parsePerformStatement()
	case lexer.IF:
		return p.parseIfStatement()
	case lexer.EVALUATE:
		return p.parseEvaluateStatement()
	case lexer.STOP:
		return p.parseStopRunStatement()
	case lexer.INITIALIZE:
		return p.parseInitializeStatement()
	case lexer.ACCEPT:
		return p.parseAcceptStatement()
	case lexer.STRING:
		return p.parseStringStatement()
	case lexer.UNSTRING:
		return p.parseUnstringStatement()
	case lexer.HTTP_GET:
		return p.parseHttpGetStatement()
	case lexer.HTTP_POST:
		return p.parseHttpPostStatement()
	case lexer.HTTP_PUT:
		return p.parseHttpPutStatement()
	case lexer.HTTP_PATCH:
		return p.parseHttpPatchStatement()
	case lexer.HTTP_DELETE:
		return p.parseHttpDeleteStatement()
	case lexer.HTTP_LISTEN:
		return p.parseHttpListenStatement()
	case lexer.HTTP_RESPOND:
		return p.parseHttpRespondStatement()
	default:
		p.error("unexpected token in statement: " + p.peek().Literal)
		p.advance()
		return nil
	}
}

func (p *Parser) parseMoveStatement() *Move {
	tok := p.advance() // consume MOVE
	stmt := &Move{Line: tok.Line, Col: tok.Column}
	stmt.From = p.parseExpression()

	if p.match(lexer.TO) {
		if p.peek().Type == lexer.IDENTIFIER {
			stmt.To = p.advance().Literal
		} else {
			p.error("expected identifier after TO")
		}
	} else {
		p.error("expected TO in MOVE statement")
	}

	return stmt
}

func (p *Parser) parseComputeStatement() *Compute {
	tok := p.advance() // consume COMPUTE
	stmt := &Compute{Line: tok.Line, Col: tok.Column}

	if p.peek().Type == lexer.IDENTIFIER {
		stmt.Target = p.advance().Literal
	} else {
		p.error("expected identifier in COMPUTE")
	}

	if p.match(lexer.EQUALS) {
		stmt.Expr = p.parseExpression()
	} else {
		p.error("expected = in COMPUTE statement")
	}

	return stmt
}

func (p *Parser) parseDisplayStatement() *Display {
	tok := p.advance() // consume DISPLAY
	stmt := &Display{Line: tok.Line, Col: tok.Column}

	// Parse expressions until terminator
	for !p.atEnd() && !p.isScopeTerminator() &&
		p.peek().Type != lexer.PERIOD &&
		!p.isStatementStart() {
		expr := p.parseExpression()
		if expr != nil {
			stmt.Items = append(stmt.Items, expr)
		} else {
			break
		}
	}

	return stmt
}

func (p *Parser) parsePerformStatement() *Perform {
	tok := p.advance() // consume PERFORM
	stmt := &Perform{Line: tok.Line, Col: tok.Column}

	// PERFORM VARYING ...
	if p.match(lexer.VARYING) {
		return p.parsePerformVarying(stmt)
	}

	// PERFORM UNTIL condition ... END-PERFORM
	if p.match(lexer.UNTIL) {
		stmt.Until = p.parseExpression()
		stmt.Body = p.parsePerformBody()
		return stmt
	}

	// PERFORM paragraph-name ...
	if p.peek().Type == lexer.IDENTIFIER {
		stmt.Paragraph = p.advance().Literal
	} else {
		p.error("expected paragraph name, VARYING, or UNTIL in PERFORM")
		return stmt
	}

	// Optional TIMES clause
	if p.match(lexer.INTEGER_LITERAL) {
		tok := p.previous()
		val, _ := strconv.Atoi(tok.Literal)
		stmt.Times = &IntegerLiteralExpr{Value: val}
		if p.peek().Type == lexer.TIMES {
			p.advance() // consume TIMES
		}
		return stmt
	}

	return stmt
}

func (p *Parser) parsePerformBody() []Statement {
	var body []Statement
	for !p.atEnd() {
		if p.match(lexer.END_PERFORM) {
			break
		}
		if p.peek().Type == lexer.PERIOD {
			break
		}
		s := p.parseStatement()
		if s != nil {
			body = append(body, s)
		} else {
			break
		}
		if p.match(lexer.PERIOD) {
			break
		}
	}
	return body
}

func (p *Parser) parsePerformVarying(stmt *Perform) *Perform {
	vp := &VaryingPhrase{}

	// VARYING variable
	if p.peek().Type == lexer.IDENTIFIER {
		vp.Variable = p.advance().Literal
	} else {
		p.error("expected variable name after VARYING")
		return stmt
	}

	// FROM expression
	if p.match(lexer.FROM) {
		vp.From = p.parseExpression()
	} else {
		p.error("expected FROM in PERFORM VARYING")
		return stmt
	}

	// BY expression
	if p.match(lexer.BY) {
		vp.By = p.parseExpression()
	} else {
		p.error("expected BY in PERFORM VARYING")
		return stmt
	}

	// UNTIL condition
	if p.match(lexer.UNTIL) {
		vp.Until = p.parseExpression()
	} else {
		p.error("expected UNTIL in PERFORM VARYING")
		return stmt
	}

	stmt.Varying = vp
	stmt.Body = p.parsePerformBody()
	return stmt
}

func (p *Parser) parseIfStatement() *If {
	tok := p.advance() // consume IF
	stmt := &If{Line: tok.Line, Col: tok.Column}

	// Parse condition (simple expression for v0.1)
	stmt.Condition = p.parseExpression()

	// Parse then-body statements (stop at ELSE, END-IF, or PERIOD)
	stmt.ThenBody = p.parseStatementsUntil(lexer.ELSE, lexer.END_IF)

	// Check for ELSE
	if p.match(lexer.ELSE) {
		stmt.ElseBody = p.parseStatementsUntil(lexer.END_IF)
	}

	// Expect END-IF
	if !p.match(lexer.END_IF) {
		p.error("expected END-IF")
	}

	return stmt
}

func (p *Parser) parseStatementsUntil(stopTokens ...lexer.TokenType) []Statement {
	var stmts []Statement
	for !p.atEnd() {
		tok := p.peek()
		if tok.Type == lexer.PERIOD {
			break
		}
		isStop := false
		for _, stop := range stopTokens {
			if tok.Type == stop {
				isStop = true
				break
			}
		}
		if isStop {
			break
		}
		stmt := p.parseStatement()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
		// Consume period if present within body (sentence-level period)
		if p.match(lexer.PERIOD) {
			break
		}
	}
	return stmts
}

func (p *Parser) parseEvaluateStatement() *Evaluate {
	tok := p.advance() // consume EVALUATE
	stmt := &Evaluate{Line: tok.Line, Col: tok.Column}

	// Subject expression
	stmt.Subject = p.parseExpression()

	// Parse WHEN clauses
	for !p.atEnd() && p.peek().Type != lexer.END_EVALUATE {
		if p.match(lexer.WHEN) {
			if p.match(lexer.OTHER) {
				// WHEN OTHER
				stmt.WhenOther = p.parseStatementsUntil(lexer.WHEN, lexer.END_EVALUATE)
				// Need to continue loop to check for END-EVALUATE
				continue
			}
			// WHEN value
			wc := WhenClause{}
			wc.Values = append(wc.Values, p.parseExpression())
			wc.Body = p.parseStatementsUntil(lexer.WHEN, lexer.END_EVALUATE)
			stmt.WhenClauses = append(stmt.WhenClauses, wc)
		} else {
			// Skip unexpected tokens
			if p.peek().Type == lexer.END_EVALUATE {
				break
			}
			p.advance()
		}
	}

	if !p.match(lexer.END_EVALUATE) {
		p.error("expected END-EVALUATE")
	}

	return stmt
}

func (p *Parser) parseStopRunStatement() *StopRun {
	tok := p.advance() // consume STOP
	if !p.match(lexer.RUN) {
		p.error("expected RUN after STOP")
	}
	return &StopRun{Line: tok.Line, Col: tok.Column}
}

func (p *Parser) parseInitializeStatement() *Initialize {
	tok := p.advance() // consume INITIALIZE
	stmt := &Initialize{Line: tok.Line, Col: tok.Column}

	for !p.atEnd() && !p.isScopeTerminator() &&
		p.peek().Type != lexer.PERIOD &&
		!p.isStatementStart() {
		if p.peek().Type == lexer.IDENTIFIER {
			stmt.Items = append(stmt.Items, p.advance().Literal)
		} else {
			break
		}
	}

	return stmt
}

func (p *Parser) parseAcceptStatement() *Accept {
	tok := p.advance()
	stmt := &Accept{Line: tok.Line, Col: tok.Column}
	if p.peek().Type == lexer.IDENTIFIER {
		stmt.Name = p.advance().Literal
	} else {
		p.error("expected identifier after ACCEPT")
	}
	return stmt
}

func (p *Parser) parseStringStatement() *StringStmt {
	tok := p.advance() // consume STRING
	stmt := &StringStmt{Line: tok.Line, Col: tok.Column}

	// Parse sending fields: expr DELIMITED BY (SIZE / SPACE / expr) ...
	for {
		if p.atEnd() || p.isScopeTerminator() {
			break
		}
		source := p.parseExpression()
		if source == nil {
			break
		}
		if !p.match(lexer.DELIMITED) {
			p.error("expected DELIMITED BY in STRING")
			break
		}
		if !p.match(lexer.BY) {
			p.error("expected BY after DELIMITED in STRING")
			break
		}
		delim := StringDelim{}
		if p.match(lexer.SIZE) {
			delim.Type = DelimBySize
		} else if p.match(lexer.SPACE) {
			delim.Type = DelimBySpace
		} else {
			delim.Type = DelimByIdentifier
			delim.Value = p.parseExpression()
			if delim.Value == nil {
				p.error("expected delimiter in STRING")
				break
			}
		}
		stmt.Sending = append(stmt.Sending, StringSendingField{
			Source:    source,
			Delimiter: delim,
		})
		if p.atEnd() || p.isScopeTerminator() {
			break
		}
		// Allow chaining without separator
		if p.peek().Type == lexer.INTO {
			break
		}
	}

	// INTO destination
	if !p.match(lexer.INTO) {
		p.error("expected INTO in STRING")
	}
	if p.peek().Type == lexer.IDENTIFIER {
		stmt.Into = p.advance().Literal
	} else {
		p.error("expected identifier after INTO")
	}

	// Optional WITH POINTER
	if p.match(lexer.POINTER) {
		if p.peek().Type == lexer.IDENTIFIER {
			stmt.Pointer = p.advance().Literal
		} else {
			p.error("expected identifier after POINTER")
		}
	}

	// Optional ON OVERFLOW / NOT ON OVERFLOW
	if p.match(lexer.ON) {
		if p.match(lexer.OVERFLOW) {
			stmt.OnOverflow = p.parseStatementsUntil(lexer.END_STRING, lexer.PERIOD)
		}
	}
	if p.match(lexer.NOT) {
		if p.match(lexer.ON) {
			if p.match(lexer.OVERFLOW) {
				stmt.NotOnOverflow = p.parseStatementsUntil(lexer.END_STRING, lexer.PERIOD)
			}
		}
	}

	// Expect END-STRING
	if !p.match(lexer.END_STRING) {
		p.error("expected END-STRING")
	}

	return stmt
}

func (p *Parser) parseUnstringStatement() *UnstringStmt {
	tok := p.advance() // consume UNSTRING
	stmt := &UnstringStmt{Line: tok.Line, Col: tok.Column}

	// Source expression
	stmt.Source = p.parseExpression()
	if stmt.Source == nil {
		p.error("expected source in UNSTRING")
	}

	// Optional DELIMITED BY [ALL] delimiter [OR [ALL] delimiter ...]
	if p.match(lexer.DELIMITED) {
		if !p.match(lexer.BY) {
			p.error("expected BY after DELIMITED in UNSTRING")
		}
		for {
			delim := StringDelim{}
			if p.match(lexer.SPACE) {
				delim.Type = DelimBySpace
			} else if p.match(lexer.SIZE) {
				p.error("SIZE not valid as UNSTRING delimiter")
				break
			} else {
				delim.Type = DelimByIdentifier
				delim.Value = p.parseExpression()
				if delim.Value == nil {
					p.error("expected delimiter in UNSTRING")
					break
				}
			}
			stmt.Delimiters = append(stmt.Delimiters, delim)
			if !p.match(lexer.OR) {
				break
			}
		}
	}

	// INTO dest1 [DELIMITER IN delim1] [COUNT IN cnt1] ...
	if !p.match(lexer.INTO) {
		p.error("expected INTO in UNSTRING")
	}
	for {
		if p.atEnd() || p.isScopeTerminator() {
			break
		}
		if p.peek().Type != lexer.IDENTIFIER {
			break
		}
		field := UnstringReceivingField{
			Destination: p.advance().Literal,
		}
		if p.match(lexer.DELIMITER) {
			if p.match(lexer.IN) {
				if p.peek().Type == lexer.IDENTIFIER {
					field.DelimiterIn = p.advance().Literal
				} else {
					p.error("expected identifier after DELIMITER IN")
				}
			} else {
				p.error("expected IN after DELIMITER")
			}
		}
		if p.match(lexer.COUNT) {
			if p.match(lexer.IN) {
				if p.peek().Type == lexer.IDENTIFIER {
					field.CountIn = p.advance().Literal
				} else {
					p.error("expected identifier after COUNT IN")
				}
			} else {
				p.error("expected IN after COUNT")
			}
		}
		stmt.Into = append(stmt.Into, field)
	}

	// Optional WITH POINTER
	if p.match(lexer.POINTER) {
		if p.peek().Type == lexer.IDENTIFIER {
			stmt.Pointer = p.advance().Literal
		} else {
			p.error("expected identifier after POINTER")
		}
	}

	// Optional TALLYING IN
	if p.match(lexer.TALLYING) {
		if p.match(lexer.IN) {
			if p.peek().Type == lexer.IDENTIFIER {
				stmt.Tallying = p.advance().Literal
			} else {
				p.error("expected identifier after TALLYING IN")
			}
		} else {
			p.error("expected IN after TALLYING")
		}
	}

	// Optional ON OVERFLOW / NOT ON OVERFLOW
	if p.match(lexer.ON) {
		if p.match(lexer.OVERFLOW) {
			stmt.OnOverflow = p.parseStatementsUntil(lexer.END_UNSTRING, lexer.PERIOD)
		}
	}
	if p.match(lexer.NOT) {
		if p.match(lexer.ON) {
			if p.match(lexer.OVERFLOW) {
				stmt.NotOnOverflow = p.parseStatementsUntil(lexer.END_UNSTRING, lexer.PERIOD)
			}
		}
	}

	// Expect END-UNSTRING
	if !p.match(lexer.END_UNSTRING) {
		p.error("expected END-UNSTRING")
	}

	return stmt
}

// --- HTTP verb parsers ---

func (p *Parser) parseHttpGetStatement() *HttpGet {
	tok := p.advance() // consume HTTP-GET
	stmt := &HttpGet{Line: tok.Line, Col: tok.Column}

	stmt.URL = p.parseExpression()
	p.parseHttpCommonPost(stmt, false)
	return stmt
}

func (p *Parser) parseHttpPostStatement() *HttpPost {
	tok := p.advance() // consume HTTP-POST
	stmt := &HttpPost{Line: tok.Line, Col: tok.Column}

	stmt.URL = p.parseExpression()
	p.parseHttpCommon(stmt)
	return stmt
}

func (p *Parser) parseHttpPutStatement() *HttpPut {
	tok := p.advance() // consume HTTP-PUT
	stmt := &HttpPut{Line: tok.Line, Col: tok.Column}

	stmt.URL = p.parseExpression()
	p.parseHttpCommon(stmt)
	return stmt
}

func (p *Parser) parseHttpPatchStatement() *HttpPatch {
	tok := p.advance() // consume HTTP-PATCH
	stmt := &HttpPatch{Line: tok.Line, Col: tok.Column}

	stmt.URL = p.parseExpression()
	p.parseHttpCommon(stmt)
	return stmt
}

func (p *Parser) parseHttpDeleteStatement() *HttpDelete {
	tok := p.advance() // consume HTTP-DELETE
	stmt := &HttpDelete{Line: tok.Line, Col: tok.Column}

	stmt.URL = p.parseExpression()
	p.parseHttpCommonDelete(stmt)
	return stmt
}

// parseHttpCommon handles SENDING, MAPPING, HEADERS, GIVING, STATUS, ON EXCEPTION for POST/PUT/PATCH
func (p *Parser) parseHttpCommon(stmt interface{}) {
	// We need to handle clauses in any order
	for !p.atEnd() {
		switch p.peek().Type {
		case lexer.SENDING:
			p.advance()
			expr := p.parseExpression()
			switch s := stmt.(type) {
			case *HttpPost:
				s.Sending = expr
			case *HttpPut:
				s.Sending = expr
			case *HttpPatch:
				s.Sending = expr
			}
		case lexer.MAPPING:
			p.advance()
			if p.peek().Type == lexer.IDENTIFIER {
				name := p.advance().Literal
				switch s := stmt.(type) {
				case *HttpPost:
					s.Mapping = &name
				case *HttpPut:
					s.Mapping = &name
				case *HttpPatch:
					s.Mapping = &name
				case *HttpGet:
					s.Mapping = &name
				case *HttpDelete:
					s.Mapping = &name
				}
			}
	case lexer.HEADERS:
		p.advance()
		hp := &HeadersPhrase{}
		if p.match(lexer.IDENTIFIER) {
			name := p.previous().Literal
			if p.match(lexer.IDENTIFIER) {
				hp.Count = name
				hp.Group = p.previous().Literal
			} else {
				hp.Group = name
			}
		}
		switch s := stmt.(type) {
		case *HttpPost:
			s.Headers = hp
		case *HttpPut:
			s.Headers = hp
		case *HttpPatch:
			s.Headers = hp
		case *HttpGet:
			s.Headers = hp
		case *HttpDelete:
			s.Headers = hp
		}
		case lexer.GIVING:
			p.advance()
			if p.peek().Type == lexer.IDENTIFIER {
				name := p.advance().Literal
				switch s := stmt.(type) {
				case *HttpPost:
					s.Giving = &name
				case *HttpPut:
					s.Giving = &name
				case *HttpPatch:
					s.Giving = &name
				case *HttpGet:
					s.Giving = &name
				case *HttpDelete:
					s.Giving = &name
				}
			}
		case lexer.STATUS:
			p.advance()
			if p.peek().Type == lexer.IDENTIFIER {
				switch s := stmt.(type) {
				case *HttpPost:
					s.Status = p.advance().Literal
				case *HttpPut:
					s.Status = p.advance().Literal
				case *HttpPatch:
					s.Status = p.advance().Literal
				case *HttpGet:
					s.Status = p.advance().Literal
				case *HttpDelete:
					s.Status = p.advance().Literal
				default:
					p.advance()
				}
			}
		case lexer.ON:
			p.advance()
			if p.match(lexer.EXCEPTION) {
				exceptions := p.parseStatementsUntil(lexer.NOT, lexer.END_HTTP)
				switch s := stmt.(type) {
				case *HttpPost:
					s.OnException = exceptions
				case *HttpPut:
					s.OnException = exceptions
				case *HttpPatch:
					s.OnException = exceptions
				case *HttpGet:
					s.OnException = exceptions
				case *HttpDelete:
					s.OnException = exceptions
				}
			}
			if p.match(lexer.NOT) {
				if p.match(lexer.EXCEPTION) {
					notExceptions := p.parseStatementsUntil(lexer.END_HTTP)
					switch s := stmt.(type) {
					case *HttpPost:
						s.NotOnException = notExceptions
					case *HttpPut:
						s.NotOnException = notExceptions
					case *HttpPatch:
						s.NotOnException = notExceptions
					case *HttpGet:
						s.NotOnException = notExceptions
					case *HttpDelete:
						s.NotOnException = notExceptions
					}
				}
			}
		case lexer.END_HTTP:
			p.advance()
			return
		default:
			return
		}
	}
}

// parseHttpCommonPost handles HTTP-GET (no SENDING) and HTTP-DELETE
func (p *Parser) parseHttpCommonPost(stmt *HttpGet, _ bool) {
	// Same as parseHttpCommon but without SENDING
	for !p.atEnd() {
		switch p.peek().Type {
		case lexer.MAPPING:
			p.advance()
			if p.peek().Type == lexer.IDENTIFIER {
				name := p.advance().Literal
				stmt.Mapping = &name
			}
		case lexer.HEADERS:
			p.advance()
			hp := &HeadersPhrase{}
			if p.match(lexer.IDENTIFIER) {
				name := p.previous().Literal
				if p.match(lexer.IDENTIFIER) {
					hp.Count = name
					hp.Group = p.previous().Literal
				} else {
					hp.Group = name
				}
			}
			stmt.Headers = hp
		case lexer.GIVING:
			p.advance()
			if p.peek().Type == lexer.IDENTIFIER {
				name := p.advance().Literal
				stmt.Giving = &name
			}
		case lexer.STATUS:
			p.advance()
			if p.peek().Type == lexer.IDENTIFIER {
				stmt.Status = p.advance().Literal
			}
		case lexer.ON:
			p.advance()
			if p.match(lexer.EXCEPTION) {
				stmt.OnException = p.parseStatementsUntil(lexer.NOT, lexer.END_HTTP)
			}
			if p.match(lexer.NOT) {
				if p.match(lexer.EXCEPTION) {
					stmt.NotOnException = p.parseStatementsUntil(lexer.END_HTTP)
				}
			}
		case lexer.END_HTTP:
			p.advance()
			return
		default:
			return
		}
	}
}

func (p *Parser) parseHttpCommonDelete(stmt *HttpDelete) {
	for !p.atEnd() {
		switch p.peek().Type {
		case lexer.MAPPING:
			p.advance()
			if p.peek().Type == lexer.IDENTIFIER {
				name := p.advance().Literal
				stmt.Mapping = &name
			}
		case lexer.HEADERS:
			p.advance()
			hp := &HeadersPhrase{}
			if p.match(lexer.IDENTIFIER) {
				name := p.previous().Literal
				if p.match(lexer.IDENTIFIER) {
					hp.Count = name
					hp.Group = p.previous().Literal
				} else {
					hp.Group = name
				}
			}
			stmt.Headers = hp
		case lexer.GIVING:
			p.advance()
			if p.peek().Type == lexer.IDENTIFIER {
				name := p.advance().Literal
				stmt.Giving = &name
			}
		case lexer.STATUS:
			p.advance()
			if p.peek().Type == lexer.IDENTIFIER {
				stmt.Status = p.advance().Literal
			}
		case lexer.ON:
			p.advance()
			if p.match(lexer.EXCEPTION) {
				stmt.OnException = p.parseStatementsUntil(lexer.NOT, lexer.END_HTTP)
			}
			if p.match(lexer.NOT) {
				if p.match(lexer.EXCEPTION) {
					stmt.NotOnException = p.parseStatementsUntil(lexer.END_HTTP)
				}
			}
		case lexer.END_HTTP:
			p.advance()
			return
		default:
			return
		}
	}
}

func (p *Parser) parseHttpListenStatement() *HttpListen {
	tok := p.advance() // consume HTTP-LISTEN
	stmt := &HttpListen{Line: tok.Line, Col: tok.Column}

	for !p.atEnd() {
		switch p.peek().Type {
		case lexer.PORT:
			p.advance()
			if p.peek().Type == lexer.INTEGER_LITERAL {
				valTok := p.advance()
				stmt.Port, _ = strconv.Atoi(valTok.Literal)
			} else {
				p.error("expected port number after PORT")
			}
		case lexer.MAPPING:
			p.advance()
			if p.peek().Type == lexer.IDENTIFIER {
				name := p.advance().Literal
				stmt.Mapping = &name
			}
		case lexer.HEADERS:
			p.advance()
			hp := &HeadersPhrase{}
			if p.match(lexer.IDENTIFIER) {
				name := p.previous().Literal
				if p.match(lexer.IDENTIFIER) {
					hp.Count = name
					hp.Group = p.previous().Literal
				} else {
					hp.Group = name
				}
			}
			stmt.Headers = hp
		case lexer.STATUS:
			p.advance()
			if p.peek().Type == lexer.IDENTIFIER {
				stmt.Status = p.advance().Literal
			}
		case lexer.NOT:
			p.advance()
			if p.match(lexer.ON) {
				if p.match(lexer.EXCEPTION) {
					stmt.NotOnException = p.parseStatementsUntil(lexer.ON, lexer.END_HTTP)
				}
			}
		case lexer.ON:
			p.advance()
			if p.match(lexer.EXCEPTION) {
				stmt.OnException = p.parseStatementsUntil(lexer.NOT, lexer.END_HTTP)
			}
			if p.match(lexer.NOT) {
				if p.match(lexer.EXCEPTION) {
					stmt.NotOnException = p.parseStatementsUntil(lexer.END_HTTP)
				}
			}
		case lexer.END_HTTP:
			p.advance()
			return stmt
		default:
			return stmt
		}
	}

	return stmt
}

func (p *Parser) parseHttpRespondStatement() *HttpRespond {
	tok := p.advance() // consume HTTP-RESPOND
	stmt := &HttpRespond{Line: tok.Line, Col: tok.Column}

	for !p.atEnd() {
		switch p.peek().Type {
		case lexer.STATUS:
			p.advance()
			if p.peek().Type == lexer.INTEGER_LITERAL {
				valTok := p.advance()
				stmt.Status, _ = strconv.Atoi(valTok.Literal)
			} else {
				p.error("expected status code after STATUS")
			}
		case lexer.BODY:
			p.advance()
			stmt.Body = p.parseExpression()
		case lexer.CONTENT_TYPE:
			p.advance()
			stmt.ContentType = p.parseExpression()
		case lexer.HEADERS:
			p.advance()
			hp := &HeadersPhrase{}
			if p.match(lexer.IDENTIFIER) {
				name := p.previous().Literal
				if p.match(lexer.IDENTIFIER) {
					hp.Count = name
					hp.Group = p.previous().Literal
				} else {
					hp.Group = name
				}
			}
			stmt.Headers = hp
		case lexer.MAPPING:
			p.advance()
			if p.peek().Type == lexer.IDENTIFIER {
				name := p.advance().Literal
				stmt.Mapping = &name
			}
		case lexer.NOT:
			p.advance()
			if p.match(lexer.ON) {
				if p.match(lexer.EXCEPTION) {
					stmt.NotOnException = p.parseStatementsUntil(lexer.ON, lexer.END_HTTP)
				}
			}
		case lexer.ON:
			p.advance()
			if p.match(lexer.EXCEPTION) {
				stmt.OnException = p.parseStatementsUntil(lexer.NOT, lexer.END_HTTP)
			}
			if p.match(lexer.NOT) {
				if p.match(lexer.EXCEPTION) {
					stmt.NotOnException = p.parseStatementsUntil(lexer.ON, lexer.END_HTTP)
				}
			}
		case lexer.END_HTTP:
			p.advance()
			return stmt
		default:
			return stmt
		}
	}

	return stmt
}

// --- Expression parsing ---

func (p *Parser) parseExpression() Expression {
	return p.parseComparison()
}

func (p *Parser) parseComparison() Expression {
	left := p.parseAddSub()
	for !p.atEnd() {
		matched := true
		switch {
		case p.match(lexer.GREATER):
			right := p.parseAddSub()
			if p.previous().Literal == ">=" {
				left = &BinaryExpr{Left: left, Operator: OpGe, Right: right}
			} else {
				left = &BinaryExpr{Left: left, Operator: OpGt, Right: right}
			}
		case p.match(lexer.LESS):
			right := p.parseAddSub()
			lit := p.previous().Literal
			if lit == "<=" {
				left = &BinaryExpr{Left: left, Operator: OpLe, Right: right}
			} else if lit == "<>" {
				left = &BinaryExpr{Left: left, Operator: OpLt, Right: right}
			} else {
				left = &BinaryExpr{Left: left, Operator: OpLt, Right: right}
			}
		case p.match(lexer.EQUALS):
			right := p.parseAddSub()
			left = &BinaryExpr{Left: left, Operator: OpEq, Right: right}
		default:
			matched = false
		}
		if !matched {
			break
		}
	}
	return left
}

func (p *Parser) parseAddSub() Expression {
	left := p.parseMulDiv()
	for !p.atEnd() {
		if p.match(lexer.PLUS) {
			right := p.parseMulDiv()
			left = &BinaryExpr{Left: left, Operator: OpAdd, Right: right}
		} else if p.match(lexer.MINUS) {
			right := p.parseMulDiv()
			left = &BinaryExpr{Left: left, Operator: OpSub, Right: right}
		} else {
			break
		}
	}
	return left
}

func (p *Parser) parseMulDiv() Expression {
	left := p.parsePrimary()
	for !p.atEnd() {
		if p.match(lexer.MULTIPLY) {
			right := p.parsePrimary()
			left = &BinaryExpr{Left: left, Operator: OpMul, Right: right}
		} else if p.match(lexer.DIVIDE) {
			right := p.parsePrimary()
			left = &BinaryExpr{Left: left, Operator: OpDiv, Right: right}
		} else {
			break
		}
	}
	return left
}

func (p *Parser) parsePrimary() Expression {
	if p.atEnd() {
		return nil
	}

	tok := p.peek()

	if tok.Type == lexer.INTEGER_LITERAL {
		p.advance()
		val, _ := strconv.Atoi(tok.Literal)
		return &IntegerLiteralExpr{Value: val}
	}

	if tok.Type == lexer.STRING_LITERAL {
		p.advance()
		return &StringLiteralExpr{Value: tok.Literal}
	}

	if tok.Type == lexer.IDENTIFIER {
		p.advance()
		return &IdentifierExpr{Name: tok.Literal, Line: tok.Line, Column: tok.Column}
	}

	if p.match(lexer.LPAREN) {
		expr := p.parseExpression()
		p.expect(lexer.RPAREN)
		return expr
	}

	return nil
}
