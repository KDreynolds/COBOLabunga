package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/KDreynolds/COBOLabunga/codegen"
	"github.com/KDreynolds/COBOLabunga/lexer"
	"github.com/KDreynolds/COBOLabunga/parser"
	"github.com/KDreynolds/COBOLabunga/typeck"
)

func main() {
	wasmTarget := false
	bareMetal := false
	outName := ""
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--wasm" {
			wasmTarget = true
			args = append(args[:i], args[i+1:]...)
			i--
		} else if args[i] == "--target" && i+1 < len(args) {
			next := args[i+1]
			if next == "x86-bare" {
				bareMetal = true
				args = append(args[:i], args[i+2:]...)
				i--
			}
		} else if args[i] == "-o" && i+1 < len(args) {
			outName = args[i+1]
			args = append(args[:i], args[i+2:]...)
			i--
		}
	}
	if len(args) < 1 {
		fmt.Println("Usage: cobolabunga [--wasm] [--target x86-bare] [-o output] <file.cbl>")
		os.Exit(1)
	}

	// --- Phase 1: Lex ---
	src, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	l := lexer.New(string(src))
	tokens := l.Tokenize()

	// --- Phase 2: Parse ---
	p := parser.New(tokens)
	prog, parseErrs := p.Parse()
	if len(parseErrs) > 0 {
		fmt.Println("Parse errors:")
		for _, e := range parseErrs {
			fmt.Printf("  %s\n", e)
		}
		os.Exit(1)
	}

	// --- Phase 3: Type check ---
	tc := typeck.New()
	typeErrs := tc.Check(prog)
	if len(typeErrs) > 0 {
		fmt.Println("Type errors:")
		for _, e := range typeErrs {
			fmt.Printf("  %s\n", e)
		}
		os.Exit(1)
	}

	base := filepath.Base(args[0])
	name := outName
	if name == "" {
		name = base[:len(base)-len(filepath.Ext(base))]
	}

	// --- Phase 4: Codegen ---
	cg := codegen.New(prog)
	if wasmTarget {
		cg.SetTarget("wasm32-unknown-wasi")
	} else if bareMetal {
		cg.SetTarget("x86_64-pc-none-elf")
	}
	ir := cg.Generate()

	// Write IR to file
	irName := name + ".ll"
	if err := os.WriteFile(irName, []byte(ir), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing IR: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s\n", irName)

	// Compile IR to object
	objName := name + ".o"
	if wasmTarget {
		cmd := exec.Command("llc", "-filetype=obj", "-march=wasm32", "-o", objName, irName)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "llc error: %v\n%s\n", err, out)
			os.Exit(1)
		}
		fmt.Printf("Wrote %s\n", objName)
	} else {
		cmd := exec.Command("llc", "-filetype=obj", "-o", objName, irName)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "llc error: %v\n%s\n", err, out)
			os.Exit(1)
		}
		fmt.Printf("Wrote %s\n", objName)
	}

	// Link into executable
	exeName := name
	if wasmTarget {
		exeName = name + ".wasm"
		if usesHTTP(prog) {
			fmt.Fprintf(os.Stderr, "Error: HTTP features not supported in WASM target\n")
			os.Exit(1)
		}
		cmd := exec.Command("clang", "--target=wasm32-wasip1",
			"--sysroot=/usr/share/wasi-sysroot",
			"-o", exeName, objName)
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "clang (wasm) error: %v\n%s\n", err, out)
			os.Exit(1)
		}
		fmt.Printf("Wrote %s\n", exeName)
	} else if bareMetal {
		// Bare metal: just assemble to object file; linking done manually
		fmt.Printf("Wrote %s\n", objName)
	} else {
		needsRuntime := usesHTTP(prog) || usesStringRuntime(prog)
		if needsRuntime {
			runtimeDir := "runtime"
			runtimeLib := runtimeDir + "/libruntime.a"
			buildCmd := exec.Command("go", "build", "-buildmode=c-archive",
				"-o", filepath.Base(runtimeLib), ".")
			buildCmd.Dir = runtimeDir
			if out, err := buildCmd.CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stderr, "runtime build error: %v\n%s\n", err, out)
				os.Exit(1)
			}
			cmd := exec.Command("clang", "-no-pie", "-o", exeName, objName, "-Wl,--whole-archive", runtimeLib, "-Wl,--no-whole-archive")
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stderr, "clang error: %v\n%s\n", err, out)
				os.Exit(1)
			}
		} else {
			cmd := exec.Command("clang", "-no-pie", "-o", exeName, objName)
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stderr, "clang error: %v\n%s\n", err, out)
				os.Exit(1)
			}
		}
		fmt.Printf("Wrote %s\n", exeName)
	}
}

func usesHTTP(prog *parser.Program) bool {
	if prog.ProcedureDivision == nil {
		return false
	}
	for _, para := range prog.ProcedureDivision.Paragraphs {
		if statementsUseHTTP(para.Statements) {
			return true
		}
	}
	return false
}

