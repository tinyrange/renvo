.section .text
.globl _start
.type _start,@function
_start:
 lis 2,.TOC.@ha
 addi 2,2,.TOC.@l
 li 0,0
 stdu 1,-32(1)
 bl smoke
1: b 1b
.size _start,.-_start
.section .note.GNU-stack,"",@progbits
