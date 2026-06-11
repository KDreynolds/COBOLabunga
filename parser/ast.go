package parser

import "fmt"

// --- Program ---

type Program struct {
	Name              string
	WorkingStorage    *WorkingStorage
	ProcedureDivision *ProcedureDivision
}

// --- Data Division ---

type WorkingStorage struct {
	Items []*DataItem
}

type PicType int

const (
	PicX PicType = iota
	Pic9
)

type Picture struct {
	Type   PicType
	Size   int
	IsComp bool
}

type DataItem struct {
	Level    int
	Name     string
	Picture  *Picture
	Occurs   int
	Value    string
	Children []*DataItem
	Line     int
	Column   int
}

// --- Procedure Division ---

type ProcedureDivision struct {
	Paragraphs []*Paragraph
}

type Paragraph struct {
	Name       string
	Statements []Statement
}

// --- Statement Interface ---

type Statement interface {
	stmtTag()
}

type Move struct {
	From Expression
	To   string
	Line int
	Col  int
}

type Compute struct {
	Target string
	Expr   Expression
	Line   int
	Col    int
}

type Display struct {
	Items []Expression
	Line  int
	Col   int
}

type VaryingPhrase struct {
	Variable string
	From     Expression
	By       Expression
	Until    Expression
}

type Perform struct {
	Paragraph string
	Times     Expression
	Varying   *VaryingPhrase
	Until     Expression
	Body      []Statement
	Line      int
	Col       int
}

type If struct {
	Condition Expression
	ThenBody  []Statement
	ElseBody  []Statement
	Line      int
	Col       int
}

type Evaluate struct {
	Subject     Expression
	WhenClauses []WhenClause
	WhenOther   []Statement
	Line        int
	Col         int
}

type WhenClause struct {
	Values []Expression
	Body   []Statement
}

type Accept struct {
	Name string
	Line int
	Col  int
}

type Initialize struct {
	Items []string
	Line  int
	Col   int
}

type StopRun struct {
	Line int
	Col  int
}

type Peek struct {
	Address Expression
	Into    string
	Line    int
	Col     int
}

type Poke struct {
	Address Expression
	Value   Expression
	Line    int
	Col     int
}

type PortIn struct {
	Port Expression
	Into string
	Line int
	Col  int
}

type PortOut struct {
	Port  Expression
	Value Expression
	Line  int
	Col   int
}

type HttpGet struct {
	URL            Expression
	Giving         *string
	Mapping        *string
	Status         string
	Headers        *HeadersPhrase
	OnException    []Statement
	NotOnException []Statement
	Line           int
	Col            int
}

type HttpPost struct {
	URL            Expression
	Sending        Expression
	Mapping        *string
	Headers        *HeadersPhrase
	Giving         *string
	Status         string
	OnException    []Statement
	NotOnException []Statement
	Line           int
	Col            int
}

type HttpPut struct {
	URL            Expression
	Sending        Expression
	Mapping        *string
	Headers        *HeadersPhrase
	Giving         *string
	Status         string
	OnException    []Statement
	NotOnException []Statement
	Line           int
	Col            int
}

type HttpPatch struct {
	URL            Expression
	Sending        Expression
	Mapping        *string
	Headers        *HeadersPhrase
	Giving         *string
	Status         string
	OnException    []Statement
	NotOnException []Statement
	Line           int
	Col            int
}

type HttpDelete struct {
	URL            Expression
	Mapping        *string
	Headers        *HeadersPhrase
	Giving         *string
	Status         string
	OnException    []Statement
	NotOnException []Statement
	Line           int
	Col            int
}

type HeadersPhrase struct {
	Count string
	Group string
}

type HttpListen struct {
	Port           int
	Mapping        *string
	Status         string
	Headers        *HeadersPhrase
	OnException    []Statement
	NotOnException []Statement
	Line           int
	Col            int
}

type HttpRespond struct {
	Status         int
	Body           Expression
	ContentType    Expression
	Headers        *HeadersPhrase
	Mapping        *string
	OnException    []Statement
	NotOnException []Statement
	Line           int
	Col            int
}

// --- String Delimiter Types ---

type StringDelimType int

const (
	DelimBySize StringDelimType = iota
	DelimBySpace
	DelimByIdentifier
)

type StringDelim struct {
	Type  StringDelimType
	Value Expression // for DelimByIdentifier
}

type StringSendingField struct {
	Source    Expression
	Delimiter StringDelim
}

