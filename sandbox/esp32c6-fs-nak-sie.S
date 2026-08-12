	.option norvc
	.section .text
	.globl _start

	.macro sync_sample expected
	csrr t0, 0x804
	andi t0, t0, 3
	xor t3, t0, \expected
	or a3, a3, t3
	mv t4, t0
	.rept 5
	nop
	.endr
	.endm

	.macro pid_sample shift
	csrr t0, 0x804
	andi t0, t0, 3
	xor t3, t0, t4
	sltiu t3, t3, 1
	slli t3, t3, \shift
	or a2, a2, t3
	mv t4, t0
	.rept 3
	nop
	.endr
	.endm

	.macro txcell value, delay
	.if GPIO_TX
	.if tx_index == 0
	.rept GPIO_CELL_NOPS
	nop
	.endr
	.else
	csrw 0x805, \value
	.rept GPIO_CELL_NOPS
	nop
	.endr
	.endif
	.set tx_index, tx_index + 1
	.else
	.if TEST_SCHEDULED
.Ltxcell_wait_\@:
	csrr a6, 0x7e2
	bltu a6, a4, .Ltxcell_wait_\@
	sw \value, 0(s3)
	addi a4, a4, SCHEDULE_CYCLES
	.if ((tx_index + SCHEDULE_PHASE) % SCHEDULE_PERIOD) == (SCHEDULE_PERIOD - 1)
	addi a4, a4, SCHEDULE_EXTRA
	.endif
	.else
	sw \value, 0(s3)
	.if TEST_EXACT
	.rept TEST_CELL_NOPS
	nop
	.endr
	.elseif TEST_PATTERN
	.if ((tx_index + tx_phase_offset) % TX_PERIOD) == (TX_PERIOD - 1)
	.rept TX_PATTERN_NOPS
	nop
	.endr
	.endif
	.else
	.rept \delay
	nop
	.endr
	.endif
	.endif
	.set tx_index, tx_index + 1
	.endif
	.endm

	.macro txsym value
	.if GPIO_TX
	.if tx_index == 0
	.rept GPIO_CELL_NOPS
	nop
	.endr
	.else
	csrw 0x805, \value
	.rept GPIO_CELL_NOPS
	nop
	.endr
	.endif
	.else
	.if TEST_SCHEDULED
.Ltxsym_wait_\@:
	csrr a6, 0x7e2
	bltu a6, a4, .Ltxsym_wait_\@
	sw \value, 0(s3)
	addi a4, a4, SCHEDULE_CYCLES
	.if ((tx_index + SCHEDULE_PHASE) % SCHEDULE_PERIOD) == (SCHEDULE_PERIOD - 1)
	addi a4, a4, SCHEDULE_EXTRA
	.endif
	.else
	sw \value, 0(s3)
	.if TEST_EXACT
	.rept TEST_CELL_NOPS
	nop
	.endr
	.else
	.if ((tx_index + tx_phase_offset) % TX_PERIOD) == (TX_PERIOD - 1)
	.rept TX_PATTERN_NOPS
	nop
	.endr
	.endif
	.endif
	.endif
	.endif
	.set tx_index, tx_index + 1
	.endm

_start:
	li s0, 0x4087f000
	li t0, 0x600b1020
	li t1, 0x53544152         // STAR, retained across HP watchdog reset
	sw t1, 0(t0)
	li t0, 0x600b101c         // retained stage bitmap for ROM-load diagnosis
	sw zero, 0(t0)
	li t0, 0x600b1024         // Renvo USB recovery marker
	li t1, 0x52454e56         // RENV
	sw t1, 0(t0)
	li s1, 0x6000f018
	lw s2, 0(s1)
	li s3, 0x6000f01c
	li s4, 0x60090038
	li s5, 0x600911c4
	lw s6, 0(s4)
	lw s7, 4(s4)
	lw s8, 0(s5)
	lw s9, 4(s5)
	li t0, 0x60091584
	lw t1, 0(t0)
	sw t1, 64(s0)
	lw t1, 4(t0)
	sw t1, 68(s0)
	li t0, 0x6009608c
	lw t1, 0(t0)
	sw t1, 72(s0)
	li s10, 1                 // dedicated K input value
	li s11, 2                 // dedicated J input value

	// Result block: reset, SETUP, DATA0, IN, ACK sent, NAK sent, done.
	sw zero, 0(s0)
	sw zero, 4(s0)
	sw zero, 8(s0)
	sw zero, 12(s0)
	sw zero, 16(s0)
	sw zero, 20(s0)
	sw zero, 24(s0)
	sw zero, 28(s0)
	sw zero, 32(s0)
	sw zero, 36(s0)
	sw zero, 40(s0)
	sw zero, 44(s0)
	sw zero, 48(s0)
	sw zero, 52(s0)
	sw zero, 56(s0)
	sw zero, 60(s0)
	sw zero, 76(s0)
	sw zero, 80(s0)

	// Exact ten-cycle full-speed bit clock.
	li t0, 0x60096118
	li t1, 0x10000
	sw t1, 0(t0)
	.if MMIO_CPU80
	li t1, 0x100            // PLL / 6 = 80 MHz, synchronous with APB
	sw t1, 0(t0)
	.endif
	.if MMIO_CPU160
	sw zero, 0(t0)           // PLL / 3 = 160 MHz
	.endif

	.if MMIO_TIMING
	// Measure CPU-visible latency of a TEST_REG write at the forced 120 MHz
	// clock. Pack minimum and maximum cycle counts into the retained marker.
	li t0, 0x6000f01c
	li a0, 128
	li a1, 0xffff
	li a2, 0
97:
	csrr t1, 0x7e2
	sw zero, 0(t0)
	csrr t2, 0x7e2
	sub t2, t2, t1
	bltu t2, a1, 95f
	j 94f
95:
	mv a1, t2
94:
	bltu a2, t2, 93f
	j 92f
