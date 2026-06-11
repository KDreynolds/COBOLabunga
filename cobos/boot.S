# COBOLabunga bare metal boot stub
# Multiboot2 compliant
# AT&T syntax (GNU as)

.set MAGIC,       0xE85250D6
.set ARCH,        0
.set HEADER_LEN,  (header_end - header_start)
.set CHECKSUM,    -(MAGIC + ARCH + HEADER_LEN)

.section .multiboot
.align 8
header_start:
    .long MAGIC
    .long ARCH
    .long HEADER_LEN
    .long CHECKSUM
    .short 0
    .short 0
    .long 8
header_end:

.section .text
.global _start
_start:
    mov $stack_top, %esp

    mov $__bss_start, %edi
    mov $__bss_end, %ecx
    sub %edi, %ecx
    xor %eax, %eax
    rep stosb

    call cob.KERNEL_MAIN

halt:
    cli
    hlt
    jmp halt

.section .bss
.align 16
stack_bottom:
    .skip 16384
stack_top:
