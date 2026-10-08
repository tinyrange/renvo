/* Linux x86-64 entry stack is aligned; calls use the ordinary SysV ABI. */
.section .text
.global _start
.type _start, @function
_start:
 xor %ebp, %ebp
 call main
 call select_validation_seeds
 call main
 xor %edi, %edi
 mov $60, %eax
 syscall
.size _start, .-_start
.section .note.GNU-stack,"",@progbits
