# COBOLabunga bare metal boot stub
# Multiboot2 compliant (QEMU requires entry address tag for x86_64)
# AT&T syntax (GNU as)

.set MB2_MAGIC,     0xE85250D6
.set MB2_ARCH,      0
.set MB2_HEADER_LEN,(mb2_end - mb2_start)
.set MB2_CHECKSUM,  -(MB2_MAGIC + MB2_ARCH + MB2_HEADER_LEN)

.section .multiboot, "a"
.align 8

# Multiboot1 header (QEMU x86)
.set MB1_MAGIC,     0x1BADB002
.set MB1_FLAGS,     0x00000003
.set MB1_CHECKSUM,  -(MB1_MAGIC + MB1_FLAGS)
.long MB1_MAGIC
.long MB1_FLAGS
.long MB1_CHECKSUM

# Multiboot2 header
mb2_start:
    .long MB2_MAGIC
    .long MB2_ARCH
    .long MB2_HEADER_LEN
    .long MB2_CHECKSUM

    # Entry address tag (type=3, size=12)
    .short 3
    .short 0
    .long 12
    .long _start

    # Terminating tag
    .short 0
    .short 0
    .long 8
mb2_end:

.section .text, "ax"
.global _start
_start:
    mov $stack_top, %esp

    mov $__bss_start, %edi
    mov $__bss_end, %ecx
    sub %edi, %ecx
    xor %eax, %eax
    rep stosb

    # Clear VGA text mode screen (80x25)
    mov $0xB8000, %edi
    mov $2000, %ecx
    mov $0x0720, %eax
    rep stosw

    call cob.KERNEL_MAIN

    # Shutdown: write to QEMU debug exit port
    mov $0xF4, %dx
    mov $0, %al
    outb %al, %dx

halt:
    cli
    hlt
    jmp halt

.section .bss, "aw"
.align 16
stack_bottom:
    .skip 16384
stack_top:
