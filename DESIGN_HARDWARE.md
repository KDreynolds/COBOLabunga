# COBOS: Bare Metal COBOL Operating System — Design Document

Phase 3 of the COBOLabunga project. This document defines the target architecture, runtime abstraction, new COBOL verbs, boot strategy, and roadmap for booting a COBOL kernel on bare x86-64 hardware, with ESP32 as a secondary target.

---

## 1. Bare Metal LLVM Target

### Concept

COBOLabunga currently emits LLVM IR for `x86_64-pc-linux-gnu` (hosted, libc-linked) and `wasm32-unknown-wasi` (WASI preview 1). For bare metal, no operating system exists: no libc, no syscalls, no process boundary. The compiler must produce ELF binaries that link directly against hardware.

### Target Triples

| Target | Triple | ABI | Linker Script |
|--------|--------|-----|---------------|
| x86-64 bare metal | `x86_64-unknown-none` | ELF64, no libc | Custom `cobos.ld` |
| ESP32 (Xtensa) | `xtensa-esp32-none-elf` | ELF32 Xtensa | ESP-IDF `sections.ld` |

### What Changes

- **No libc**: `puts`, `printf`, `fgets`, `atoi` are unavailable. All I/O emits inline LLVM IR or calls target-specific runtime stubs.
- **No Go runtime**: `libruntime.a` (HTTP, JSON, STRING/UNSTRING stubs) is not linked. Each target provides its own runtime as a set of C stubs or inline IR.
- **No startup crt**: No `_start` from glibc. The bootloader or a minimal crt0 assembly stub sets up the stack and calls the COBOL entry point directly.
- **Linker script**: Must define `.text`, `.data`, `.bss` at known physical addresses (e.g., `0x100000` for x86-64 multiboot).

### CLI Interface

```
./cobolabunga prog.cbl --target=x86-bare
./cobolabunga prog.cbl --target=esp32
```

Mapping:

| `--target` value | LLVM triple | Runtime |
|------------------|-------------|---------|
| (default) | `x86_64-pc-linux-gnu` | Go c-archive |
| `wasm` | `wasm32-unknown-wasi` | WASI libc |
| `x86-bare` | `x86_64-unknown-none` | `runtime/x86_bare/` |
| `esp32` | `xtensa-esp32-none-elf` | ESP-IDF |

When `--target` is `x86-bare` or `esp32`, the compiler:
1. Sets `c.isBareMetal = true` (disables Go runtime declarations).
2. Emits only hardware-safe LLVM IR (no libc `declare`s except those explicitly provided).
3. After `llc`, invokes the target linker (e.g., `ld` with `cobos.ld` for x86, or `esptool.py` via ESP-IDF for ESP32).

---

## 2. Runtime Abstraction Layer

### Current Architecture

```
prog.cbl → lexer → parser → typeck → codegen → prog.ll → llc → prog.o
                                                              + libruntime.a (Go c-archive)
                                                              → clang -no-pie → prog
```

### Bare Metal Architecture

```
prog.cbl → lexer → parser → typeck → codegen → prog.ll → llc → prog.o
                                                              + runtime_stubs.c (target-specific)
                                                              + crt0.o (target-specific)
                                                              → ld -T cobos.ld → prog.elf
```

### Runtime Interface Boundary

The compiler emits calls to a fixed set of runtime symbols. Each target provides implementations:

| Symbol | Current (Go) | WASM (WASI) | x86-bare | ESP32 |
|--------|-------------|-------------|----------|-------|
| `cob_http_get` | Go HTTP client | Not available | Not available | ESP-IDF `esp_http_client` |
| `cob_http_post` | Go HTTP client | Not available | Not available | ESP-IDF `esp_http_client` |
| `cob_string_init` | Go alloc | Inline IR | Inline IR | Inline IR |
| `cob_unstring_init` | Go alloc | Inline IR | Inline IR | Inline IR |
| `puts` / `printf` | libc | WASI libc | VGA driver stub | UART stub |
| `fgets` | libc | WASI libc | PS/2 keyboard stub | UART RX stub |
| `atoi` | libc | WASI libc | Inline IR | Inline IR |

On bare metal targets, the compiler does NOT emit `declare` for libc functions. Instead, it either:
- Emits inline LLVM IR (e.g., `memcpy` for STRING, `icmp`/`br` for UNSTRING scan loops), or
- Declares target-specific runtime stubs provided in `runtime/x86_bare/` or `runtime/esp32/`.

### Link-Time Selection

The `Makefile` or build script selects the runtime directory based on `--target`:

