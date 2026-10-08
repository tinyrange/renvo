/* Linux supplies a 16-byte aligned stack; no libc or dynamic runtime. */
.section .text
.global _start
.type _start, %function
_start:
 mov x29, xzr
 mov x30, xzr
 bl main
 bl select_validation_seeds
 bl main
 mov x0, #0
 mov x8, #93
 svc #0
.size _start, .-_start
.section .note.GNU-stack,"",%progbits