93:
	mv a2, t2
92:
	addi a0, a0, -1
	bnez a0, 97b
	// Confirm the elapsed time of sixteen stores on the intended ten-cycle
	// packet cadence; this also exposes any APB write-buffer backpressure.
	csrr t1, 0x7e2
	.if MMIO_PATTERN
	li t3, 7
	li t4, 11
	.set mmio_index, 0
	.rept 96
	.if (mmio_index % 2) == 0
	sw t3, 0(t0)
	.else
	sw t4, 0(t0)
	.endif
	.rept MMIO_GAP
	nop
	.endr
	.if ((mmio_index + TX_PATTERN_PHASE) % TX_PERIOD) == (TX_PERIOD - 1)
	.rept TX_PATTERN_NOPS
	nop
	.endr
	.endif
	.set mmio_index, mmio_index + 1
	.endr
	.else
	.rept 16
	sw zero, 0(t0)
	.rept MMIO_GAP
	nop
	.endr
	.endr
	.endif
	csrr t2, 0x7e2
	sub t2, t2, t1
	sw zero, 0(t0)
	slli t2, t2, 16
	or a1, a1, t2
	li t0, 0x600b1020
	sw a1, 0(t0)
96:
	j 96b
	.endif

	// Recovery watchdog, approximately five seconds. Successful enumeration
	// disables it; failed timing experiments promptly restore the ROM device.
	li t0, 0x6009603c         // enable TIMERGROUP0 peripheral clock
	lw t1, 0(t0)
	ori t1, t1, 1
	andi t1, t1, -3
	sw t1, 0(t0)
	li t0, 0x60096044         // enable TIMERGROUP0 watchdog clock
	lw t1, 0(t0)
	li t2, 0x400000
	or t1, t1, t2
	sw t1, 0(t0)
	li t0, 0x60008064
	li t1, 0x50d83aa1
	sw t1, 0(t0)
	li t0, 0x60008048
	sw zero, 0(t0)
	li t0, 0x6000804c
	li t1, 0x4e200001
	sw t1, 0(t0)
	li t0, 0x60008050
	li t1, WATCHDOG_TICKS
	sw t1, 0(t0)
	li t0, 0x60008060
	li t1, 1
	sw t1, 0(t0)
	li t0, 0x60008048
	li t1, 0xe0400000
	sw t1, 0(t0)
	li t0, 0x60008064
	sw zero, 0(t0)

	// D- -> dedicated bit 0; D+ -> bit 1, preserving USB pin function.
	ori t0, s6, 0x200
	sw t0, 0(s4)
	ori t0, s7, 0x200
	sw t0, 4(s4)
	li t0, 0x8c
	sw t0, 0(s5)
	li t0, 0x8d
	sw t0, 4(s5)

	.if GPIO_TX || GPIO_ACK
	.if RESET_FIXED
	// Stop the fixed CDC/JTAG SIE without disabling the USB PHY clock.
	li t0, 0x6009608c
	lw t1, 0(t0)
	ori t1, t1, 2
	sw t1, 0(t0)
	.endif
	// Route both USB pads to the HP hart's dedicated-GPIO channels.  Keep the
	// digital input enabled so the same CSR continues to receive D-/D+.
	csrw 0x803, zero
	li t0, -0x7001
	and t1, s6, t0
	li t2, 0x1200           // GPIO function 1 + input enable
	or t1, t1, t2
	sw t1, 0(s4)
	and t1, s7, t0
	or t1, t1, t2
	sw t1, 4(s4)
	li t0, 0x60091584
	li t1, 28               // GPIO12 <- CPU_GPIO_OUT0
	sw t1, 0(t0)
	li t1, 29               // GPIO13 <- CPU_GPIO_OUT1
	sw t1, 4(t0)
	li t1, 2                // idle full-speed J, still high impedance
	csrw 0x805, t1
	.endif

	// Detach, release the raw driver, and attach with the internal D+ pull-up.
	li t0, -0x1f01
	and t1, s2, t0
	li t0, 0x4100
	or t1, t1, t0
	sw t1, 0(s1)
	sw zero, 0(s3)
	// Give the ROM loader enough time to finish closing its fixed USB CDC
	// endpoint before this RAM image disconnects that controller.
	li t2, 12000000
1:
	addi t2, t2, -1
	bnez t2, 1b
	.if (!GPIO_TX) || HYBRID_TX
	li t0, 1
	sw t0, 0(s3)
	.endif
	ori t1, t1, 0x200
	sw t1, 0(s1)

	// Observe a sustained reset and its release.
2:
	csrr t0, 0x804
	andi t0, t0, 3
	bnez t0, 2b
	li t2, 200
3:
	csrr t0, 0x804
	andi t0, t0, 3
	bnez t0, 2b
	addi t2, t2, -1
	bnez t2, 3b
4:
	csrr t0, 0x804
	andi t0, t0, 3
	bne t0, s11, 4b
	li t0, 0x52535431         // RST1
	sw t0, 0(s0)
	li t1, 0x600b1020
	sw t0, 0(t1)
	.if RESET_FIXED_LATE
	li t0, 0x6009608c
	lw t1, 0(t0)
	ori t1, t1, 2
	sw t1, 0(t0)
	.endif
	.if GPIO_DRIVE_TEST
	li t0, 1                // force generic K
	csrw 0x805, t0
	li t0, 3
	csrw 0x803, t0
	.rept 100
	nop
	.endr
	csrr a0, 0x804          // sample while PHY test mode owns the pad
	andi a0, a0, 3
	sw zero, 0(s3)
	.rept 100
	nop
	.endr
	csrr a1, 0x804          // sample after releasing PHY test ownership
	andi a1, a1, 3
	csrw 0x803, zero
	li t0, 1
	sw t0, 0(s3)
	slli a1, a1, 4
	or a0, a0, a1
	li t0, 0x44520000       // DR + test-on/off line-state nibbles
	or a0, a0, t0
	li t0, 0x600b1020
	sw a0, 0(t0)
	j finish_restore
	.endif
	.if GPIO_ATTACH_ONLY
	li t2, 12000000