```
# x86-bare
ld -T cobos.ld prog.o runtime/x86_bare/*.o crt0.o -o cobos.elf

# esp32  
idf.py build    (prog.o is linked via ESP-IDF build system)
```

---

## 3. New COBOL Verbs for Hardware Access

### PEEK — Read from Memory Address

```
PEEK WS-VALUE FROM WS-ADDRESS.
```

Semantics: Loads the 32-bit value at the memory address stored in `WS-ADDRESS` and stores it in `WS-VALUE`.

LLVM IR emission (volatile load with `inttoptr`):

```llvm
%addr = load i32, ptr @cob.WS_ADDRESS
%ptr = inttoptr i32 %addr to ptr
%val = load volatile i32, ptr %ptr
store i32 %val, ptr @cob.WS_VALUE
```

### POKE — Write to Memory Address

```
POKE WS-VALUE TO WS-ADDRESS.
```

Semantics: Stores the 32-bit value in `WS-VALUE` to the memory address stored in `WS-ADDRESS`.

LLVM IR emission (volatile store with `inttoptr`):

```llvm
%addr = load i32, ptr @cob.WS_ADDRESS
%ptr = inttoptr i32 %addr to ptr
%val = load i32, ptr @cob.WS_VALUE
store volatile i32 %val, ptr %ptr
```

### PORT-IN — Read from x86 I/O Port

```
PORT-IN WS-VALUE FROM PORT 0x60.
```

Semantics: Reads a byte from the x86 I/O port specified by the integer literal or identifier after `PORT`.

LLVM IR emission (inline assembly for `inb`):

```llvm
%val = call i8 asm sideeffect "inb $1, $0", "=a,{dx}"(i32 0x60)
%ext = zext i8 %val to i32
store i32 %ext, ptr @cob.WS_VALUE
```

For variable ports:

```llvm
%port = load i32, ptr @cob.WS_PORT
%val = call i8 asm sideeffect "inb $1, $0", "=a,{dx}"(i32 %port)
```

### PORT-OUT — Write to x86 I/O Port

```
PORT-OUT WS-VALUE TO PORT 0x3F8.
```

Semantics: Writes a byte to the x86 I/O port specified.

LLVM IR emission (inline assembly for `outb`):

```llvm
%val = load i32, ptr @cob.WS_VALUE
%trunc = trunc i32 %val to i8
call void asm sideeffect "outb $0, $1", "{ax},{dx}"(i8 %trunc, i32 0x3F8)
```

### Token Summary

| Token | Parser Action |
|-------|---------------|
| `PEEK` | Parse two identifiers: `WS-VALUE FROM WS-ADDRESS` |
| `POKE` | Parse two identifiers: `WS-VALUE TO WS-ADDRESS` |
| `PORT` | Qualifier for PORT-IN / PORT-OUT |
| `PORT_IN` | Parse `WS-VALUE FROM PORT expr` |
| `PORT_OUT` | Parse `WS-VALUE TO PORT expr` |

These tokens are only recognized in bare-metal mode. If used with `--target` default (hosted), the compiler emits a compile-time error.

---

## 4. Bootloader Strategy

### Option A: Pure Assembly Stub (Recommended)

A small `boot.asm` file with:

```asm
; Multiboot2 header
section .multiboot
align 8
    dd 0xE85250D6           ; magic
    dd 0                    ; architecture (i386)
    dd header_end - header_start
    dd -(0xE85250D6 + 0 + (header_end - header_start)) ; checksum
    ; end tag
    dw 0, 0
    dd 8
header_end:

section .text
global _start
_start:
    mov esp, stack_top
    ; zero BSS
    extern __bss_start, __bss_end
    mov edi, __bss_start
    mov ecx, __bss_end
    sub ecx, edi
    xor eax, eax
    rep stosb
    ; call COBOL entry
    extern cob.KERNEL_MAIN
    call cob.KERNEL_MAIN
    cli
    hlt

section .bss
stack_bottom:
    resb 16384
stack_top:
```

**Why Option A is recommended:**

1. **Reliable**: Multiboot2 header format is delicate. Hand-written assembly gives precise control over alignment, section placement, and checksums.
2. **Stack setup**: Assembly must set up the stack before calling any COBOL code. COBOL code (through LLVM IR) assumes a valid `%rsp`.
3. **BSS zeroing**: The COBOL compiler emits global variables initialized with zeros or spaces. The bootloader must zero the BSS section before calling COBOL code.
4. **Separation of concerns**: The bootloader is a tiny, well-understood piece of assembly that never changes. The COBOL kernel is where new development happens.

### Option B: Multiboot2 Header in COBOL DATA DIVISION

