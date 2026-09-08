/* PDP-11 host boundary. No display, filesystem, or ESP32 dependencies in core. */
#ifndef RENVO_PDP11_H
#define RENVO_PDP11_H
#include <stdint.h>
#define PDP_RAM_SIZE (248 * 1024)
extern uint16_t pdp_r[8];
extern uint16_t pdp_psw;
/* Hosts with a separate memory reservation supply storage before pdp_reset().
 * The pointer must cover PDP_RAM_SIZE writable bytes for the machine lifetime. */
#ifdef PDP_EXTERNAL_RAM
extern unsigned char *pdp_ram;
#else
extern unsigned char pdp_ram[PDP_RAM_SIZE];
#endif
extern int pdp_halted;
extern int pdp_fault;
void pdp_reset(void);
void pdp_step(void);
void pdp_clock(void);
void pdp_receive(int ch);
int pdp_rx_ready(void);
int pdp_boot(void);
/* Byte offsets and byte counts; return zero on success. Host owns disk policy. */
int host_disk(int unit, int writing, uint32_t offset, unsigned char *buffer, int count);
void host_tx(int ch);
#endif