type StringStmt struct {
	Sending       []StringSendingField
	Into          string
	Pointer       string
	OnOverflow    []Statement
	NotOnOverflow []Statement
	Line          int
	Col           int
}

// --- UNSTRING Types ---

type UnstringReceivingField struct {
	Destination string
	DelimiterIn string
	CountIn     string
}

type UnstringStmt struct {
	Source        Expression
	Delimiters    []StringDelim // OR'd list
	Into          []UnstringReceivingField
	Pointer       string
	Tallying      string
	OnOverflow    []Statement
	NotOnOverflow []Statement
	Line          int
	Col           int
}

func (*Move) stmtTag()        {}
func (*Compute) stmtTag()     {}
func (*Display) stmtTag()     {}
func (*Perform) stmtTag()     {}
func (*If) stmtTag()          {}
func (*Evaluate) stmtTag()    {}
func (*Accept) stmtTag()      {}
func (*Initialize) stmtTag()  {}
func (*StopRun) stmtTag()     {}
func (*Peek) stmtTag()       {}
func (*Poke) stmtTag()       {}
func (*PortIn) stmtTag()     {}
func (*PortOut) stmtTag()    {}
func (*StringStmt) stmtTag()  {}
func (*UnstringStmt) stmtTag() {}
func (*HttpGet) stmtTag()     {}
func (*HttpPost) stmtTag()    {}
func (*HttpPut) stmtTag()     {}
func (*HttpPatch) stmtTag()   {}
func (*HttpDelete) stmtTag()  {}
func (*HttpListen) stmtTag()  {}
func (*HttpRespond) stmtTag() {}

// --- Expression Interface ---

type Expression interface {
	exprTag()
	ExprLine() int
	ExprCol() int
}

type IdentifierExpr struct {
	Name   string
	Line   int
	Column int
}

type IntegerLiteralExpr struct {
	Value int
}

type StringLiteralExpr struct {
	Value string
}

type BinaryExpr struct {
	Left     Expression
	Operator Op
	Right    Expression
}

type Op int

const (
	OpAdd Op = iota
	OpSub
	OpMul
	OpDiv
	OpGt
	OpLt
	OpEq
	OpGe
	OpLe
)

func (e *IdentifierExpr) exprTag()     {}
func (e *IntegerLiteralExpr) exprTag() {}
func (e *StringLiteralExpr) exprTag()  {}
func (e *BinaryExpr) exprTag()         {}

func (e *IdentifierExpr) ExprLine() int     { return e.Line }
func (e *IdentifierExpr) ExprCol() int      { return e.Column }
func (e *IntegerLiteralExpr) ExprLine() int { return 0 }
func (e *IntegerLiteralExpr) ExprCol() int  { return 0 }
func (e *StringLiteralExpr) ExprLine() int  { return 0 }
func (e *StringLiteralExpr) ExprCol() int   { return 0 }
func (e *BinaryExpr) ExprLine() int         { return e.Left.ExprLine() }
func (e *BinaryExpr) ExprCol() int          { return e.Left.ExprCol() }

// --- AST Debug Printer ---

func PrintAST(p *Program) {
	fmt.Printf("Program: %s\n", p.Name)
	if p.WorkingStorage != nil {
		fmt.Println("  DATA DIVISION:")
		fmt.Println("    WORKING-STORAGE SECTION:")
		for _, item := range p.WorkingStorage.Items {
			printDataItem(item, 2)
		}
	}
	if p.ProcedureDivision != nil {
		fmt.Println("  PROCEDURE DIVISION:")
		for _, para := range p.ProcedureDivision.Paragraphs {
			fmt.Printf("    Paragraph: %s\n", para.Name)
			for _, stmt := range para.Statements {
				printStatement(stmt, 3)
			}
		}
	}
}

func printDataItem(item *DataItem, indent int) {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}
	fmt.Printf("%sDataItem: level=%d name=%s", prefix, item.Level, item.Name)
	if item.Picture != nil {
		picType := "X"
		if item.Picture.Type == Pic9 {
			picType = "9"
		}
		comp := ""
		if item.Picture.IsComp {
			comp = " COMP"
		}
		fmt.Printf(" pic=%s(%d)%s", picType, item.Picture.Size, comp)
	}
	if item.Occurs > 0 {
		fmt.Printf(" occurs=%d", item.Occurs)
	}
	if item.Value != "" {
		fmt.Printf(" value=%q", item.Value)
	}
	fmt.Println()
	for _, child := range item.Children {
		printDataItem(child, indent+1)
	}
}