A special `01`-level data item that the compiler recognizes and transforms into the multiboot header in the correct ELF section:

```cobol
DATA DIVISION.
WORKING-STORAGE SECTION.
01 MULTIBOOT-HEADER.
   05 MB-MAGIC    PIC 9(8) COMP VALUE 0xE85250D6.
   05 MB-ARCH     PIC 9(8) COMP VALUE 0.
   05 MB-LENGTH   PIC 9(8) COMP.
   05 MB-CKSUM    PIC 9(8) COMP.
```

The compiler would:
1. Detect the `MULTIBOOT-HEADER` name pattern.
2. Emit these bytes in a `.multiboot` ELF section instead of `.data` or `.bss`.
3. Compute the checksum and length fields automatically.

**Why Option A is preferred over Option B:**

1. The COBOL data item abstraction leaks: checksum computation requires arithmetic the compiler does not do at compile time.
2. Section placement (`.multiboot` vs `.data`) requires a new codegen concept: "this data item goes in a specific ELF section".
3. The bootloader header must appear first in the ELF file, before any code. The compiler's data emission order does not guarantee this.
4. Assembly is 30 lines and never changes. Not worth abstracting.

### Boot Flow

```
QEMU/BIOS → boot.S (_start) → setup stack → zero BSS → call cob.KERNEL_MAIN
                                                                  ↓
                                                         COBOL code runs
                                                                  ↓
                                                         DISPLAY to VGA
                                                         ACCEPT from PS/2
                                                         PERFORM forever
```

---

## 5. DISPLAY and ACCEPT on Bare Metal

### VGA Text Mode (x86)

The VGA text mode buffer is at physical address `0xB8000`. Each character cell is 2 bytes: ASCII byte + attribute byte (foreground, background color, blink).

**DISPLAY `"HELLO"` on bare metal:**

```llvm
; VGA buffer base
%vga = inttoptr i64 0xB8000 to ptr

; Write 'H' at row 0, col 0 (offset 0)
%cell0 = getelementptr i8, ptr %vga, i64 0
store i8 72, ptr %cell0                    ; 'H'
%attr0 = getelementptr i8, ptr %vga, i64 1
store i8 0x07, ptr %attr0                  ; white on black

; Write 'E' at row 0, col 1 (offset 2)
%cell1 = getelementptr i8, ptr %vga, i64 2
store i8 69, ptr %cell1
%attr1 = getelementptr i8, ptr %vga, i64 3
store i8 0x07, ptr %attr1
; ... etc
```

A VGA cursor position counter (stored in a global or passed via alloca) tracks the current column/row, wrapping at 80 columns and scrolling at 25 rows.

### UART Display (ESP32)

ESP32 UART is memory-mapped at `0x3FF40000` (UART1). Transmit by writing to the FIFO register at offset `0x00`:

```llvm
%uart = inttoptr i64 0x3FF40000 to ptr
%data = load i8, ptr @char_ptr
store volatile i8 %data, ptr %uart
```

A spin loop checks the THR (transmit holding register) empty bit at offset `0x1C` before writing.

### Target Selection in Codegen

The `emitDisplay` function currently dispatches to `puts`/`printf`. On bare metal:

```
emitDisplay(s):
  if c.isBareMetal && c.targetTriple == "x86_64-unknown-none":
    emitVgaDisplay(s)
  elif c.isBareMetal && c.targetTriple == "xtensa-esp32-none-elf":
    emitUartDisplay(s)
  else:
    emitHostedDisplay(s)   # current puts/printf path
```

`emitVgaDisplay` emits the inline VGA buffer writes shown above.
`emitUartDisplay` emits the UART register writes shown above.

### ACCEPT from PS/2 Keyboard (x86)

PS/2 keyboard input reads from I/O port `0x60`. Scancodes must be translated to ASCII:

1. **Wait for data**: Poll port `0x64` bit 0 (output buffer status). Loop until set.
2. **Read scancode**: `inb` from port `0x60`.
3. **Translate**: Map the scancode byte to ASCII. For the first release, support only the main alphanumeric keys (set 1 scancodes).
4. **Store**: Write the ASCII byte to the field. On Enter (scancode `0x1C`), terminate.

The scancode-to-ASCII translation table is a 128-byte constant emitted in the `.data` section by the compiler.

### ACCEPT from UART (ESP32)

Read from the UART FIFO register. Poll the RX status bit, read a byte, echo it back via UART display.

---

## 6. Minimal COBOL Kernel Design

### The Kernel

