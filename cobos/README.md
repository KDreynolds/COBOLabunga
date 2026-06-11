# COBOS Build System

Bare metal COBOL kernel build infrastructure for 
COBOLabunga.

## Requirements

- COBOLabunga compiler (`../cobolabunga`)
- LLVM (`llc`)
- GNU binutils (`as`, `ld`)
- QEMU (`qemu-system-x86_64`) for testing

## Build a kernel

    ./cobos/build.sh path/to/kernel.cbl

## Run in QEMU

    qemu-system-x86_64 -kernel cobos/kernel.elf

## Files

| File | Purpose |
|------|---------|
| boot.S | Multiboot2 stub, stack setup, BSS zero, calls COBOL entry |
| cobos.ld | Linker script, loads kernel at 1MB |
| build.sh | Full build pipeline |
| README.md | This file |

## What boot.S does

1. Embeds Multiboot2 header so GRUB/QEMU can load it
2. Sets up a 16KB stack
3. Zeros the BSS segment
4. Calls cob.KERNEL_MAIN — the COBOL PROCEDURE DIVISION entry
5. Halts on return (kernel should never return)

## Architecture

    kernel.cbl
        → cobolabunga --target x86-bare
        → kernel.ll (LLVM IR, no libc, no runtime)
        → llc → kernel.o
        → ld + cobos.ld + boot.o
        → kernel.elf (Multiboot2, loads at 0x100000)
        → qemu-system-x86_64 -kernel kernel.elf
        → COBOL running on bare metal