98:
	addi t2, t2, -1
	bnez t2, 98b
	j finish_restore
	.endif

	li t0, 1
	csrw 0x7e0, t0
	li t0, 3
	csrw 0x7e1, t0
	csrw 0x7e2, zero
	li a7, 0

setup_wait:
	call receive_packet
	.if DIAGNOSTIC
	lw t0, 28(s0)
	li t1, 0x4087e000
	slli t2, t0, 2
	add t1, t1, t2
	sw a0, 0(t1)
	addi t0, t0, 1
	sw t0, 28(s0)
	li t1, 16
	beq t0, t1, finish_restore
	.endif
	call count_stage_attempt
	beqz a3, stage_fail
	li t0, 0xc3              // DATA0 following the initial SETUP token
	bne a0, t0, setup_wait
	sw zero, 32(s0)
	lw t0, 4(s0)
	addi t0, t0, 1
	sw t0, 4(s0)
	lw t0, 8(s0)
	addi t0, t0, 1
	sw t0, 8(s0)
	li t0, 0x44415430         // DAT0
	li t1, 0x600b1020
	sw t0, 0(t1)
	call transmit_ack
	li t0, 0x41434b31         // ACK1
	sw t0, 16(s0)

in_wait:
	call receive_packet
	call count_stage_attempt
	beqz a3, stage_fail
	li t0, 0x69              // IN
	beq a0, t0, got_in
	li t0, 0x2d              // host repeated SETUP: ACK its DATA0 again
	bne a0, t0, in_wait
	j setup_wait

got_in:
	sw zero, 32(s0)
	lw t0, 12(s0)
	addi t0, t0, 1
	sw t0, 12(s0)
	li t0, 0x494e3031         // IN01
	li t1, 0x600b1020
	sw t0, 0(t1)
initial_data_response:
	.if FIRST_FULL
	call transmit_data1_18_dead
	.elseif FIRST_QUAL
	call transmit_qualifier_full
	.elseif FIRST_CONFIG
	call transmit_config_full
	.elseif FIRST_ZLP
	call transmit_data1_zero
	.elseif FIRST_TWO
	call transmit_data1_2
	.else
	call transmit_data1_8
	.endif
	.if !STOP_AFTER_FIRST_C
	// The status OUT proves acceptance. A repeated IN means the data packet was
	// lost and must be retransmitted with the same toggle.
	li a5, 0x69e1
	call receive_matching_two
	li t0, 0x69
	beq a0, t0, initial_data_response
	j continue_after_first_out
	.endif
	// The host ACK proves acceptance of the pre-encoded data and CRC16.
ack_wait:
	call receive_packet
	li t0, 0xd2
	beq a0, t0, got_host_ack
	li t0, 0x2d            // a fresh SETUP restarts address-zero probing
	beq a0, t0, setup_wait
	li t0, 0x69
	bne a0, t0, ack_wait
	.if FIRST_FULL
	call transmit_data1_18_dead
	.elseif FIRST_QUAL
	call transmit_qualifier_full
	.elseif FIRST_CONFIG
	call transmit_config_full
	.elseif FIRST_ZLP
	call transmit_data1_zero
	.elseif FIRST_TWO
	call transmit_data1_2
	.else
	call transmit_data1_8
	.endif
	j ack_wait
got_host_ack:
	.if STOP_AFTER_FIRST_C
	li t0, 0x4841434b         // HACK
	sw t0, 24(s0)
	li t1, 0x600b1020
	sw t0, 0(t1)
	j finish_restore
	.else
	// Re-arm immediately: the host may start the status OUT token after only
	// two bit times of inter-packet delay.
	j continue_after_first
	.endif

	// USB 2.0 full-speed devices are queried for a high-speed qualifier. This
	// device has no high-speed mode, but a well-formed qualifier keeps Linux's
	// enumeration sequence deterministic for the software-only experiment.
qualifier_start:
	li a5, 0x2d
	call receive_matching_packet
	li a5, 0xc3
	call receive_matching_packet
	call transmit_ack
	li a5, 0x69
	call receive_matching_packet
	call transmit_stall
	lw t0, 40(s0)
	addi t0, t0, 1
	sw t0, 40(s0)
	li t1, 3
	bltu t0, t1, qualifier_start

	// Linux next fetches the nine-byte configuration header. With EP0 MPS 8,
	// that is a DATA1 packet followed by a one-byte DATA0 packet.
config_start:
	li a5, 0x2d
	call receive_matching_packet
config_header_setup_data:
	li a5, 0xc3
	call receive_matching_packet
	call transmit_ack
	li a5, 0x2d69          // IN, or a retried SETUP if our ACK was lost
	call receive_matching_two
	li t0, 0x2d
	beq a0, t0, config_header_setup_data
config_header_response:
	call transmit_config_full
	li a5, 0x69e1          // retried IN, or status OUT proving acceptance
	call receive_matching_two
	li t0, 0x69
	beq a0, t0, config_header_response
	j config_header_status_data
config_header_first_ack:
	call receive_packet
	li t0, 0xd2
	beq a0, t0, config_header_acked
	li t0, 0x2d
	beq a0, t0, config_header_setup_data
	li t0, 0x69
	bne a0, t0, config_header_first_ack
	call transmit_config_full
	j config_header_first_ack
config_header_acked:
	li a5, 0xe1
	call receive_matching_packet
config_header_status_data:
	li a5, 0x4b
	call receive_matching_packet
config_header_status_ack:
	call transmit_ack
	li a5, 0x2de1          // retried status OUT, or next request SETUP
	call receive_matching_two
	li t0, 0xe1
	beq a0, t0, config_header_status_data
	j config_setup_data     // next SETUP was consumed

	// Fetch of the complete minimal nine-byte configuration.
	config_setup:
	li a5, 0x2d
	call receive_matching_packet
