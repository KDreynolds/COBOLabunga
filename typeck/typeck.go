package typeck

import (
	"fmt"

	"github.com/KDreynolds/COBOLabunga/parser"
)

type SymType int

const (
	TypeAlphanumeric SymType = iota
	TypeNumeric
	TypeNumericComp
	TypeGroup
)

type Symbol struct {
	Name   string
	Type   SymType
	Size   int
	Level  int
	Line   int
	Column int
}

type Checker struct {
	symbols map[string]*Symbol
	errs    []error
}

func New() *Checker {
	return &Checker{
		symbols: make(map[string]*Symbol),
	}
}

func (c *Checker) Check(prog *parser.Program) []error {
	if prog.WorkingStorage != nil {
		c.buildSymbolTable(prog.WorkingStorage.Items)
	}

	if prog.ProcedureDivision != nil {
		for _, para := range prog.ProcedureDivision.Paragraphs {
			c.checkParagraph(para)
		}
	}

	return c.errs
}

// --- Symbol table building ---

func (c *Checker) buildSymbolTable(items []*parser.DataItem) {
	for _, item := range items {
		c.addSymbol(item)
	}
}

func (c *Checker) addSymbol(item *parser.DataItem) {
	sym := &Symbol{
		Name:   item.Name,
		Level:  item.Level,
		Line:   item.Line,
		Column: item.Column,
	}

	if len(item.Children) > 0 {
		sym.Type = TypeGroup
	} else if item.Picture != nil {
		switch item.Picture.Type {
		case parser.PicX:
			sym.Type = TypeAlphanumeric
		case parser.Pic9:
			if item.Picture.IsComp {
				sym.Type = TypeNumericComp
			} else {
				sym.Type = TypeNumeric
			}
		}
		sym.Size = item.Picture.Size
	}

	if existing, ok := c.symbols[item.Name]; ok {
		c.errs = append(c.errs, fmt.Errorf(
			"line %d:%d: duplicate symbol '%s' (first defined at line %d:%d)",
			item.Line, item.Column, item.Name, existing.Line, existing.Column))
	} else {
		c.symbols[item.Name] = sym
	}

	// Process children recursively
	if len(item.Children) > 0 {
		c.buildSymbolTable(item.Children)
	}
}

// --- Procedure Division validation ---

func (c *Checker) checkParagraph(para *parser.Paragraph) {
	for _, stmt := range para.Statements {
		c.checkStatement(stmt)
	}
}

func (c *Checker) checkStatement(stmt parser.Statement) {
	switch s := stmt.(type) {
	case *parser.Move:
		c.checkMove(s)
	case *parser.Compute:
		c.checkCompute(s)
	case *parser.Display:
		c.checkDisplay(s)
	case *parser.Perform:
		c.checkPerform(s)
	case *parser.If:
		c.checkIf(s)
	case *parser.Evaluate:
		c.checkEvaluate(s)
	case *parser.StopRun:
		// always valid
	case *parser.Initialize:
		c.checkInitialize(s)
	case *parser.Accept:
		c.checkAccept(s)
	case *parser.StringStmt:
		c.checkString(s)
	case *parser.UnstringStmt:
		c.checkUnstring(s)
	case *parser.HttpGet:
		c.checkHttpGet(s)
	case *parser.HttpPost:
		c.checkHttpPost(s)
	case *parser.HttpPut:
		c.checkHttpPut(s)
	case *parser.HttpPatch:
		c.checkHttpPatch(s)
	case *parser.HttpDelete:
		c.checkHttpDelete(s)
	case *parser.HttpListen:
		c.checkHttpListen(s)
	case *parser.HttpRespond:
		c.checkHttpRespond(s)
	}
}

