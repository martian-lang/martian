// Copyright (c) 2017 10X Genomics, Inc. All rights reserved.

// Implements the rt_sigaction syscall.
// Based off of https://golang.org/src/runtime/sys_linux_amd64.s

#include "textflag.h"

TEXT ·rt_sigaction(SB),NOSPLIT,$0-36
	MOVQ	sig+0(FP), DI
	MOVQ	new+8(FP), SI
	MOVQ	old+16(FP), DX
	MOVQ	size+24(FP), R10
	MOVL	$13, AX			// syscall entry
	SYSCALL
	MOVL	AX, ret+32(FP)
	RET
