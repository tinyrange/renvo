#pragma go "host.go"
#include "cpu.c"
#include "pdp11.h"
#include <stdio.h>
extern int host_open(int argc, char **argv);
extern int host_tick(int input_ready);
extern void host_close(void);

int main(int argc, char **argv) {
    int quantum = 0, clock = 0, input;
    if (host_open(argc, argv))
        return 1;
    pdp_reset();
    if (pdp_boot()) {
        puts("Unable to read RK0 boot sector");
        host_close();
        return 1;
    }
    while (!pdp_halted) {
        pdp_step();
        if (++clock == 16667) {
            clock = 0;
            pdp_clock();
        }
        if (++quantum == 4096) {
            quantum = 0;
            input = host_tick(pdp_rx_ready());
            if (input < 0)
                break;
            if (input && pdp_rx_ready())
                pdp_receive(input - 1);
        }
    }
    host_close();
    printf("\nPDP stop PC=%06o PSW=%06o fault=%o R0=%06o R1=%06o R2=%06o R3=%06o R4=%06o R5=%06o "
           "SP=%06o\n",
           pdp_r[7], pdp_psw, pdp_fault, pdp_r[0], pdp_r[1], pdp_r[2], pdp_r[3], pdp_r[4], pdp_r[5],
           pdp_r[6]);
    return pdp_halted ? 1 : 0;
}