func (c *Checker) checkMove(s *parser.Move) {
	fromType, _ := c.exprType(s.From)
	toSym := c.lookup(s.To, s.Line, s.Col)

	if toSym == nil {
		return // error already reported
	}

	if fromType != nil && !c.isMoveCompatible(fromType, toSym) {
		c.err("move type mismatch: cannot move %s to %s field '%s'",
			s.Line, s.Col, typeName(fromType), typeName(&Symbol{Type: toSym.Type}), s.To)
	}

	// Size mismatch warning: moving a larger source into a smaller target truncates
	if fromType != nil && toSym.Type != TypeGroup && fromType.Size > 0 && toSym.Size > 0 {
		if fromType.Size > toSym.Size*2 {
			c.err("size mismatch in MOVE: source size (%d) is more than double target size (%d), truncation may occur",
				s.Line, s.Col, fromType.Size, toSym.Size)
		}
	}
}

func (c *Checker) isMoveCompatible(from *Symbol, to *Symbol) bool {
	// In COBOL, most moves are valid; we flag obvious mismatches
	if to.Type == TypeAlphanumeric {
		return from.Type == TypeAlphanumeric || from.Type == TypeNumeric || from.Type == TypeNumericComp
	}
	if to.Type == TypeNumeric || to.Type == TypeNumericComp {
		return from.Type == TypeNumeric || from.Type == TypeNumericComp || from.Type == TypeAlphanumeric
	}
	return true
}

func (c *Checker) checkCompute(s *parser.Compute) {
	target := c.lookup(s.Target, s.Line, s.Col)
	if target == nil {
		return
	}
	if target.Type == TypeAlphanumeric || target.Type == TypeGroup {
		c.err("compute target '%s' must be numeric (PIC 9), got %s",
			s.Line, s.Col, s.Target, typeName(target))
	}
	c.checkExpr(s.Expr)
}

func (c *Checker) checkDisplay(s *parser.Display) {
	for _, item := range s.Items {
		c.checkExpr(item)
	}
}

func (c *Checker) checkInitialize(s *parser.Initialize) {
	for _, name := range s.Items {
		c.lookup(name, s.Line, s.Col)
	}
}

func (c *Checker) checkAccept(s *parser.Accept) {
	c.lookup(s.Name, s.Line, s.Col)
}

func (c *Checker) checkPerform(s *parser.Perform) {
	if s.Varying != nil {
		vp := s.Varying
		// Varying variable must exist and be numeric
		vSym := c.lookup(vp.Variable, s.Line, s.Col)
		if vSym != nil {
			if vSym.Type == TypeAlphanumeric {
				c.err("varying variable '%s' must be numeric", s.Line, s.Col, vp.Variable)
			}
		}
		// FROM, BY, UNTIL expressions are checked
		c.checkExpr(vp.From)
		c.checkExpr(vp.By)
		c.checkExpr(vp.Until)
		// Check inline body
		for _, stmt := range s.Body {
			c.checkStatement(stmt)
		}
	} else if s.Paragraph != "" {
		// Paragraph names are in the procedure division, not data division
		// For v0.1, we just trust they exist (forward reference is valid in COBOL)
	}
}

func (c *Checker) checkIf(s *parser.If) {
	c.checkExpr(s.Condition)
	for _, stmt := range s.ThenBody {
		c.checkStatement(stmt)
	}
	for _, stmt := range s.ElseBody {
		c.checkStatement(stmt)
	}
}

func (c *Checker) checkEvaluate(s *parser.Evaluate) {
	c.checkExpr(s.Subject)
	for _, wc := range s.WhenClauses {
		for _, val := range wc.Values {
			c.checkExpr(val)
		}
		for _, stmt := range wc.Body {
			c.checkStatement(stmt)
		}
	}
	for _, stmt := range s.WhenOther {
		c.checkStatement(stmt)
	}
}

func (c *Checker) checkString(s *parser.StringStmt) {
	for _, f := range s.Sending {
		c.checkExpr(f.Source)
		if f.Delimiter.Type == parser.DelimByIdentifier && f.Delimiter.Value != nil {
			c.checkExpr(f.Delimiter.Value)
		}
	}
	if s.Into != "" {
		sym := c.lookup(s.Into, s.Line, s.Col)
		if sym != nil && sym.Type != TypeAlphanumeric {
			c.err("string into target '%s' must be alphanumeric (PIC X)", s.Line, s.Col, s.Into)
		}
	}
	if s.Pointer != "" {
		sym := c.lookup(s.Pointer, s.Line, s.Col)
		if sym != nil && sym.Type != TypeNumericComp {
			c.err("string pointer '%s' must be PIC 9(n) COMP", s.Line, s.Col, s.Pointer)
		}
	}
	for _, stmt := range s.OnOverflow {
		c.checkStatement(stmt)
	}
	for _, stmt := range s.NotOnOverflow {
		c.checkStatement(stmt)
	}
}

