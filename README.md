# COBOLabunga

A from-scratch COBOL compiler written in Go, vibe-coded into
existence. Emits LLVM IR, compiles to native binaries and WASM.
Not a fork of GnuCOBOL. Not a transpiler. It just works.

## Current status

All working and tested:

- Lexer → Parser → Typeck → LLVM IR → llc → clang → binary
- WASM target via `--wasm` flag, WASI preview 1, wasmtime-compatible
- Statements: MOVE, COMPUTE, DISPLAY, STOP RUN, IF/ELSE,
  EVALUATE/WHEN, PERFORM (basic), STRING, UNSTRING
- HTTP client: GET, POST, PUT, PATCH, DELETE (pure Go runtime)
- HTTP server: LISTEN, RESPOND with HEADERS clause
- JSON: field expressions, request/response headers, PIC X equality
- Runtime: pure Go c-archive linked via clang

## Known bugs / deferred

- STRING/UNSTRING blocked under `--wasm` (needs pure LLVM IR
  or C runtime, no Go runtime in WASM)
- Forward-referenced globals fixed via inline GEP — do not revert
- Period-between-statements fix for paragraphs — do not revert

## Hello world

**examples/hello.cbl:**
```cobol
       IDENTIFICATION DIVISION.
       PROGRAM-ID. hello.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-MESSAGE    PIC X(50) VALUE "Hello, COBOLabunga!".
       PROCEDURE DIVISION.
       MAIN-PARA.
           DISPLAY WS-MESSAGE
           STOP RUN.
```

**Compile and run:**
```
$ ./cobolabunga examples/hello.cbl
Wrote hello.ll
Wrote hello.o
Wrote hello
$ ./hello
Hello, COBOLabunga!
```

Pipeline: lex → parse → typeck → LLVM IR → `llc` → object → `clang` → binary.

## HTTP GET example

**examples/httpget.cbl:**
```cobol
       IDENTIFICATION DIVISION.
       PROGRAM-ID. httpget.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-RESPONSE   PIC X(200).
       01 WS-STATUS     PIC 9(5) COMP.
       PROCEDURE DIVISION.
       MAIN.
           HTTP-GET "https://httpbin.org/get"
               GIVING WS-RESPONSE
               STATUS WS-STATUS
           END-HTTP
           DISPLAY "Status: " WS-STATUS
           DISPLAY WS-RESPONSE
           STOP RUN.
```

**Compile and run:**
```
$ ./cobolabunga examples/httpget.cbl
Wrote httpget.ll
Wrote httpget.o
Wrote httpget
$ ./httpget
Status:
200{ ... }
```

The HTTP runtime is a `c-archive` built from pure Go
(`net/http`, 30s timeout, follows redirects). It is linked
automatically when the compiler detects HTTP verbs in source.
The compiler itself stays pure Go.

## Building from source

**Requirements:**
- Go 1.21 or later
- LLVM (tested on 22.1.5) — provides `llc`
- `clang` — for the final link step

No LLVM Go bindings. No CGO in the compiler. LLVM IR is
emitted as text and compiled with `llc`.

**Build the compiler:**
```
$ git clone https://github.com/KDreynolds/COBOLabunga
$ cd COBOLabunga
$ go build -o cobolabunga .
```

**Compile a COBOL program:**
```
$ ./cobolabunga examples/compute.cbl
Wrote compute.ll
Wrote compute.o
Wrote compute
$ ./compute
Hello from COBOLabunga!
A + B =
30
```

## Project structure

```
COBOLabunga/
├── main.go              # Pipeline driver
├── lexer/
│   ├── token.go         # Token types
│   └── lexer.go         # Lexer (case-insensitive, hyphenated IDs)
├── parser/
│   ├── ast.go           # AST node types + debug printer
│   └── parser.go        # Recursive descent parser
├── typeck/
│   └── typeck.go        # Symbol table, type checking, validation
├── codegen/
│   └── codegen.go       # Text-based LLVM IR emission
├── runtime/
│   ├── go.mod           # Separate module for CGO build
│   └── runtime.go       # Pure Go HTTP client (net/http, //export cgo)
└── examples/
    ├── hello.cbl        # Hello world
    ├── compute.cbl      # MOVE / COMPUTE / DISPLAY
    ├── httpget.cbl      # HTTP-GET with GIVING + STATUS
    └── httpmap.cbl      # HTTP-GET with MAPPING (JSON → fields)
```

## Relationship to COBOLScript

[COBOLScript](https://github.com/KDreynolds/COBOLScript) is a
fork of GnuCOBOL that adds HTTP extensions. COBOLabunga is the
clean rewrite from scratch with HTTP as a first-class citizen
from day one. Both live under `github.com/KDreynolds`.