config_setup_data:
	li a5, 0xc3
	call receive_matching_packet
	call transmit_ack
	li a5, 0x2d69          // IN, or a retried SETUP if our ACK was lost
	call receive_matching_two
	li t0, 0x2d
	beq a0, t0, config_setup_data
config_response:
	call transmit_config_full
	li a5, 0x69e1          // retried IN, or status OUT proving acceptance
	call receive_matching_two
	li t0, 0x69
	beq a0, t0, config_response
	j config_status_data
config_first_ack:
	call receive_packet
	li t0, 0xd2
	beq a0, t0, config_acked
	li t0, 0x2d
	beq a0, t0, config_setup_data
	li t0, 0x69
	bne a0, t0, config_first_ack
	call transmit_config_full
	j config_first_ack
config_acked:
	li a5, 0xe1
	call receive_matching_packet
config_status_data:
	li a5, 0x4b
	call receive_matching_packet
config_status_ack:
	call transmit_ack
	li a5, 0x2de1          // retried status OUT, or SET_CONFIGURATION SETUP
	call receive_matching_two
	li t0, 0xe1
	beq a0, t0, config_status_data
	j set_configuration_data

	// SET_CONFIGURATION has no data phase and completes with an IN DATA1 ZLP.
	set_configuration_start:
	li a5, 0x2d
	call receive_matching_packet
	set_configuration_data:
	li a5, 0xc3
	call receive_matching_packet
	call transmit_ack
	li a5, 0x2d69
	call receive_matching_two
	li t0, 0x2d
	beq a0, t0, set_configuration_data
	call transmit_data1_zero
	li t0, 0x43464744         // CFGD
	li t1, 0x600b1020
	sw t0, 0(t1)

	// Enumeration succeeded. Disarm recovery while leaving the custom PHY
	// personality attached for host-side verification.
	li t0, 0x60008064
	li t1, 0x50d83aa1
	sw t1, 0(t0)
	li t0, 0x60008048
	sw zero, 0(t0)
	li t0, 0x60008064
	sw zero, 0(t0)
	li t0, 0x600b1024
	sw zero, 0(t0)
configured_loop:
	call receive_packet
	li t0, 0x69
	bne a0, t0, configured_loop
	call transmit_nak
	j configured_loop
continue_after_first:
	.if CAPTURE_AFTER_ACK
	// Capture timestamped line transitions beginning at the ACK's EOP J.  The
	// host controller may defer the status transaction for hundreds of
	// microseconds, so an edge log is more useful than a short dense sample.
	li t1, 0x4087d000
	li t2, 256
	csrr a0, 0x7e2
	li a2, 300000
	li a3, 2
77:
	csrr t0, 0x804
	andi t0, t0, 3
	beq t0, a3, 78f
	csrr a1, 0x7e2
	sub a1, a1, a0
	slli a3, t0, 30
	or a1, a1, a3
	sw a1, 0(t1)
	addi t1, t1, 4
	addi t2, t2, -1
	beqz t2, 79f
	mv a3, t0
78:
	csrr a1, 0x7e2
	sub a1, a1, a0
	bltu a1, a2, 77b
79:
	li t0, 256
	sub t0, t0, t2
	sw t0, 36(s0)
	j finish_restore
	.endif
	// Complete the control-read status transaction before waiting for the next
	// setup request. The host sends an OUT token and DATA1 ZLP; omitting this ACK
	// makes usbcore report -EPROTO even though it accepted all descriptor bytes.
	li a5, 0xe1              // OUT
	call receive_matching_packet
continue_after_first_out:
	li a5, 0x4b              // DATA1 ZLP
	call receive_matching_packet
initial_status_ack:
	call transmit_ack
	.if STOP_AFTER_STATUS
	li t0, 0x54415453         // STAT
	li t1, 0x600b1020
	sw t0, 0(t1)
	li t2, 120000
90:
	addi t2, t2, -1
	bnez t2, 90b
	j finish_restore
	.endif
	li a7, 1                 // initial descriptor transfer completed
	li a5, 0x2de1            // retried status OUT, or SET_ADDRESS SETUP
	call receive_matching_two
	li t0, 0xe1
	beq a0, t0, continue_after_first_out
	j set_address_setup_data  // SETUP was consumed

	// Linux now performs SET_ADDRESS. Skip SOFs and controller housekeeping.
	li a5, 0x2d              // SETUP token
	call receive_matching_packet
set_address_setup_data:
	li a5, 0xc3              // DATA0 request
	call receive_matching_packet
	call transmit_ack
	ori a7, a7, 2

	// The no-data control status stage is a zero-length DATA1 packet.
	li a5, 0x69              // IN
	call receive_matching_packet
set_address_response:
	call transmit_data1_zero
	li a5, 0x2d69            // retried IN, or next SETUP proving acceptance
	call receive_matching_two
	li t0, 0x69
	beq a0, t0, set_address_response
	ori a7, a7, 4
	j descriptor_setup_data    // the next SETUP token was consumed above

	// A subsequent DATA0 setup payload proves the host moved on to descriptor
	// retrieval using the newly assigned device address.
	descriptor_setup:
	li a5, 0x2d              // SETUP at the new address
	call receive_matching_packet
descriptor_setup_data:
	li a5, 0xc3              // DATA0 GET_DESCRIPTOR request
	call receive_matching_packet
	call transmit_ack
	ori a7, a7, 8

	li a5, 0x2d69            // IN, or retried SETUP after a lost ACK
	call receive_matching_two
	li t0, 0x2d
	beq a0, t0, descriptor_setup_data
descriptor_data_response:
	call transmit_data1_18_dead
	li a5, 0x69e1            // retried IN, or status OUT proving acceptance
	call receive_matching_two
	li t0, 0x69
	beq a0, t0, descriptor_data_response
	j descriptor_status_data
