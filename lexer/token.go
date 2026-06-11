package lexer

type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF

	// Structural keywords
	IDENTIFICATION
	DIVISION
	PROGRAM_ID
	DATA
	WORKING_STORAGE
	SECTION
	PROCEDURE
	PIC
	PICTURE
	COMP
	OCCURS
	TIMES
	VALUE

	// Procedure keywords
	MOVE
	TO
	COMPUTE
	DISPLAY
	PERFORM
	IF
	ELSE
	END_IF
	EVALUATE
	WHEN
	OTHER
	END_EVALUATE
	STOP
	RUN

	// PERFORM modifiers
	VARYING
	UNTIL
	FROM
	BY
	END_PERFORM

	// HTTP keywords
	HTTP_GET
	HTTP_POST
	HTTP_PUT
	HTTP_PATCH
	HTTP_DELETE
	HTTP_LISTEN
	HTTP_RESPOND
	PORT
	GIVING
	SENDING
	STATUS
	HEADERS
	MAPPING
	BODY
	CONTENT_TYPE
	ON
	EXCEPTION
	NOT
	END_HTTP

	// Literals
	STRING_LITERAL
	INTEGER_LITERAL
	IDENTIFIER

	// Punctuation / operators
	PERIOD
	LPAREN
	RPAREN
	EQUALS
	PLUS
	MINUS
	MULTIPLY
	DIVIDE
	GREATER
	LESS
)

var tokenNames = map[TokenType]string{
	ILLEGAL:         "ILLEGAL",
	EOF:             "EOF",
	IDENTIFICATION:  "IDENTIFICATION",
	DIVISION:        "DIVISION",
	PROGRAM_ID:      "PROGRAM-ID",
	DATA:            "DATA",
	WORKING_STORAGE: "WORKING-STORAGE",
	SECTION:         "SECTION",
	PROCEDURE:       "PROCEDURE",
	PIC:             "PIC",
	PICTURE:         "PICTURE",
	COMP:            "COMP",
	OCCURS:          "OCCURS",
	TIMES:           "TIMES",
	VALUE:           "VALUE",
	MOVE:            "MOVE",
	TO:              "TO",
	COMPUTE:         "COMPUTE",
	DISPLAY:         "DISPLAY",
	PERFORM:         "PERFORM",
	IF:              "IF",
	ELSE:            "ELSE",
	END_IF:          "END-IF",
	EVALUATE:        "EVALUATE",
	WHEN:            "WHEN",
	OTHER:           "OTHER",
	END_EVALUATE:    "END-EVALUATE",
	STOP:            "STOP",
	RUN:             "RUN",
	HTTP_GET:        "HTTP-GET",
	HTTP_POST:       "HTTP-POST",
	HTTP_PUT:        "HTTP-PUT",
	HTTP_PATCH:      "HTTP-PATCH",
	HTTP_DELETE:     "HTTP-DELETE",
	HTTP_LISTEN:     "HTTP-LISTEN",
	HTTP_RESPOND:    "HTTP-RESPOND",
	PORT:            "PORT",
	GIVING:          "GIVING",
	SENDING:         "SENDING",
	STATUS:          "STATUS",
	HEADERS:         "HEADERS",
	MAPPING:         "MAPPING",
	BODY:            "BODY",
	CONTENT_TYPE:    "CONTENT-TYPE",
	ON:              "ON",
	EXCEPTION:       "EXCEPTION",
	NOT:             "NOT",
	END_HTTP:        "END-HTTP",
	STRING_LITERAL:  "STRING_LITERAL",
	INTEGER_LITERAL: "INTEGER_LITERAL",
	IDENTIFIER:      "IDENTIFIER",
	VARYING:         "VARYING",
	UNTIL:           "UNTIL",
	FROM:            "FROM",
	BY:              "BY",
	END_PERFORM:     "END-PERFORM",
	PERIOD:          ".",
	LPAREN:          "(",
	RPAREN:          ")",
	EQUALS:          "=",
	PLUS:            "+",
	MINUS:           "-",
	MULTIPLY:        "*",
	DIVIDE:          "/",
	GREATER:         ">",
	LESS:            "<",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return "UNKNOWN"
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

func (t Token) String() string {
	return t.Type.String() + "(" + t.Literal + ")"
}
