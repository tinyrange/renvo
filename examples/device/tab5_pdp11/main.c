/* Board glue only: the host and Tab5 execute the same PDP-11 core. */
#pragma go "board.go"
#define PDP_EXTERNAL_RAM
#include "../../pdp11/cpu.c"
extern unsigned char *host_memory(int size);
extern int host_open(void);
extern int host_tick(int input_ready);
extern int host_clock(void);
extern void host_stop(int pc, int psw, int fault);
extern void host_test(void);
extern void host_progress(int pc, int psw, int fault);

int pdp_run(void) {
#ifdef PDP_AUTOTEST
    host_test();
#endif
    if (host_open())
        return 1;
    pdp_ram = host_memory(PDP_RAM_SIZE);
    if (!pdp_ram)
        return 1;
    pdp_reset();
    if (pdp_boot()) {
        host_stop(0, 0, -1);
        return 1;
    }
    while (!pdp_halted) {
        /* Bound input/display latency without sleeping between CPU batches. */
        for (int i = 0; i < 1024 && !pdp_halted; i++)
            pdp_step();
#ifdef PDP_AUTOTEST
        host_progress(pdp_r[7], pdp_psw, pdp_fault);
#endif
        if (host_clock())
            pdp_clock();
        int input = host_tick(pdp_rx_ready());
        if (input < 0)
            return 0;
        if (input && pdp_rx_ready())
            pdp_receive(input - 1);
    }
    host_stop(pdp_r[7], pdp_psw, pdp_fault);
    return 1;
}