descriptor_first_ack:
	call receive_packet
	li t0, 0xd2
	beq a0, t0, descriptor_acked
	li t0, 0x2d
	beq a0, t0, descriptor_setup_data
	li t0, 0x69
	bne a0, t0, descriptor_first_ack
	call transmit_data1_18_dead
	j descriptor_first_ack
descriptor_acked:
	ori a7, a7, 64

	// Host terminates the control read with an OUT DATA1 zero-length packet.
	li a5, 0xe1              // OUT
	call receive_matching_packet
descriptor_status_data:
	ori a7, a7, 64
	li a5, 0x4b              // DATA1 ZLP
	call receive_matching_packet
descriptor_status_ack:
	call transmit_ack
	li a5, 0x2de1            // retried status OUT, or configuration SETUP
	call receive_matching_two
	li t0, 0xe1
	beq a0, t0, descriptor_status_data
	ori a7, a7, 128
	li t0, 0x46444553         // FDES
	sw t0, 28(s0)
	sw a7, 60(s0)
	li t1, 0x600b101c
	sw a7, 0(t1)
	li t1, 0x600b1020
	sw t0, 0(t1)
	// USB 1.1 full-speed devices have no device-qualifier request.
	j config_header_setup_data // configuration SETUP was consumed

stage_fail:
	li t0, 0x41464149         // AFAI
	sw t0, 28(s0)
	sw a7, 60(s0)
	li t1, 0x600b101c
	sw a7, 0(t1)
	li t1, 0x600b1020
	sw t0, 0(t1)

	// Restore fixed USB-JTAG.
finish_restore:
	csrw 0x803, zero
	li t0, 0x6009608c
	lw t1, 72(s0)
	sw t1, 0(t0)
	lw t1, 0(s1)
	li t0, -0x201
	and t1, t1, t0
	sw t1, 0(s1)
	sw zero, 0(s3)
	li t2, 600000
5:
	addi t2, t2, -1
	bnez t2, 5b
	sw s2, 0(s1)
	sw s8, 0(s5)
	sw s9, 4(s5)
	li t0, 0x60091584
	lw t1, 64(s0)
	sw t1, 0(t0)
	lw t1, 68(s0)
	sw t1, 4(t0)
	sw s6, 0(s4)
	sw s7, 4(s4)
	li t0, 0x60008064
	li t1, 0x50d83aa1
	sw t1, 0(t0)
	li t0, 0x60008048
	sw zero, 0(t0)
	li t0, 0x60008064
	sw zero, 0(t0)
6:
	j 6b

// Return the first packet matching a5 without leaving the streaming receive
// loop between non-target packets.
receive_matching_packet:
	li a6, 1
	j receive_packet_start

// Return on either PID packed into the low and high bytes of a5.
receive_matching_two:
	li a6, 2
	j receive_packet_start

// Return PID in a0 and the cycle count at the center of the first SE0 in a1.
receive_packet:
	li a6, 0
receive_packet_start:
	li a4, 240000
7:
	csrr t0, 0x804
	andi t0, t0, 3
	beq t0, s11, 70f
	addi a4, a4, -1
	bnez a4, 7b
	li a0, 0
	beqz a6, receive_return
	j receive_packet_start
70:
	li a4, 240000
7:
	csrr t0, 0x804
	andi t0, t0, 3
	beq t0, s10, 71f
	addi a4, a4, -1
	bnez a4, 7b
	li a0, 0
	beqz a6, receive_return
	j receive_packet_start
71:
	.rept RX_CENTER
	nop
	.endr

	// SYNC K J K J K J K K.  Never dispatch a PID from a packet whose
	// synchronization field was sampled at the wrong phase.
	li a3, 0
	sync_sample s10
	sync_sample s11
	sync_sample s10
	sync_sample s11
	sync_sample s10
	sync_sample s11
	sync_sample s10
	sync_sample s10

	li a2, 0
	pid_sample 0
	pid_sample 1
	pid_sample 2
	pid_sample 3
	pid_sample 4
	pid_sample 5
	pid_sample 6
	pid_sample 7

8:
	csrr t0, 0x804
	andi t0, t0, 3
	beqz t0, 9f
	.rept 6
	nop
	.endr
	j 8b
9:
	csrr a1, 0x7e2
	// Validate during the two-cell SE0. Returning at the first EOP J edge leaves
	// the complete inter-packet gap available to arm for a back-to-back token.
	li a0, 0
	bnez a3, 10f
	andi t0, a2, 0x0f
	srli t1, a2, 4
	xori t1, t1, 0x0f
	andi t1, t1, 0x0f
	bne t0, t1, 10f
	mv a0, a2
10:
	csrr t0, 0x804
	andi t0, t0, 3
	beqz t0, 10b
	beqz a6, receive_return
	li t0, 1
	beq a6, t0, receive_match_one
	andi t0, a5, 0xff
	beq a0, t0, receive_matched
	srli t0, a5, 8
	bne a0, t0, receive_packet_start
	j receive_matched
receive_match_one:
	bne a0, a5, receive_packet_start
receive_matched:
	li a6, 0
receive_return:
	ret

count_stage_attempt:
	lw t1, 32(s0)
	addi t1, t1, 1
	sw t1, 32(s0)
	li t0, 256
	sltu a3, t1, t0
	ret

// Wait for the next packet boundary and return the first-SE0 cycle in a1.
// This is used only for the known OUT/DATA1 status transaction in the probe.
wait_for_eop:
	li a4, 240000
74:
	csrr t0, 0x804
	andi t0, t0, 3
	beqz t0, 75f
	addi a4, a4, -1
	bnez a4, 74b
	li a3, 0
	ret
75:
	csrr a1, 0x7e2
76:
	csrr t0, 0x804
	andi t0, t0, 3
	beqz t0, 76b
	li a3, 1
	ret

