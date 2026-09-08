/* Deterministic instruction/MMU/device tests; no guest image or host runtime. */
#include "../cpu.c"
#include <stdio.h>
static unsigned char disk[512];
static int transmitted;
static int last_unit;
int host_disk(int unit, int writing, uint32_t offset, unsigned char *buffer, int count) {
    last_unit = unit;
    if (unit > 1 || offset || count != 512)
        return 1;
    if (writing)
        memcpy(disk, buffer, 512);
    else
        memcpy(buffer, disk, 512);
    return 0;
}
void host_tx(int value) { transmitted = value; }
static void word(int address, int value) {
    pdp_ram[address] = value;
    pdp_ram[address + 1] = value >> 8;
}
static int fail(const char *text) {
    puts(text);
    return 1;
}
int main(void) {
    int i;
    pdp_reset();
    word(0, 012700);
    word(2, 0177777); /* MOV #-1,R0 */
    word(4, 010001);  /* MOV R0,R1 */
    word(6, 0105201); /* INCB R1, preserve register high byte */
    word(8, 0112702);
    word(10, 0200); /* MOVB #0200,R2, sign extend */
    word(12, 0062700);
    word(14, 1); /* ADD #1,R0 */
    for (i = 0; i < 5; i++)
        pdp_step();
    if (pdp_r[0] != 0 || pdp_r[1] != 0177400 || pdp_r[2] != 0177600 || (pdp_psw & 15) != (Z | C))
        return fail("FAIL integer/byte flags");

    pdp_reset();
    word(0, 012021);
    word(0100, 012345);
    pdp_r[0] = 0100;
    pdp_r[1] = 0200;
    pdp_step();
    if (physical_read(0200, 0) != 012345 || pdp_r[0] != 0102 || pdp_r[1] != 0202)
        return fail("FAIL deferred addressing");

    pdp_reset();
    mmr0 = 1;
    mmr3 = 4;
    par[0] = 0100;
    par[8] = 0200;
    pdr[0] = pdr[8] = 077406;
    word(0100 * 64 + 014, 01111);
    word(0200 * 64 + 014, 0700);
    word(0200 * 64 + 016, 0);
    pdp_r[7] = 0400;
    pdp_r[6] = 01000;
    take_trap(014);
    if (pdp_halted || pdp_fault || pdp_r[7] != 0700 || pdp_r[6] != 0774 ||
        read_virtual(0774, 1, 0, 0) != 0400)
        return fail("FAIL kernel D trap vector");
    pdp_fault = 0;
    translate(0, 0, 1, 3);
    if (pdp_fault != 0250 || !(mmr0 & 0100000))
        return fail("FAIL nonresident page");

    pdp_reset();
    pdp_receive('x');
    if (pdp_rx_ready() || io_read(0177562) != 'x' || !pdp_rx_ready())
        return fail("FAIL receive latch");
    io_write(0177564, 0100);
    io_write(0177566, 'y');
    if (transmitted != 'y' || !txirq)
        return fail("FAIL transmit interrupt");
    io_write(0177546, 0100);
    pdp_clock();
    if (!clockirq)
        return fail("FAIL clock interrupt");

    pdp_reset();
    for (i = 0; i < 512; i++)
        disk[i] = (unsigned char)i;
    rkba = 01000;
    rkwc = 0177400;
    rkda = 0;
    rkcs = 0105;
    disk_go();
    if (rker || rkwc || !diskirq || pdp_ram[01001] != 1 || pdp_ram[01777] != 255)
        return fail("FAIL disk DMA read");
    pdp_ram[01000] = 42;
    pdp_ram[01001] = 43;
    rkba = 01000;
    rkwc = 0177777;
    rkda = 0;
    rkcs = 3;
    disk_go();
    if (disk[0] != 42 || disk[1] != 43 || disk[2] != 2 || disk[511] != 255)
        return fail("FAIL partial disk write preservation");
    rkba = 01000;
    rkwc = 0177400;
    rkda = 020000;
    rkcs = 5;
    rker = 0;
    disk_go();
    if (rker || last_unit != 1 || pdp_ram[01000] != 42)
        return fail("FAIL drive/cylinder decoding");
    /* Translation-cache hits must preserve self-modifying code and all MMU
     * invalidation/fault semantics. Use architectural register writes here. */
    pdp_reset();
    word(0, 012345);
    if (fetch() != 012345 || !fetch_page_valid)
        return fail("FAIL fetch cache fill");
    word(0, 054321);
    pdp_r[7] = 0;
    if (fetch() != 054321)
        return fail("FAIL cached self modification");
    io_write(0177572, 1);
    io_write(0172340, 0100);
    io_write(0172300, 077402);
    word(010000, 011111);
    word(020000, 022222);
    pdp_r[7] = 0;
    if (fetch() != 011111 || !fetch_page_valid || !(pdr[0] & 0200))
        return fail("FAIL mapped fetch");
    io_write(0172340, 0200);
    pdp_r[7] = 0;
    if (fetch() != 022222)
        return fail("FAIL PAR cache invalidation");
    io_write(0172300, 0);
    pdp_r[7] = 0;
    fetch();
    if (pdp_fault != 0250 || pdp_r[7] != 0)
        return fail("FAIL PDR cache invalidation");
    pdp_reset();
    io_write(0177572, 1);
    io_write(0172340, 0100);
    io_write(0172300, 077402);
    word(010000, 011111);
    fetch();
    io_write(0172516, 4);
    io_write(0172360, 0200); /* Kernel D PAR0, distinct from instruction space. */
    io_write(0172320, 077406);
    word(020000, 022222);
    pdp_r[7] = 0;
    if (fetch() != 011111 || read_virtual(0, 1, 0, 0) != 022222)
        return fail("FAIL cached split I/D");
    io_write(0177640, 0300);
    io_write(0177600, 077402);
    word(030000, 033333);
    psw_write(0140000);
    pdp_r[7] = 0;
    if (fetch() != 033333)
        return fail("FAIL cached mode change");
    io_write(0177572, 0);
    word(0, 044444);
    pdp_r[7] = 0;
    if (fetch() != 044444)
        return fail("FAIL cached MMU disable");
    pdp_reset();
    fetch();
    pdp_r[7] = 1;
    fetch();
    if (pdp_fault != 4 || pdp_r[7] != 1)
        return fail("FAIL cached odd fetch");
    puts("PASS");
    return 0;
}