func statementsUseHTTP(stmts []parser.Statement) bool {
	for _, stmt := range stmts {
		switch stmt.(type) {
		case *parser.HttpGet:
			s := stmt.(*parser.HttpGet)
			if statementsUseHTTP(s.OnException) || statementsUseHTTP(s.NotOnException) {
				return true
			}
			return true
		case *parser.HttpPost:
			s := stmt.(*parser.HttpPost)
			if statementsUseHTTP(s.OnException) || statementsUseHTTP(s.NotOnException) {
				return true
			}
			return true
		case *parser.HttpPut:
			s := stmt.(*parser.HttpPut)
			if statementsUseHTTP(s.OnException) || statementsUseHTTP(s.NotOnException) {
				return true
			}
			return true
		case *parser.HttpPatch:
			s := stmt.(*parser.HttpPatch)
			if statementsUseHTTP(s.OnException) || statementsUseHTTP(s.NotOnException) {
				return true
			}
			return true
		case *parser.HttpDelete:
			s := stmt.(*parser.HttpDelete)
			if statementsUseHTTP(s.OnException) || statementsUseHTTP(s.NotOnException) {
				return true
			}
			return true
		case *parser.HttpListen:
			s := stmt.(*parser.HttpListen)
			if statementsUseHTTP(s.OnException) || statementsUseHTTP(s.NotOnException) {
				return true
			}
			return true
		case *parser.HttpRespond:
			s := stmt.(*parser.HttpRespond)
			if statementsUseHTTP(s.OnException) || statementsUseHTTP(s.NotOnException) {
				return true
			}
			return true
		case *parser.Perform:
			s := stmt.(*parser.Perform)
			if statementsUseHTTP(s.Body) {
				return true
			}
		case *parser.If:
			s := stmt.(*parser.If)
			if statementsUseHTTP(s.ThenBody) || statementsUseHTTP(s.ElseBody) {
				return true
			}
		case *parser.Evaluate:
			s := stmt.(*parser.Evaluate)
			for _, wc := range s.WhenClauses {
				if statementsUseHTTP(wc.Body) {
					return true
				}
			}
			if statementsUseHTTP(s.WhenOther) {
				return true
			}
		case *parser.StringStmt:
			s := stmt.(*parser.StringStmt)
			if statementsUseHTTP(s.OnOverflow) || statementsUseHTTP(s.NotOnOverflow) {
				return true
			}
		case *parser.UnstringStmt:
			s := stmt.(*parser.UnstringStmt)
			if statementsUseHTTP(s.OnOverflow) || statementsUseHTTP(s.NotOnOverflow) {
				return true
			}
		}
	}
	return false
}

func usesStringRuntime(prog *parser.Program) bool {
	if prog.ProcedureDivision == nil {
		return false
	}
	for _, para := range prog.ProcedureDivision.Paragraphs {
		if statementsUseString(para.Statements) {
			return true
		}
		if statementsUseStringCmp(para.Statements, prog.WorkingStorage) {
			return true
		}
	}
	return false
}

func statementsUseString(stmts []parser.Statement) bool {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *parser.StringStmt:
			if statementsUseString(s.OnOverflow) || statementsUseString(s.NotOnOverflow) {
				return true
			}
			return true
		case *parser.UnstringStmt:
			if statementsUseString(s.OnOverflow) || statementsUseString(s.NotOnOverflow) {
				return true
			}
			return true
		case *parser.Perform:
			if statementsUseString(s.Body) {
				return true
			}
		case *parser.If:
			if statementsUseString(s.ThenBody) || statementsUseString(s.ElseBody) {
				return true
			}
		case *parser.Evaluate:
			for _, wc := range s.WhenClauses {
				if statementsUseString(wc.Body) {
					return true
				}
			}
			if statementsUseString(s.WhenOther) {
				return true
			}
		}
	}
	return false
}

func statementsUseStringCmp(stmts []parser.Statement, ws *parser.WorkingStorage) bool {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *parser.If:
			if exprUsesStringCmp(s.Condition, ws) {
				return true
			}
			if statementsUseStringCmp(s.ThenBody, ws) || statementsUseStringCmp(s.ElseBody, ws) {
				return true
			}
		case *parser.Evaluate:
			if evalUsesStringCmp(s, ws) {
				return true
			}
			for _, wc := range s.WhenClauses {
				if statementsUseStringCmp(wc.Body, ws) {
					return true
				}
			}
			if statementsUseStringCmp(s.WhenOther, ws) {
				return true
			}
		case *parser.Perform:
			if statementsUseStringCmp(s.Body, ws) {
				return true
			}
		case *parser.StringStmt:
			if statementsUseStringCmp(s.OnOverflow, ws) || statementsUseStringCmp(s.NotOnOverflow, ws) {
				return true
			}
		case *parser.UnstringStmt:
			if statementsUseStringCmp(s.OnOverflow, ws) || statementsUseStringCmp(s.NotOnOverflow, ws) {
				return true
			}
		}
	}
	return false
}

func exprUsesStringCmp(e parser.Expression, ws *parser.WorkingStorage) bool {
	switch x := e.(type) {
	case *parser.BinaryExpr:
		if x.Operator == parser.OpEq {
			leftIsStr := isStringField(x.Left, ws)
			rightIsStr := isStringField(x.Right, ws)
			return leftIsStr || rightIsStr
		}
		return exprUsesStringCmp(x.Left, ws) || exprUsesStringCmp(x.Right, ws)
	}
	return false
}

func evalUsesStringCmp(s *parser.Evaluate, ws *parser.WorkingStorage) bool {
	if isStringField(s.Subject, ws) {
		return true
	}
	for _, wc := range s.WhenClauses {
		for _, v := range wc.Values {
			if isStringField(v, ws) {
				return true
			}
		}
	}
	return false
}

func isStringField(e parser.Expression, ws *parser.WorkingStorage) bool {
	switch x := e.(type) {
	case *parser.StringLiteralExpr:
		return true
	case *parser.IdentifierExpr:
		if ws == nil {
			return false
		}
		for _, item := range ws.Items {
			if item.Name == x.Name && item.Picture != nil && item.Picture.Type == parser.PicX {
				return true
			}
		}
		return false
	}
	return false
}