wait_for_pid:
	mv a6, ra
	sw zero, 32(s0)
wait_for_pid_loop:
	call receive_packet
	.if TRACE_STAGE
	lw t0, 36(s0)
	li t1, 32
	bgeu t0, t1, 73f
	li t1, 0x4087e000
	slli t2, t0, 2
	add t1, t1, t2
	sw a0, 0(t1)
	addi t0, t0, 1
	sw t0, 36(s0)
73:
	.endif
	beq a0, a5, wait_for_pid_found
	call count_stage_attempt
	bnez a3, wait_for_pid_loop
	mv ra, a6
	ret
wait_for_pid_found:
	li a3, 1
	mv ra, a6
	ret

// Target the middle of the legal response window.  The first PHY store takes
// about 75 ns to become visible at the pad, so issue it roughly one cell early.
	.macro response_wait
	addi t2, a1, RESPONSE_OFFSET
.Lresponse_wait_\@:
	csrr t1, 0x7e2
	bltu t1, t2, .Lresponse_wait_\@
	.rept TX_PHASE
	nop
	.endr
	.endm

	.macro tx_start phase=TX_PATTERN_PHASE
	.set tx_phase_offset, \phase
	.if GPIO_TX
	li t4, 2                 // J: D+ on dedicated bit 1
	li t5, 1                 // K: D- on dedicated bit 0
	li t6, 0                 // SE0
	csrw 0x805, t5           // preload SOP while the pads are released
	.if HYBRID_TX
	// The fixed SIE did not observe the token while PHY test mode was active.
	// Hold that SIE in reset while releasing the analog override so its fixed
	// transmitter cannot contend with the dedicated-GPIO waveform.
	.if RESET_DURING_TX
	li t3, 0x6009608c
	lw t2, 0(t3)
	ori t2, t2, 2
	sw t2, 0(t3)
	.endif
	sw zero, 0(s3)
	.endif
	response_wait
	li t0, 3
	csrw 0x803, t0           // atomically expose K at SOP
	.else
	li t4, 7                 // J: test + OE + DP
	li t5, 11                // K: test + OE + DM
	li t6, 3                 // SE0
	li t0, 1                 // test mode with PHY output released
	.if TX_DITHER
	// Walk through CPU/PHY relative phases across USB retries. The internal
	// packet cadence is unchanged; only SOP moves within the legal window.
	lw t3, 76(s0)
	addi t2, t3, 1
	sw t2, 76(s0)
	andi t3, t3, 7
	beqz t3, .Ltxdither_done_\@
.Ltxdither_\@:
	addi t3, t3, -1
	bnez t3, .Ltxdither_\@
.Ltxdither_done_\@:
	.endif
	.if TX_CPU160
	response_wait
	li t3, 0x60096118
	sw zero, 0(t3)           // switch packet engine to 160 MHz
	// Fill the APB write pipeline with electrically harmless release values so
	// the first driving store starts from a repeatable bridge phase.
	.rept TX_PRIME_WRITES
	sw t0, 0(s3)
	.endr
	.elseif TEST_SCHEDULED
	.if SCHEDULE_CPU160
	response_wait
	li t3, 0x60096118
	sw zero, 0(t3)
	csrr a4, 0x7e2
	addi a4, a4, SCHEDULE_START
	.else
	.rept TX_PRIME_WRITES
	sw t0, 0(s3)
	.endr
	addi a4, a1, RESPONSE_OFFSET
	.endif
	.else
	response_wait
	.if TEST_PATTERN
	// Harmless release writes put the APB bridge into a repeatable saturated
	// phase before the first pad-driving symbol reaches TEST_REG.
	li t3, 5                 // test + DP value, output still released
	.rept TX_PRIME_WRITES
	sw t3, 0(s3)
	xori t3, t3, 12          // alternate DP/DM without enabling the driver
	.endr
	.endif
	.endif
	.endif
	.set tx_index, 0
	.endm

	.macro tx_end release_nops=0
	.if GPIO_TX
	// Two exact SE0 cells, one J cell, then release both pads.
	csrw 0x805, t6
	.rept (2 * (GPIO_CELL_NOPS + 1) - 1)
	nop
	.endr
	csrw 0x805, t4
	.rept GPIO_CELL_NOPS
	nop
	.endr
	csrw 0x803, zero
	.if HYBRID_TX
	li t0, 1
	sw t0, 0(s3)             // return the pad to test-mode receive ownership
	.if RESET_DURING_TX
	li t3, 0x6009608c
	lw t2, 0(t3)
	andi t2, t2, -3
	sw t2, 0(t3)
	.endif
	.endif
	.else
	.if TEST_SCHEDULED
	txsym t6
	txsym t6
	txsym t4
.Ltxrelease_wait_\@:
	csrr a6, 0x7e2
	bltu a6, a4, .Ltxrelease_wait_\@
	sw t0, 0(s3)
	.if SCHEDULE_CPU160
	li t3, 0x60096118
	li t2, 0x10000
	sw t2, 0(t3)
	.endif
	.else
	sw t6, 0(s3)
	.if TEST_EXACT
	.rept (2 * (TEST_CELL_NOPS + 1) - 1)
	nop
	.endr
	sw t4, 0(s3)
	.rept TEST_CELL_NOPS
	nop
	.endr
	li t0, 1
	sw t0, 0(s3)
	.elseif TEST_PATTERN
	txsym t6
	.if EOP_EXTRA_SE0
	txsym t6
	.endif
	.if EOP_RELEASE_J
	// Releasing the internal PHY driver makes the enabled D+ pull-up produce
	// EOP J and then hold idle. This avoids a redundant, phase-sensitive write
	// of driven J followed by a separate high-impedance write.
	txsym t0
	.else
	txsym t4
	.rept \release_nops
	nop
	.endr
	sw t0, 0(s3)
	.endif
	.if TX_CPU160
	li t3, 0x60096118
	li t2, 0x10000
	sw t2, 0(t3)             // return receiver to exact 120 MHz timing
	.endif
	.else
	.rept 10
	nop
	.endr
	sw t4, 0(s3)
	li t0, 1
	sw t0, 0(s3)
	.endif
	.endif
	.endif
	.endm

	.if (!GPIO_TX) && (!TEST_EXACT) && (!TEST_PATTERN) && (!TEST_SCHEDULED)
	.org 0x980
	.endif
	.balign 64
