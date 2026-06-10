# COBOLabunga

A from-scratch COBOL compiler written in Go that emits LLVM IR
and compiles to native binaries. It has its own hand-written lexer,
recursive descent parser, type checker, and LLVM codegen. It is
not a fork of GnuCOBOL. It is not a transpiler. It is a person
who sat down and said "I'm going to write a COBOL compiler" and
then did it. The HTTP client runtime is pure Go. WASM is on the
table. This is absurd and it works.

## What works

| Feature | Status |
|---|---|
| IDENTIFICATION DIVISION, PROGRAM-ID | Done |
| DATA DIVISION, WORKING-STORAGE SECTION | Done |
| PIC X(n) and PIC 9(n) fields | Done |
| PIC 9(n) COMP (binary) fields | Done |
| Hierarchical data items (01/05 level) | Done |
| PROCEDURE DIVISION, paragraphs | Done |
| MOVE (literal, numeric, variable) | Done |
| COMPUTE (+, -, *, /) | Done |
| DISPLAY (strings, integers, mixed) | Done |
| IF / ELSE / END-IF | Done |
| EVALUATE / WHEN / WHEN OTHER | Done |
| PERFORM (paragraph call) | Done |
| STOP RUN | Done |
| HTTP-GET with GIVING and STATUS | Done |
| ON EXCEPTION / NOT ON EXCEPTION | Done |
| HTTP-POST / PUT / PATCH / DELETE | Codegen wired, runtime wired |
| MAPPING phrase (JSON ↔ DATA DIVISION) | Not yet |
| PERFORM VARYING / UNTIL | Not yet |
| HTTP-LISTEN / HTTP-RESPOND | Not yet |

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

Three steps happen silently: lex → parse → typeck → LLVM IR →
`llc` → object file → `clang` → executable.

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
200{
  "args": {},
  "headers": {
    "Accept-Encoding": "gzip",
    "Host": "httpbin.org",
    "User-Agent": "Go-http-client/2.0",
    "X-Amzn-Trace-Id": "Root=1-6a29e960-6a0f613a599d225017321169"
```

The HTTP runtime is a `c-archive` built from pure Go
(`net/http`, 30s timeout, follows redirects, body streamed
as `io.Reader`). It is linked automatically when the compiler
detects HTTP verbs in the source.

When HTTP detects the program uses HTTP, it builds
`runtime/libruntime.a` with `go build -buildmode=c-archive`
and links it. The compiler itself stays pure Go.

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
├── main.go              # Pipeline driver: lex → parse → typeck → codegen → llc → clang
├── lexer/
│   ├── token.go         # Token types
│   └── lexer.go         # Hand-written lexer (case-insensitive, hyphenated IDs)
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

## Roadmap

- [x] Hand-written lexer
- [x] Recursive descent parser
- [x] Symbol table and type checking
- [x] LLVM IR codegen (text emission)
- [x] Native binary output via llc + clang
- [x] HTTP-GET with GIVING and STATUS
- [x] ON EXCEPTION / NOT ON EXCEPTION branching
- [x] HTTP-POST / PUT / PATCH / DELETE (codegen + runtime)
- [x] MAPPING phrase (JSON ↔ DATA DIVISION fields)
- [x] GIVING / MAPPING mutual exclusion checking
- [ ] PERFORM VARYING / UNTIL (loop construct)
- [ ] HTTP-LISTEN / HTTP-RESPOND (server-side)
- [ ] WASM target
- [ ] Full-stack COBOL demo (HTTP-GET → parse JSON → DISPLAY)

## Relationship to COBOLScript

[COBOLScript](https://github.com/KDreynolds/COBOLScript) is a
fork of GnuCOBOL that adds HTTP extensions — it is a real COBOL
compiler that happens to speak HTTP. COBOLabunga is the clean
rewrite from scratch: its own lexer, parser, type system, and
codegen, with HTTP as a first-class citizen from day one. Both
live under `github.com/KDreynolds`. They disagree on approach
and agree on goal.
