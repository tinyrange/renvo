/* Run the same instruction/MMU tests on the real prepared RISC-V target. */
#pragma go "board.go"
#define PDP_EXTERNAL_RAM
#define main core_test
#include "../../../pdp11/tests/core.c"
#undef main
extern unsigned char *host_memory(int size);
int run_test(void) {
    pdp_ram = host_memory(PDP_RAM_SIZE);
    if (!pdp_ram) return 1;
    return core_test();
}