transmit_ack:
	.if GPIO_ACK
	// Use the HP hart's single-cycle dedicated-GPIO CSRs for the latency-
	// critical handshake. PHY test mode still owns the pads during receive.
	li t4, 2
	li t5, 1
	li t6, 0
	csrw 0x805, t5
	response_wait
	sw zero, 0(s3)
	li t0, 3
	csrw 0x803, t0
	.set tx_index, 0
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t4,t4,t5,t4,t4,t5,t5,t5
	.if tx_index != 0
	csrw 0x805, \symbol
	.endif
	.rept 9
	nop
	.endr
	.set tx_index, tx_index + 1
	.endr
	csrw 0x805, t6
	.rept 19
	nop
	.endr
	csrw 0x805, t4
	.rept 9
	nop
	.endr
	csrw 0x803, zero
	li t0, 1
	sw t0, 0(s3)
	ret
	.else
	.macro ack_packet phase, release
	tx_start \phase
	// SYNC + ACK PID D2: KJKJKJKK JJKJJKKK.
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t4,t4,t5,t4,t4,t5,t5,t5
	txsym \symbol
	.endr
	tx_end \release
	ret
	.endm
	.if ACK_ROTATE
	lw t3, 80(s0)
	addi t2, t3, 1
	sw t2, 80(s0)
	andi t3, t3, 3
	beqz t3, transmit_ack_phase0
	li t2, 1
	beq t3, t2, transmit_ack_phase1
	li t2, 2
	beq t3, t2, transmit_ack_phase2
	j transmit_ack_phase3
transmit_ack_phase0:
	ack_packet 0, 0
transmit_ack_phase1:
	ack_packet 1, 0
transmit_ack_phase2:
	ack_packet 2, 0
transmit_ack_phase3:
	ack_packet 2, 1
	.else
	ack_packet ACK_PHASE, ACK_RELEASE
	.endif
	.endif

	.balign 64
transmit_stall:
	tx_start ACK_PHASE
	// SYNC followed by the STALL PID (1E).
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t4,t4,t4,t4,t4,t5,t4,t5
	txsym \symbol
	.endr
	tx_end ACK_RELEASE
	ret

	.if (!GPIO_TX) && (!TEST_EXACT) && (!TEST_PATTERN) && (!TEST_SCHEDULED)
	.org 0xa40
	.endif
transmit_data1_18:
	tx_start
	// Complete 18-byte device descriptor, EP0 MPS 64, VID cafe, PID 00c6,
	// CRC16 df97. Generated and independently decoded by check_fs_vectors.go.
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t4,t4,t5,t4,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t5,t4,t5,t5,t5,t5,t5,t5,t5,t4,t4,t5,t5,t4,t4,t5,t4,t4,t4,t5,t5,t5,t4,t5,t4,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t4,t4,t4,t5,t5,t4,t5,t5,t5,t5,t5,t5,t5,t4,t5,t5,t5
	txsym \symbol
	.endr
	tx_end
	ret

transmit_data1_18_fixed:
	tx_start
	// Complete USB 2.0 device descriptor, EP0 MPS 64, VID cafe, PID 00c6.
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t4,t4,t5,t4,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t5,t4,t5,t5,t5,t5,t5,t5,t5,t4,t4,t5,t5,t4,t4,t5,t4,t4,t4,t5,t5,t5,t4,t5,t4,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t4,t4,t4,t5,t5,t4,t5,t5,t5,t5,t5,t5,t5,t4,t5,t5,t5
	txsym \symbol
	.endr
	tx_end
	ret

	.balign 64
transmit_data1_18_split:
	tx_start FULL_PHASE
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t4,t4,t5,t4,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t4,t4,t4,t4,t4,t4,t5,t5,t4,t4,t5,t5,t4,t5,t5,t5,t4,t4,t4,t5,t4,t5,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t5,t4,t5,t5,t5,t4,t4,t4,t5,t5,t4,t5,t5,t4,t5,t4
	txsym \symbol
	.endr
	// This packet ends two core cycles earlier than the APB/PHY release phase
	// accepted by the host; compensate without perturbing any symbol cell.
	tx_end REL18
	ret

	.balign 64
transmit_data1_18_dead:
	tx_start FULL_PHASE
	// Complete USB 1.1 descriptor, VID dead, PID 00c6. This payload has no
	// stuffed cells, retaining the already-proven 160 MHz store/EOP phase.
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t4,t4,t5,t4,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t5,t4,t4,t4,t5,t5,t4,t4,t5,t5,t5,t5,t5,t4,t4,t4,t5,t5,t5,t4,t5,t4,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t5,t4,t4,t5,t5,t5,t4,t5,t5,t5,t5,t4,t4,t5
	txsym \symbol
	.endr
	tx_end REL18
	ret

	.if (!GPIO_TX) && (!TEST_EXACT) && (!TEST_PATTERN) && (!TEST_SCHEDULED)
	.org 0xe00
	.endif
	.balign 64
transmit_nak:
	tx_start
	// SYNC + NAK PID 5A: KJKJKJKK JJKKKJJK.
	txcell t5, 0
	txcell t4, 0
	txcell t5, 1
	txcell t4, 0
	txcell t5, 0
	txcell t4, 1
	txcell t5, 0
	txcell t5, 0
	txcell t4, 1
	txcell t4, 0
	txcell t5, 0
	txcell t5, 1
	txcell t5, 0
	txcell t4, 0
	txcell t4, 1
	txcell t5, 0
	tx_end
	ret

	.if (!GPIO_TX) && (!TEST_EXACT) && (!TEST_PATTERN) && (!TEST_SCHEDULED)
	.org 0xf00
	.endif
	.balign 64