```cobol
       IDENTIFICATION DIVISION.
       PROGRAM-ID. cobos-kernel.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-VGA-BUFFER   PIC 9(8) COMP VALUE 753664.
       01 WS-MESSAGE      PIC X(20) VALUE "COBOS v0.1 BOOTING".
       01 WS-INPUT        PIC X(80).
       01 WS-PROMPT       PIC X(2) VALUE "> ".
       PROCEDURE DIVISION.
       KERNEL-MAIN.
           DISPLAY WS-MESSAGE
           DISPLAY WS-PROMPT
           PERFORM KERNEL-LOOP.
       KERNEL-LOOP.
           ACCEPT WS-INPUT
           PERFORM HANDLE-INPUT
           DISPLAY WS-PROMPT
           PERFORM KERNEL-LOOP.
       HANDLE-INPUT.
           IF WS-INPUT = "HELLO"
               DISPLAY "HI"
           ELSE
               DISPLAY "?"
           END-IF.
```

### What Must Work for This to Boot

| Component | Requirement | Mechanism |
|-----------|-------------|-----------|
| **Bootloader** | Hand control to `cob.KERNEL_MAIN` | `boot.S` multiboot stub |
| **DISPLAY** | Write "COBOS v0.1 BOOTING" to screen | VGA at `0xB8000`, white on black |
| **ACCEPT** | Read line from keyboard | PS/2 port `0x60`, scancode table |
| **IF** | String comparison | `memcmp` or inline `icmp` loop |
| **PERFORM** | Infinite loop | `br` back to `KERNEL-LOOP` |
| **DISPLAY (response)** | Show "HI" or "?" | VGA buffer, next available row |
| **Stack** | Not overflow | `boot.S` reserves 16 KB |

### Linker Script (cobos.ld)

```
ENTRY(_start)

SECTIONS {
    . = 1M;                    /* Multiboot standard: 1 MB */

    .multiboot : {
        *(.multiboot)
    }

    .text : {
        *(.text)
    }

    .rodata : {
        *(.rodata)
    }

    .data : {
        *(.data)
    }

    .bss : {
        *(COMMON)
        *(.bss)
    }
}
```

The kernel binary target is roughly 4–8 KB for a minimal kernel, dominated by the scancode table and VGA font data if included.

---

## 7. ESP32 Angle

### Why ESP32

- **Xtensa LX6 CPU**: LLVM has an Xtensa backend (though experimental). The same COBOL IR that targets x86-64 can target ESP32 with a different triple.
- **520KB SRAM + 4MB flash**: Enough for non-trivial COBOL programs. A COBOL PIC X(1000) field takes 1KB; hundreds of such fields fit.
- **Built-in WiFi + Bluetooth**: HTTP-GET and HTTP-POST verbs become meaningful on actual hardware. A COBOL program on an ESP32 can make real HTTP requests.
- **Cost**: ~$3 per chip. A COBOL-programmable IoT device at commodity pricing.

### What Changes vs x86 Bare Metal

| Aspect | x86-64 Bare Metal | ESP32 |
|--------|-------------------|-------|
| Triple | `x86_64-unknown-none` | `xtensa-esp32-none-elf` |
| Linker | `ld` + `cobos.ld` | ESP-IDF `idf.py` |
| DISPLAY | VGA text mode at `0xB8000` | UART at `0x3FF40000` |
| ACCEPT | PS/2 port `0x60` | UART RX |
| HTTP | Not available | `esp_http_client` |
| C-runtime | None | ESP-IDF provides partial libc |
| Boot | Multiboot + assembly stub | ESP-IDF bootloader + `app_main` |

### HTTP on ESP32

The `cob_http_get` runtime call would map to ESP-IDF's `esp_http_client` API rather than Go's `net/http`. The compiler declares the same external symbol, and the ESP32 runtime (written in C, part of the ESP-IDF project) implements it:

```c
// runtime/esp32/cob_http.c
#include <esp_http_client.h>

int cob_http_get(char *url, char *response, long int responseCap,
                 long int *responseLen, int *statusCode) {
    esp_http_client_config_t config = {
        .url = url,
        .method = HTTP_METHOD_GET,
    };
    esp_http_client_handle_t client = esp_http_client_init(&config);
    esp_http_client_perform(client);
    *statusCode = esp_http_client_get_status_code(client);
    // read response body...
    esp_http_client_cleanup(client);
    return 0;
}
```

### ESP32 Roadmap

After x86-bare metal boots, ESP32 support is:

1. Cross-compile COBOLabunga (Go → any host) or run COBOLabunga on x86 and cross-compile for ESP32.
2. Provide ESP32 target triple and runtime stubs.
3. Map DISPLAY to UART, ACCEPT to UART RX.
4. Map HTTP verbs to ESP-IDF.
5. Flash and run via `esptool.py`.