func (c *Checker) checkUnstring(s *parser.UnstringStmt) {
	c.checkExpr(s.Source)
	for _, f := range s.Into {
		sym := c.lookup(f.Destination, s.Line, s.Col)
		if sym != nil && sym.Type != TypeAlphanumeric && sym.Type != TypeNumeric {
			c.err("unstring target '%s' must be alphanumeric (PIC X)", s.Line, s.Col, f.Destination)
		}
		if f.DelimiterIn != "" {
			dSym := c.lookup(f.DelimiterIn, s.Line, s.Col)
			if dSym != nil && dSym.Type != TypeAlphanumeric {
				c.err("delimiter field '%s' must be alphanumeric (PIC X)", s.Line, s.Col, f.DelimiterIn)
			}
		}
		if f.CountIn != "" {
			cSym := c.lookup(f.CountIn, s.Line, s.Col)
			if cSym != nil && cSym.Type != TypeNumeric && cSym.Type != TypeNumericComp {
				c.err("count field '%s' must be numeric", s.Line, s.Col, f.CountIn)
			}
		}
	}
	if s.Pointer != "" {
		sym := c.lookup(s.Pointer, s.Line, s.Col)
		if sym != nil && sym.Type != TypeNumericComp {
			c.err("unstring pointer '%s' must be PIC 9(n) COMP", s.Line, s.Col, s.Pointer)
		}
	}
	if s.Tallying != "" {
		sym := c.lookup(s.Tallying, s.Line, s.Col)
		if sym != nil && sym.Type != TypeNumeric && sym.Type != TypeNumericComp {
			c.err("tallying field '%s' must be numeric", s.Line, s.Col, s.Tallying)
		}
	}
	for _, stmt := range s.OnOverflow {
		c.checkStatement(stmt)
	}
	for _, stmt := range s.NotOnOverflow {
		c.checkStatement(stmt)
	}
}

func (c *Checker) checkHttpGet(s *parser.HttpGet) {
	c.checkExpr(s.URL)
	if s.Giving != nil && s.Mapping != nil {
		c.err("giving and mapping are mutually exclusive", s.Line, s.Col)
	}
	if s.Giving != nil {
		c.lookup(*s.Giving, s.Line, s.Col)
	}
	if s.Mapping != nil {
		if sym := c.lookup(*s.Mapping, s.Line, s.Col); sym != nil && sym.Type != TypeGroup {
			c.err("mapping target '%s' is not a group item", s.Line, s.Col, *s.Mapping)
		}
	}
	if s.Status != "" {
		if sym := c.lookup(s.Status, s.Line, s.Col); sym != nil && sym.Type != TypeNumericComp {
			c.err("status field '%s' must be PIC 9(n) COMP", s.Line, s.Col, s.Status)
		}
	}
	if s.Headers != nil {
		c.checkHeaders(s.Headers, s.Line, s.Col)
	}
	c.checkExceptionBlocks(s.OnException, s.NotOnException)
}

func (c *Checker) checkHttpPost(s *parser.HttpPost) {
	c.checkExpr(s.URL)
	if s.Sending != nil && s.Mapping != nil {
		c.err("sending and mapping are mutually exclusive", s.Line, s.Col)
	}
	if s.Mapping != nil {
		if sym := c.lookup(*s.Mapping, s.Line, s.Col); sym != nil && sym.Type != TypeGroup {
			c.err("mapping target '%s' is not a group item", s.Line, s.Col, *s.Mapping)
		}
	}
	if s.Giving != nil {
		c.lookup(*s.Giving, s.Line, s.Col)
	}
	if s.Status != "" {
		if sym := c.lookup(s.Status, s.Line, s.Col); sym != nil && sym.Type != TypeNumericComp {
			c.err("status field '%s' must be PIC 9(n) COMP", s.Line, s.Col, s.Status)
		}
	}
	if s.Headers != nil {
		c.checkHeaders(s.Headers, s.Line, s.Col)
	}
	c.checkExceptionBlocks(s.OnException, s.NotOnException)
}

