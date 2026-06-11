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
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--wasm" {
			wasmTarget = true
			args = append(args[:i], args[i+1:]...)
			break
		}
	}
	if len(args) < 1 {
		fmt.Println("Usage: cobolabunga [--wasm] <file.cbl>")
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
	name := base[:len(base)-len(filepath.Ext(base))]

	// --- Phase 4: Codegen ---
	cg := codegen.New(prog)
	if wasmTarget {
		cg.SetTarget("wasm32-unknown-wasi")
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
		needHTTP := usesHTTP(prog)
		if needHTTP {
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
	} else {
		needHTTP := usesHTTP(prog)
		if needHTTP {
			runtimeDir := "runtime"
			runtimeLib := runtimeDir + "/libruntime.a"
			buildCmd := exec.Command("go", "build", "-buildmode=c-archive",
				"-o", filepath.Base(runtimeLib), ".")
			buildCmd.Dir = runtimeDir
			if out, err := buildCmd.CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stderr, "runtime build error: %v\n%s\n", err, out)
				os.Exit(1)
			}
			cmd := exec.Command("clang", "-no-pie", "-o", exeName, objName, runtimeLib)
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
		}
	}
	return false
}
