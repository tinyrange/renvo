//go:build !renvo && cgo && linux && amd64

.text
.globl renvo_rfe_enter
.type renvo_rfe_enter,@function
renvo_rfe_enter:
	// Preserve the System V callee-owned registers on the C stack, not the
	// Go goroutine stack. No Go callback or runtime-private entry is used.
	pushq %rbp
	pushq %rbx
	pushq %r12
	pushq %r13
	pushq %r14
	pushq %r15
	movq %rdi, %r10
	movq %rsi, %rax
	movq %rcx, %r11
	movq %rsp, %rcx
	movq %r11, %rsp
	subq $16, %rsp
	movq %rcx, 8(%rsp)
	// Native ABI: AX=state, DX=context, aligned private stack; generated
	// dispatcher establishes and preserves the rest of its ownership.
	call *%r10
	movq 8(%rsp), %rcx
	movq %rcx, %rsp
	popq %r15
	popq %r14
	popq %r13
	popq %r12
	popq %rbx
	popq %rbp
	ret
.size renvo_rfe_enter,.-renvo_rfe_enter
.section .note.GNU-stack,"",@progbits