func (c *Checker) checkHttpPut(s *parser.HttpPut) {
	c.checkExpr(s.URL)
	if s.Sending != nil && s.Mapping != nil {
		c.err("sending and mapping are mutually exclusive", s.Line, s.Col)
	}
	if s.Mapping != nil {
		if sym := c.lookup(*s.Mapping, s.Line, s.Col); sym != nil && sym.Type != TypeGroup {
			c.err("mapping target '%s' is not a group item", s.Line, s.Col, *s.Mapping)
		}
	}
	if s.Giving != nil {
		c.lookup(*s.Giving, s.Line, s.Col)
	}
	if s.Status != "" {
		if sym := c.lookup(s.Status, s.Line, s.Col); sym != nil && sym.Type != TypeNumericComp {
			c.err("status field '%s' must be PIC 9(n) COMP", s.Line, s.Col, s.Status)
		}
	}
	if s.Headers != nil {
		c.checkHeaders(s.Headers, s.Line, s.Col)
	}
	c.checkExceptionBlocks(s.OnException, s.NotOnException)
}

func (c *Checker) checkHttpPatch(s *parser.HttpPatch) {
	c.checkExpr(s.URL)
	if s.Sending != nil && s.Mapping != nil {
		c.err("sending and mapping are mutually exclusive", s.Line, s.Col)
	}
	if s.Mapping != nil {
		if sym := c.lookup(*s.Mapping, s.Line, s.Col); sym != nil && sym.Type != TypeGroup {
			c.err("mapping target '%s' is not a group item", s.Line, s.Col, *s.Mapping)
		}
	}
	if s.Giving != nil {
		c.lookup(*s.Giving, s.Line, s.Col)
	}
	if s.Status != "" {
		if sym := c.lookup(s.Status, s.Line, s.Col); sym != nil && sym.Type != TypeNumericComp {
			c.err("status field '%s' must be PIC 9(n) COMP", s.Line, s.Col, s.Status)
		}
	}
	if s.Headers != nil {
		c.checkHeaders(s.Headers, s.Line, s.Col)
	}
	c.checkExceptionBlocks(s.OnException, s.NotOnException)
}

func (c *Checker) checkHttpDelete(s *parser.HttpDelete) {
	c.checkExpr(s.URL)
	if s.Giving != nil && s.Mapping != nil {
		c.err("giving and mapping are mutually exclusive", s.Line, s.Col)
	}
	if s.Giving != nil {
		c.lookup(*s.Giving, s.Line, s.Col)
	}
	if s.Mapping != nil {
		if sym := c.lookup(*s.Mapping, s.Line, s.Col); sym != nil && sym.Type != TypeGroup {
			c.err("mapping target '%s' is not a group item", s.Line, s.Col, *s.Mapping)
		}
	}
	if s.Status != "" {
		if sym := c.lookup(s.Status, s.Line, s.Col); sym != nil && sym.Type != TypeNumericComp {
			c.err("status field '%s' must be PIC 9(n) COMP", s.Line, s.Col, s.Status)
		}
	}
	if s.Headers != nil {
		c.checkHeaders(s.Headers, s.Line, s.Col)
	}
	c.checkExceptionBlocks(s.OnException, s.NotOnException)
}

func (c *Checker) checkHttpListen(s *parser.HttpListen) {
	if s.Mapping != nil {
		if sym := c.lookup(*s.Mapping, s.Line, s.Col); sym != nil && sym.Type != TypeGroup {
			c.err("mapping target '%s' is not a group item", s.Line, s.Col, *s.Mapping)
		}
	}
	if s.Status != "" {
		if sym := c.lookup(s.Status, s.Line, s.Col); sym != nil && sym.Type != TypeNumericComp {
			c.err("status field '%s' must be PIC 9(n) COMP", s.Line, s.Col, s.Status)
		}
	}
	c.checkExceptionBlocks(s.OnException, s.NotOnException)
}