---

## 8. Phase 3 Roadmap

Ordered implementation steps, each producing a verifiable milestone.

### Step 1: x86-bare Target Triple in CLI

- Add `--target=x86-bare` flag in `main.go`.
- Set `targetTriple = "x86_64-unknown-none"`.
- Set `c.isBareMetal = true` in codegen.
- When bare metal, skip Go runtime declarations and libc declarations.
- Output is `prog.o` (ELF64 object).
- **Verify**: `llc` produces a valid `.o` file from a DISPLAY-less program. `readelf -h` shows `x86_64-unknown-none`.

### Step 2: VGA DISPLAY Implementation

- Add `emitVgaDisplay` function in codegen that emits inline VGA buffer writes.
- Dispatch in `emitDisplay` when `c.isBareMetal`.
- Emit a cursor position global (`cob.DISPLAY_CURSOR`) initialized to `0`.
- Character writes increment cursor; newline sets cursor to next row start.
- **Verify**: `echo "DISPLAY \"X\"." | ./cobolabunga --target=x86-bare` produces LLVM IR with `inttoptr` to `0xB8000` and `store volatile`.

### Step 3: Bootloader Stub

- Write `boot.S`: multiboot2 header, stack setup, BSS zero, call COBOL entry.
- Write `cobos.ld`: link at `0x100000`, include `.multiboot` section.
- Compile COBOL program → `prog.o`, link with `boot.o` and `cobos.ld` → `cobos.elf`.
- **Verify**: `qemu-system-x86_64 -kernel cobos.elf` boots and shows nothing (no VGA DISPLAY yet), or shows output from Step 2 if completed.

### Step 4: PEEK and POKE Verbs

- Add tokens `PEEK`, `POKE` (bare-metal only).
- Add parser methods `parsePeekStatement`, `parsePokeStatement`.
- Add codegen `emitPeek` / `emitPoke` with `volatile load/store` and `inttoptr`.
- **Verify**: `PEEK WS-VAL FROM WS-ADDR` emits `load volatile i32, ptr inttoptr(i32 %addr to ptr)`.

### Step 5: PORT-IN and PORT-OUT Verbs

- Add tokens `PORT_IN`, `PORT_OUT` (bare-metal only).
- Add parser methods implementing the `FROM PORT` / `TO PORT` syntax.
- Add codegen with inline assembly `inb` / `outb`.
- **Verify**: `PORT-IN WS-VAL FROM PORT 0x60` emits `call i8 asm sideeffect "inb $1, $0"`.

### Step 6: ACCEPT from Keyboard

- Add scancode-to-ASCII translation table as a compiler-emitted constant.
- Add `emitBareAccepts` function: poll port `0x64`, read port `0x60`, translate, store.
- Handle Enter (scancode `0x1C`) to terminate input.
- **Verify**: QEMU boots, COBOL kernel reads keystrokes.

### Step 7: Boot in QEMU

- Combine Steps 1–6.
- Full kernel: DISPLAY prompt, ACCEPT input, IF/THEN on input, PERFORM loop.
- **Verify**: `qemu-system-x86_64 -kernel cobos.elf` shows a working COBOL shell.

### Step 8: Boot on Real Hardware

- Write `cobos.elf` to a USB drive (or PXE boot, or burn to CD for El Torito).
- Boot on a physical x86-64 machine.
- Handle real hardware quirks: PS/2 vs USB keyboard, UEFI vs BIOS.
- **Verify**: COBOL kernel runs on bare metal, not just QEMU.

### Step 9: ESP32 Target

- Add `--target=esp32` flag.
- Write ESP32 runtime stubs (UART DISPLAY, UART ACCEPT, ESP-IDF HTTP).
- Cross-compile COBOL program for Xtensa.
- Flash via `esptool.py`.
- **Verify**: ESP32 prints COBOL DISPLAY output over serial, responds to UART input, makes HTTP requests.

---

## Summary

COBOS is a COBOL operating system kernel built entirely from COBOLabunga's LLVM IR output. It replaces the Go and libc runtimes with inline hardware access: VGA memory at `0xB8000`, PS/2 keyboard I/O at port `0x60`, and UART registers for ESP32. New verbs (PEEK, POKE, PORT-IN, PORT-OUT) give COBOL programs direct hardware access. The bootloader is a 30-line assembly stub. The full stack — from multiboot header to COBOL DISPLAY — is approximately 100 lines of new compiler code across the codegen, parser, and main.go files, plus the assembly linker script and boot stub.

Target binary size for a minimal kernel: under 8 KB.
