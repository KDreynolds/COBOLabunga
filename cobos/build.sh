#!/bin/bash
# Build a COBOL kernel with COBOLabunga
# Usage: ./cobos/build.sh path/to/kernel.cbl

set -e

COBOLABUNGA="./cobolabunga"
BOOT="cobos/boot.S"
LINKER="cobos/cobos.ld"
SOURCE=$1

if [ -z "$SOURCE" ]; then
    echo "Usage: $0 path/to/kernel.cbl"
    exit 1
fi

STEM=$(basename "$SOURCE" .cbl)

echo "=== COBOLabunga Kernel Build ==="
echo "Source:  $SOURCE"
echo "Output:  cobos/${STEM}.elf"
echo ""

echo "[1/4] Compiling COBOL → LLVM IR → .o ..."
$COBOLABUNGA "$SOURCE" --target x86-bare

echo "[2/4] Assembling boot stub..."
as --32 "$BOOT" -o cobos/boot.o

echo "[3/4] Linking..."
ld -m elf_i386 \
    -T "$LINKER" \
    cobos/boot.o "${STEM}.o" \
    -o "cobos/${STEM}.elf"

echo "[4/4] Done."
echo ""
echo "Binary: cobos/${STEM}.elf"
echo "Run:    qemu-system-x86_64 -kernel cobos/${STEM}.elf -device isa-debug-exit,iobase=0xf4,iosize=4"