func (c *Checker) checkHttpRespond(s *parser.HttpRespond) {
	if s.Body != nil {
		c.checkExpr(s.Body)
	}
	if s.ContentType != nil {
		c.checkExpr(s.ContentType)
	}
	if s.Mapping != nil {
		if sym := c.lookup(*s.Mapping, s.Line, s.Col); sym != nil && sym.Type != TypeGroup {
			c.err("mapping target '%s' is not a group item", s.Line, s.Col, *s.Mapping)
		}
	}
	if s.Headers != nil {
		c.checkHeaders(s.Headers, s.Line, s.Col)
	}
	c.checkExceptionBlocks(s.OnException, s.NotOnException)
}

func (c *Checker) checkHeaders(h *parser.HeadersPhrase, line, col int) {
	if h.Count != "" {
		c.lookup(h.Count, line, col)
	}
	if h.Group != "" {
		c.lookup(h.Group, line, col)
	}
}

func (c *Checker) checkExceptionBlocks(onException, notOnException []parser.Statement) {
	for _, stmt := range onException {
		c.checkStatement(stmt)
	}
	for _, stmt := range notOnException {
		c.checkStatement(stmt)
	}
}

// --- Expression checking ---

func (c *Checker) checkExpr(expr parser.Expression) {
	switch e := expr.(type) {
	case *parser.IdentifierExpr:
		c.lookup(e.Name, e.Line, e.Column)
	case *parser.BinaryExpr:
		c.checkBinaryExpr(e)
	case *parser.IntegerLiteralExpr, *parser.StringLiteralExpr:
		// literals are always valid
	}
}

func (c *Checker) checkBinaryExpr(e *parser.BinaryExpr) {
	c.checkExpr(e.Left)
	c.checkExpr(e.Right)

	leftType, errL := c.exprType(e.Left)
	rightType, errR := c.exprType(e.Right)
	if errL != nil || errR != nil {
		return
	}

	switch e.Operator {
	case parser.OpAdd, parser.OpSub, parser.OpMul, parser.OpDiv:
		// Arithmetic requires numeric operands
		if leftType.Type == TypeAlphanumeric {
			c.err("arithmetic operator requires numeric operand, got alphanumeric",
				e.Left.ExprLine(), e.Left.ExprCol())
		}
		if rightType.Type == TypeAlphanumeric {
			c.err("arithmetic operator requires numeric operand, got alphanumeric",
				e.Right.ExprLine(), e.Right.ExprCol())
		}
	}
}

// --- Helpers ---

func (c *Checker) lookup(name string, line, col int) *Symbol {
	if sym, ok := c.symbols[name]; ok {
		return sym
	}
	c.err("undefined identifier '%s'", line, col, name)
	return nil
}

type errPos struct {
	line int
	col  int
}

func (c *Checker) exprType(expr parser.Expression) (*Symbol, error) {
	switch e := expr.(type) {
	case *parser.StringLiteralExpr:
		return &Symbol{Type: TypeAlphanumeric, Size: len(e.Value)}, nil
	case *parser.IntegerLiteralExpr:
		return &Symbol{Type: TypeNumeric}, nil
	case *parser.IdentifierExpr:
		sym := c.lookup(e.Name, e.Line, e.Column)
		if sym == nil {
			return nil, fmt.Errorf("undefined")
		}
		return sym, nil
	case *parser.BinaryExpr:
		// arithmetic expressions produce numeric
		_, errL := c.exprType(e.Left)
		_, errR := c.exprType(e.Right)
		if errL != nil || errR != nil {
			return nil, fmt.Errorf("type error in expression")
		}
		return &Symbol{Type: TypeNumeric}, nil
	}
	return nil, fmt.Errorf("unknown expression type")
}

func (c *Checker) err(format string, line, col int, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	c.errs = append(c.errs, fmt.Errorf("line %d:%d: %s", line, col, msg))
}

func typeName(s *Symbol) string {
	switch s.Type {
	case TypeAlphanumeric:
		return "alphanumeric"
	case TypeNumeric:
		return "numeric"
	case TypeNumericComp:
		return "numeric-comp"
	case TypeGroup:
		return "group"
	}
	return "unknown"
}