func printStatement(stmt Statement, indent int) {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}
	switch s := stmt.(type) {
	case *Move:
		fmt.Printf("%sMOVE %s TO %s\n", prefix, exprStr(s.From), s.To)
	case *Compute:
		fmt.Printf("%sCOMPUTE %s = %s\n", prefix, s.Target, exprStr(s.Expr))
	case *Display:
		fmt.Printf("%sDISPLAY %s\n", prefix, exprListStr(s.Items))
	case *Perform:
		if s.Varying != nil {
			fmt.Printf("%sPERFORM VARYING %s FROM %s BY %s UNTIL %s\n",
				prefix, s.Varying.Variable, exprStr(s.Varying.From),
				exprStr(s.Varying.By), exprStr(s.Varying.Until))
			for _, st := range s.Body {
				printStatement(st, indent+1)
			}
			fmt.Printf("%sEND-PERFORM\n", prefix)
		} else {
			t := ""
			if s.Times != nil {
				t = " " + exprStr(s.Times) + " TIMES"
			}
			fmt.Printf("%sPERFORM %s%s\n", prefix, s.Paragraph, t)
		}
	case *If:
		fmt.Printf("%sIF %s\n", prefix, exprStr(s.Condition))
		for _, st := range s.ThenBody {
			printStatement(st, indent+1)
		}
		if len(s.ElseBody) > 0 {
			fmt.Printf("%s  ELSE\n", prefix)
			for _, st := range s.ElseBody {
				printStatement(st, indent+1)
			}
		}
		fmt.Printf("%sEND-IF\n", prefix)
	case *Evaluate:
		fmt.Printf("%sEVALUATE %s\n", prefix, exprStr(s.Subject))
		for _, wc := range s.WhenClauses {
			fmt.Printf("%s  WHEN %s\n", prefix, exprListStr(wc.Values))
			for _, st := range wc.Body {
				printStatement(st, indent+1)
			}
		}
		if len(s.WhenOther) > 0 {
			fmt.Printf("%s  WHEN OTHER\n", prefix)
			for _, st := range s.WhenOther {
				printStatement(st, indent+1)
			}
		}
		fmt.Printf("%sEND-EVALUATE\n", prefix)
	case *Accept:
		fmt.Printf("%sACCEPT %s\n", prefix, s.Name)
	case *Initialize:
		fmt.Printf("%sINITIALIZE", prefix)
		for _, item := range s.Items {
			fmt.Printf(" %s", item)
		}
		fmt.Println()
	case *StopRun:
		fmt.Printf("%sSTOP RUN\n", prefix)
	case *Peek:
		fmt.Printf("%sPEEK(%s) INTO %s\n", prefix, exprStr(s.Address), s.Into)
	case *Poke:
		fmt.Printf("%sPOKE(%s, %s)\n", prefix, exprStr(s.Address), exprStr(s.Value))
	case *PortIn:
		fmt.Printf("%sPORT-IN(%s) INTO %s\n", prefix, exprStr(s.Port), s.Into)
	case *PortOut:
		fmt.Printf("%sPORT-OUT(%s, %s)\n", prefix, exprStr(s.Port), exprStr(s.Value))
	case *HttpGet:
		fmt.Printf("%sHTTP-GET %s", prefix, exprStr(s.URL))
		if s.Giving != nil {
			fmt.Printf(" GIVING %s", *s.Giving)
		}
		fmt.Printf(" STATUS %s\n", s.Status)
	case *HttpPost:
		fmt.Printf("%sHTTP-POST %s", prefix, exprStr(s.URL))
		if s.Sending != nil {
			fmt.Printf(" SENDING %s", exprStr(s.Sending))
		}
		if s.Giving != nil {
			fmt.Printf(" GIVING %s", *s.Giving)
		}
		fmt.Printf(" STATUS %s\n", s.Status)
	case *HttpPut:
		fmt.Printf("%sHTTP-PUT %s", prefix, exprStr(s.URL))
		if s.Sending != nil {
			fmt.Printf(" SENDING %s", exprStr(s.Sending))
		}
		if s.Giving != nil {
			fmt.Printf(" GIVING %s", *s.Giving)
		}
		fmt.Printf(" STATUS %s\n", s.Status)
	case *HttpPatch:
		fmt.Printf("%sHTTP-PATCH %s", prefix, exprStr(s.URL))
		if s.Sending != nil {
			fmt.Printf(" SENDING %s", exprStr(s.Sending))
		}
		if s.Giving != nil {
			fmt.Printf(" GIVING %s", *s.Giving)
		}
		fmt.Printf(" STATUS %s\n", s.Status)
	case *HttpDelete:
		fmt.Printf("%sHTTP-DELETE %s", prefix, exprStr(s.URL))
		if s.Giving != nil {
			fmt.Printf(" GIVING %s", *s.Giving)
		}
		fmt.Printf(" STATUS %s\n", s.Status)
	case *HttpListen:
		fmt.Printf("%sHTTP-LISTEN PORT %d", prefix, s.Port)
		if s.Mapping != nil {
			fmt.Printf(" MAPPING %s", *s.Mapping)
		}
		if s.Status != "" {
			fmt.Printf(" STATUS %s", s.Status)
		}
		fmt.Println()
	case *HttpRespond:
		fmt.Printf("%sHTTP-RESPOND STATUS %d", prefix, s.Status)
		if s.Body != nil {
			fmt.Printf(" BODY %s", exprStr(s.Body))
		}
		if s.ContentType != nil {
			fmt.Printf(" CONTENT-TYPE %s", exprStr(s.ContentType))
		}
		fmt.Println()
	case *StringStmt:
		fmt.Printf("%sSTRING\n", prefix)
		for _, f := range s.Sending {
			delim := ""
			switch f.Delimiter.Type {
			case DelimBySize:
				delim = "SIZE"
			case DelimBySpace:
				delim = "SPACE"
			case DelimByIdentifier:
				delim = exprStr(f.Delimiter.Value)
			}
			fmt.Printf("%s  %s DELIMITED BY %s\n", prefix, exprStr(f.Source), delim)
		}
		fmt.Printf("%s  INTO %s\n", prefix, s.Into)
		if s.Pointer != "" {
			fmt.Printf("%s  POINTER %s\n", prefix, s.Pointer)
		}
		fmt.Printf("%sEND-STRING\n", prefix)
	case *UnstringStmt:
		fmt.Printf("%sUNSTRING %s", prefix, exprStr(s.Source))
		if len(s.Delimiters) > 0 {
			fmt.Printf(" DELIMITED BY ")
			for i, d := range s.Delimiters {
				if i > 0 {
					fmt.Printf(" OR ")
				}
				switch d.Type {
				case DelimBySize:
					fmt.Printf("SIZE")
				case DelimBySpace:
					fmt.Printf("SPACE")
				case DelimByIdentifier:
					fmt.Printf("%s", exprStr(d.Value))
				}
			}
		}
		fmt.Println()
		for _, f := range s.Into {
			fmt.Printf("%s  INTO %s", prefix, f.Destination)
			if f.DelimiterIn != "" {
				fmt.Printf(" DELIMITER IN %s", f.DelimiterIn)
			}
			if f.CountIn != "" {
				fmt.Printf(" COUNT IN %s", f.CountIn)
			}
			fmt.Println()
		}
		if s.Pointer != "" {
			fmt.Printf("%s  POINTER %s\n", prefix, s.Pointer)
		}
		if s.Tallying != "" {
			fmt.Printf("%s  TALLYING IN %s\n", prefix, s.Tallying)
		}
		fmt.Printf("%sEND-UNSTRING\n", prefix)
	}
}

func exprStr(e Expression) string {
	switch v := e.(type) {
	case *IdentifierExpr:
		return v.Name
	case *IntegerLiteralExpr:
		return fmt.Sprintf("%d", v.Value)
	case *StringLiteralExpr:
		return fmt.Sprintf("%q", v.Value)
	case *BinaryExpr:
		op := "+"
		switch v.Operator {
		case OpSub:
			op = "-"
		case OpMul:
			op = "*"
		case OpDiv:
			op = "/"
		case OpGt:
			op = ">"
		case OpLt:
			op = "<"
		case OpEq:
			op = "="
		case OpGe:
			op = ">="
		case OpLe:
			op = "<="
		}
		return "(" + exprStr(v.Left) + " " + op + " " + exprStr(v.Right) + ")"
	}
	return "?"
}

func exprListStr(items []Expression) string {
	s := ""
	for i, item := range items {
		if i > 0 {
			s += " "
		}
		s += exprStr(item)
	}
	return s
}