transmit_data1_8:
	tx_start
	// DATA1 + first eight bytes of a custom USB 1.1 device descriptor. EP0
	// advertises a 64-byte maximum; CRC16 is 4111.
	// lowspeed.EncodeData generated these NRZI symbols; physical J/K polarity
	// is selected here for full speed.  The 75/75/100 ns write schedule has an
	// exact 83.333 ns mean and was accepted for the shorter ACK above.
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t4,t4,t5,t4,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t5,t4,t5,t4,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t4,t5
	txsym \symbol
	.endr
	tx_end REL8
	ret

	.if (!GPIO_TX) && (!TEST_EXACT) && (!TEST_PATTERN) && (!TEST_SCHEDULED)
	.org 0x1200
	.endif
	.balign 64
transmit_data1_zero:
	tx_start
	// DATA1 with an empty payload and CRC16 0000.
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5
	txsym \symbol
	.endr
	tx_end
	ret

	.if (!GPIO_TX) && (!TEST_EXACT) && (!TEST_PATTERN) && (!TEST_SCHEDULED)
	.org 0x1300
	.endif
	.balign 64
transmit_data0_8:
	tx_start
	// Device descriptor bytes 8..15: test VID cafe, PID 00c6, no strings.
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t5,t4,t5,t5,t5,t4,t4,t4,t4,t4,t4,t4,t5,t5,t4,t4,t5,t5,t4,t5,t5,t5,t4,t4,t4,t5,t4,t5,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t4,t5,t5,t5,t4,t5,t4,t5,t5,t5,t4,t4,t5,t4,t4
	txsym \symbol
	.endr
	tx_end
	ret

	.if (!GPIO_TX) && (!TEST_EXACT) && (!TEST_PATTERN) && (!TEST_SCHEDULED)
	.org 0x1540
	.endif
	.balign 64
transmit_data1_2:
	tx_start FINAL2_PHASE
	.if TEST_ONE
	// Diagnostic one-byte payload 55, CRC16 8080.
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t5,t4,t4,t5,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t5
	txsym \symbol
	.endr
	.elseif TEST_UNSTUFFED_TWO
	// Diagnostic payload 55 aa, CRC16 6041: exactly 48 differential bits,
	// with no stuffed bit.
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t5,t4,t4,t5,t5,t4,t4,t5,t4,t4,t5,t5,t4,t4,t5,t5,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t4,t4,t5
	txsym \symbol
	.endr
	.else
	// Final device descriptor bytes: no string, one configuration.
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t4,t4,t4,t4,t4,t4,t5,t4,t5,t5,t5,t5,t5,t4,t5,t4,t4
	txsym \symbol
	.endr
	.endif
	tx_end
	ret

	.balign 64
transmit_config_data1_8:
	tx_start
	// Minimal configuration bytes 0..7 (nine bytes total, zero interfaces).
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t5,t4,t5,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t4,t4,t5,t4,t5,t4,t4,t4,t4,t5,t4,t4,t5,t5
	txsym \symbol
	.endr
	tx_end
	ret

	.balign 64
transmit_config_data0_8:
	tx_start
	// Configuration bytes 8..15, including the vendor interface.
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t5,t4,t5,t5,t5,t4,t4,t5,t4,t4,t4,t5,t4,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t5,t5,t5,t4,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t4,t4,t4,t4,t5,t5,t5,t4,t5,t5,t4
	txsym \symbol
	.endr
	tx_end
	ret

	.balign 64
transmit_config_data1_2:
	tx_start
	// Final two zero bytes of the interface descriptor.
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t4,t4,t4,t4,t4,t5,t5,t5,t5,t5,t5,t4,t5,t5,t4
	txsym \symbol
	.endr
	tx_end
	ret

	.balign 64
transmit_config_data0_1:
	tx_start
	// Byte 8 of the configuration descriptor (bMaxPower = 100 mA).
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t5,t4,t5,t5,t5,t4,t4,t5,t4,t4,t4,t5,t4,t4,t5,t4,t5,t4,t5,t5,t5,t4,t4,t5,t5,t4,t4,t4,t5
	txsym \symbol
	.endr
	tx_end
	ret

	.balign 64
transmit_qualifier_data1_8:
	tx_start
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t4,t4,t5,t5,t4,t5,t4,t5,t4,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t5,t4,t5,t4,t5,t5,t4,t5,t4,t5,t5,t4,t5,t5,t4,t4,t4,t5,t4,t5,t5
	txsym \symbol
	.endr
	tx_end
	ret

	.balign 64
transmit_qualifier_data0_2:
	tx_start
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t5,t4,t5,t5,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t4,t4,t4,t4,t4,t5,t5,t5,t5,t5,t5,t5,t4,t4,t5,t5,t5
	txsym \symbol
	.endr
	tx_end
	ret

	.balign 64
transmit_qualifier_full:
	tx_start
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t4,t4,t5,t5,t4,t5,t4,t5,t4,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t4,t4,t5,t4,t4,t5,t5,t4,t5,t5,t5,t5,t5,t5,t5,t4
	txsym \symbol
	.endr
	tx_end RELQUAL
	ret

	.balign 64
transmit_config_full:
	tx_start
	.irp symbol,t5,t4,t5,t4,t5,t4,t5,t5,t5,t5,t4,t4,t5,t4,t4,t5,t5,t4,t5,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t5,t4,t4,t5,t5,t4,t5,t5,t5,t4,t5,t4,t4,t4,t5,t4,t4,t4,t5,t5,t5,t5,t5,t4,t4,t4,t5
	txsym \symbol
	.endr
	tx_end RELCONFIG
	ret
