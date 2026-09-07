/* PDP-11/45 integer CPU and KT11 MMU.
 * Architectural reference: Paul Nankervis, pdp11-js, revision
 * 605cc23eada3d831be54e537fe94307f1b80a85b (pdp11.js and iopage.js).
 * Paul permits use with acknowledgement of his name in modified sources.
 * This C implementation retains that acknowledgement; the host is independent.
 */
#include "pdp11.h"
#include <string.h>

#define N 8
#define Z 4
#define V 2
#define C 1
#define DATA_SPACE 0x10000u
#define REGISTER_SPACE 0x20000u
#ifdef PDP_EXTERNAL_RAM
unsigned char *pdp_ram;
#else
unsigned char pdp_ram[PDP_RAM_SIZE];
#endif
uint16_t pdp_r[8], pdp_psw;
int pdp_halted, pdp_fault;
static uint16_t alternate[6], stacks[4], par[64], pdr[64];
static uint16_t mmr0, mmr1, mmr2, mmr3, pir, stack_limit;
static uint16_t rxcs, rxbuf, txcs, clockcs;
static uint16_t rkds, rker, rkcs, rkwc, rkba, rkda;
static int rxirq, txirq, clockirq, diskirq, waiting, trapping;
static uint16_t instruction_pc;
/* One-entry instruction translation cache, not an instruction-value cache:
 * self-modifying code still reads RAM on every fetch. Only complete, readable
 * pages entirely inside RAM qualify. Odd addresses and exceptional mappings
 * retain the normal MMU/bus-fault path. Register writes invalidate the mapping.
 */
static int fetch_page_valid, fetch_page_tag;
static uint32_t fetch_page_base;

static int mode(void) { return (pdp_psw >> 14) & 3; }
static void fault(int vector) {
    if (!pdp_fault)
        pdp_fault = vector;
}
static void psw_write(uint16_t value) {
    int i, oldmode = mode(), newmode = (value >> 14) & 3;
    if ((pdp_psw ^ value) & 04000) {
        for (i = 0; i < 6; i++) {
            uint16_t t = pdp_r[i];
            pdp_r[i] = alternate[i];
            alternate[i] = t;
        }
    }
    if (oldmode != newmode) {
        stacks[oldmode] = pdp_r[6];
        pdp_r[6] = stacks[newmode];
    }
    pdp_psw = value;
}
static void reg_add(int reg, int delta) {
    pdp_r[reg] = (pdp_r[reg] + delta) & 0xffff;
    if (!(mmr0 & 0160000) && reg != 7)
        mmr1 = (mmr1 << 8) | ((delta & 31) << 3) | reg;
}
static uint32_t translate(uint16_t address, int data, int write, int usermode) {
    uint32_t physical;
    int page, access, error = 0, block;
    uint16_t descriptor;
    if (!(mmr0 & 1))
        return address >= 0160000 ? 017760000u | address : address;
    if (!(mmr3 & (usermode == 0 ? 4 : usermode == 1 ? 2 : 1)))
        data = 0;
    page = usermode * 16 + data * 8 + (address >> 13);
    descriptor = pdr[page];
    access = descriptor & 7;
    if (access == 0 || access == 3 || access == 7)
        error |= 0100000;
    if (write && (access == 1 || access == 2))
        error |= 020000;
    block = (address >> 6) & 127;
    if ((descriptor&8) ? block<((descriptor>>8)&127) : block>((descriptor>>8)&127))
        error |= 040000;
    if (error) {
        if (!(mmr0 & 0160000))
            mmr0 |= error | (page << 1);
        fault(0250);
        return 0;
    }
    pdr[page] |= write ? 0300 : 0200;
    physical = ((uint32_t)par[page] * 64 + (address & 8191)) & 0777777;
    if (physical >= 0760000)
        physical |= 017000000;
    return physical;
}
static int mmu_index(uint16_t a) {
    if (a >= 0172200 && a < 0172300)
        return 16 + ((a - 0172200) & 037) / 2 + ((a & 040) ? 64 : 0);
    if (a >= 0172300 && a < 0172400)
        return ((a - 0172300) & 037) / 2 + ((a & 040) ? 64 : 0);
    if (a >= 0177600 && a < 0177700)
        return 48 + ((a - 0177600) & 037) / 2 + ((a & 040) ? 64 : 0);
    return -1;
}
static uint16_t io_read(uint16_t a) {
    int index = mmu_index(a);
    if (index >= 0)
        return index >= 64 ? par[index - 64] : pdr[index];
    switch (a) {
    case 0177560:
        return rxcs;
    case 0177562:
        rxcs &= ~0200;
        rxirq = 0;
        return rxbuf;
    case 0177564:
        return txcs;
    case 0177566:
        return 0;
    case 0177546:
        return clockcs;
    case 0177570:
        return 0;
    case 0177572:
        return mmr0;
    case 0177574:
        return mmr1;
    case 0177576:
        return mmr2;
    case 0172516:
        return mmr3;
    case 0177760:
        return PDP_RAM_SIZE / 64 - 1;
    case 0177762:
    case 0177764:
    case 0177766:
    case 0177770:
        return 0;
    case 0177772:
        return pir;
    case 0177774:
        return stack_limit;
    case 0177776:
        return pdp_psw;
    case 0177400:
        return (rkds & 017777) | (rkda & 0160000);
    case 0177402:
        return rker;
    case 0177404:
        return rkcs;
    case 0177406:
        return rkwc;
    case 0177410:
        return rkba;
    case 0177412:
        return rkda;
    case 0177414:
    case 0177416:
        return 0;
    default:
        fault(4);
        return 0;
    }
}
static void disk_go(void) {
    unsigned char buffer[512];
    uint32_t sector = ((rkda >> 5) & 0377) * 24 + ((rkda >> 4) & 1) * 12 + (rkda & 15);
    uint32_t address = rkba | ((rkcs & 060) << 12);
    int command = (rkcs >> 1) & 7, unit = rkda >> 13, count, i;
    rkcs &= ~0201;
    if (command == 0) {
        rker = 0;
        rkcs = 0200;
        rkds = 04700;
    } else if (command == 1 || command == 2 || command == 3 || command == 5) {
        if ((rkda & 15) >= 12 || ((rkda >> 5) & 0377) >= 203)
            rker |= 040;
        while (rkwc && !rker) {
            count = (65536 - (int)rkwc) * 2;
            if (count > 512)
                count = 512;
            if (address + count > PDP_RAM_SIZE) {
                rker |= 02000;
                break;
            }
            /* Preserve the remainder of partial sectors on guest writes. */
            if (host_disk(unit, 0, sector * 512, buffer, 512)) {
                rker |= 0200;
                break;
            }
            for (i = 0; i < count; i++) {
                if (command == 1)
                    buffer[i] = pdp_ram[address + i];
                else if (command == 2)
                    pdp_ram[address + i] = buffer[i];
                else if (command == 3 && pdp_ram[address + i] != buffer[i])
                    rker |= 1;
            }
            if (command == 1 && host_disk(unit, 1, sector * 512, buffer, 512)) {
                rker |= 020000;
                break;
            }
            address += count;
            rkwc = (rkwc + count / 2) & 0xffff;
            sector++;
        }
        rkba = address & 0xffff;
        rkcs = (rkcs & ~060) | ((address >> 12) & 060);
        rkda = (unit << 13) | ((sector / 24) << 5) | (((sector / 12) & 1) << 4) | (sector % 12);
    } else if (command == 4 || command == 6) {
        rkcs |= 020000;
    }
    if (rker)
        rkcs |= 0140000;
    rkcs |= 0200;
    if (rkcs & 0100)
        diskirq = 1;
}
static void io_write(uint16_t a, uint16_t value) {
    int index = mmu_index(a);
    if (index >= 0) {
        fetch_page_valid = 0;
        if (index >= 64) {
            par[index - 64] = value;
            pdr[index - 64] &= ~0300;
        } else
            pdr[index] = value & 077417;
        return;
    }
    switch (a) {
    case 0177560:
        rxcs = (rxcs & 0200) | (value & 0100);
        rxirq = (rxcs & 0300) == 0300;
        break;
    case 0177562:
        break;
    case 0177564:
        txcs = 0200 | (value & 0100);
        txirq = (value & 0100) != 0;
        break;
    case 0177566:
        host_tx(value & 127);
        txcs |= 0200;
        if (txcs & 0100)
            txirq = 1;
        break;
    case 0177546:
        clockcs = value & 0300;
        if (!(value & 0100))
            clockirq = 0;
        break;
    case 0177570:
        break;
    case 0177572:
        fetch_page_valid = 0;
        mmr0 = value;
        break;
    case 0177574:
        break;
    case 0177576:
        mmr2 = value;
        break;
    case 0172516:
        fetch_page_valid = 0;
        mmr3 = value & 7;
        break;
    case 0177772:
        pir = value;
        break;
    case 0177774:
        stack_limit = value & 0177400;
        break;
    case 0177776:
        psw_write(value);
        break;
    case 0177404:
        rkcs = (rkcs & 0200) | (value & 0177577);
        if (value & 1)
            disk_go();
        else if ((rkcs & 0300) == 0300)
            diskirq = 1;
        break;
    case 0177406:
        rkwc = value;
        break;
    case 0177410:
        rkba = value & 0177776;
        break;
    case 0177412:
        rkda = value;
        break;
    case 0177400:
    case 0177402:
    case 0177414:
    case 0177416:
        break;
    default:
        fault(4);
        break;
    }
}
static uint16_t physical_read(uint32_t a, int byte) {
    if (!byte && (a & 1)) {
        fault(4);
        return 0;
    }
    if (a >= 017760000) {
        uint16_t v = io_read(a & 0177776);
        return byte ? (v >> ((a & 1) * 8)) & 255 : v;
    }
    if (a >= PDP_RAM_SIZE) {
        fault(4);
        return 0;
    }
    return byte ? pdp_ram[a] : pdp_ram[a] | (pdp_ram[a + 1] << 8);
}
static void physical_write(uint32_t a, uint16_t value, int byte) {
    uint16_t old;
    if (pdp_fault)
        return;
    if (!byte && (a & 1)) {
        fault(4);
        return;
    }
    if (a >= 017760000) {
        if (byte) {
            old = io_read(a & 0177776);
            if (a & 1)
                value = (old & 255) | (value << 8);
            else
                value = (old & 0177400) | (value & 255);
        }
        if (!pdp_fault)
            io_write(a & 0177776, value);
        return;
    }
    if (a >= PDP_RAM_SIZE) {
        fault(4);
        return;
    }
    pdp_ram[a] = value;
    if (!byte)
        pdp_ram[a + 1] = value >> 8;
}
static uint16_t read_virtual(uint16_t a, int data, int byte, int m) {
    uint32_t p = translate(a, data, 0, m);
    if (pdp_fault)
        return 0;
    return physical_read(p, byte);
}
static void write_virtual(uint16_t a, uint16_t value, int data, int byte, int m) {
    uint32_t p = translate(a, data, 1, m);
    if (!pdp_fault)
        physical_write(p, value, byte);
}
static uint16_t fetch(void) {
    uint16_t address = pdp_r[7], v, descriptor;
    int current_mode = mode(), tag = current_mode * 8 + (address >> 13), page;
    uint32_t physical;
    if (pdp_fault)
        return 0;
    if (fetch_page_valid && fetch_page_tag == tag && !(address & 1)) {
        physical = fetch_page_base + (address & 8191);
        v = pdp_ram[physical] | (pdp_ram[physical + 1] << 8);
        pdp_r[7] = address + 2;
        return v;
    }
    v = read_virtual(address, 0, 0, current_mode);
    if (!pdp_fault) {
        pdp_r[7] = address + 2;
        fetch_page_valid = 0;
        if (!(mmr0 & 1)) {
            physical = address & 0160000;
            if (address < 0160000) {
                fetch_page_base = physical;
                fetch_page_valid = 1;
            }
        } else {
            page = current_mode * 16 + (address >> 13);
            descriptor = pdr[page];
            physical = ((uint32_t)par[page] * 64) & 0777777;
            /* The successful slow read already checked access permission and
             * marked PDR accessed. Require an upward full-length RAM page. */
            if ((descriptor & 077410) == 077400 && physical + 8192 <= PDP_RAM_SIZE) {
                fetch_page_base = physical;
                fetch_page_valid = 1;
            }
        }
        fetch_page_tag = tag;
    }
    return v;
}
static void push(uint16_t value) {
    reg_add(6, -2);
    write_virtual(pdp_r[6], value, 1, 0, mode());
}
static uint16_t pop(void) {
    uint16_t v = read_virtual(pdp_r[6], 1, 0, mode());
    if (!pdp_fault)
        reg_add(6, 2);
    return v;
}
static void take_trap(int vector) {
    uint16_t oldpc = pdp_r[7], oldpsw = pdp_psw, newpc, newpsw;
    if (trapping) {
        pdp_halted = 1;
        return;
    }
    trapping = 1;
    pdp_fault = 0;
    waiting = 0;
    /* Trap vectors reside in kernel D space, even when kernel I/D is split. */
    newpc = read_virtual(vector, 1, 0, 0);
    newpsw = read_virtual(vector + 2, 1, 0, 0);
    newpsw = (newpsw & ~030000) | ((oldpsw >> 2) & 030000);
    if (!pdp_fault) {
        psw_write(newpsw);
        push(oldpsw);
        push(oldpc);
        pdp_r[7] = newpc;
    }
    if (pdp_fault)
        pdp_halted = 1;
    trapping = 0;
}
/* Register operands use bit 17. Memory operands use bit 16 for D space. */
static uint32_t operand(int spec, int byte) {
    int reg = spec & 7, kind = (spec >> 3) & 7, data = reg == 7 ? 0 : DATA_SPACE,
        delta = (byte && reg < 6) ? 1 : 2;
    uint16_t a, displacement;
    if (kind == 0)
        return REGISTER_SPACE | reg;
    if (kind == 1)
        return data | pdp_r[reg];
    if (kind == 2) {
        a = pdp_r[reg];
        reg_add(reg, delta);
        return data | a;
    }
    if (kind == 3) {
        a = read_virtual(pdp_r[reg], data != 0, 0, mode());
        if (!pdp_fault)
            reg_add(reg, 2);
        return DATA_SPACE | a;
    }
    if (kind == 4) {
        reg_add(reg, -delta);
        return data | pdp_r[reg];
    }
    if (kind == 5) {
        reg_add(reg, -2);
        a = read_virtual(pdp_r[reg], data != 0, 0, mode());
        return DATA_SPACE | a;
    }
    displacement = fetch();
    a = pdp_r[reg] + displacement;
    if (kind == 7)
        a = read_virtual(a, 1, 0, mode());
    return DATA_SPACE | a;
}
static uint16_t operand_read(uint32_t a, int byte, int m) {
    if (a & REGISTER_SPACE)
        return pdp_r[a & 7] & (byte ? 255 : 65535);
    return read_virtual(a & 65535, (a & DATA_SPACE) != 0, byte, m);
}
static void operand_write(uint32_t a, uint16_t value, int byte, int signextend, int m) {
    if (pdp_fault)
        return;
    if (a & REGISTER_SPACE) {
        if (byte) {
            if (signextend)
                value = (int16_t)(int8_t)value;
            else
                value = (pdp_r[a & 7] & 0177400) | (value & 255);
        }
        pdp_r[a & 7] = value;
    } else
        write_virtual(a & 65535, value, (a & DATA_SPACE) != 0, byte, m);
}
static void nz(uint32_t value, int byte) {
    int mask = byte ? 255 : 65535, sign = byte ? 128 : 32768;
    pdp_psw &= ~(N | Z);
    if (!(value & mask))
        pdp_psw |= Z;
    if (value & sign)
        pdp_psw |= N;
}
static void setflag(int flag, int yes) {
    if (yes)
        pdp_psw |= flag;
    else
        pdp_psw &= ~flag;
}
static void reset_devices(void) {
    mmr0 = mmr3 = 0;
    rxcs = 0;
    txcs = 0200;
    clockcs = 0200;
    rxirq = txirq = clockirq = diskirq = 0;
    rkds = 04700;
    rkcs = 0200;
    rker = rkwc = rkba = rkda = 0;
}
void pdp_reset(void) {
    fetch_page_valid = 0;
    memset(pdp_ram, 0, PDP_RAM_SIZE);
    memset(pdp_r, 0, sizeof(pdp_r));
    memset(alternate, 0, sizeof(alternate));
    memset(stacks, 0, sizeof(stacks));
    memset(par, 0, sizeof(par));
    memset(pdr, 0, sizeof(pdr));
    pdp_psw = 0;
    pdp_fault = pdp_halted = waiting = trapping = 0;
    mmr1 = mmr2 = pir = stack_limit = 0;
    reset_devices();
}
void pdp_clock(void) {
    clockcs |= 0200;
    if (clockcs & 0100)
        clockirq = 1;
}
int pdp_rx_ready(void) { return !(rxcs & 0200); }
void pdp_receive(int ch) {
    if (pdp_rx_ready()) {
        rxbuf = ch & 127;
        rxcs |= 0200;
        if (rxcs & 0100)
            rxirq = 1;
    }
}
int pdp_boot(void) {
    if (host_disk(0, 0, 0, pdp_ram, 512))
        return -1;
    pdp_r[0] = 0;
    pdp_r[1] = 0177404;
    pdp_r[6] = 02000;
    pdp_r[7] = 0;
    return 0;
}
void pdp_step(void) {
    uint16_t ins, source, dest, result, oldpsw;
    uint32_t sa, da, wide;
    int top, op, byte, sign, mask, reg, count, carry, condition = 0, priority;
    int32_t signedwide, signedsource, signeddest;
    if (pdp_halted)
        return;
    /* Most instruction boundaries have no pending device interrupt. Keep IPL
     * extraction and the priority ladder off that common path. */
    if (clockirq | diskirq | rxirq | txirq) {
        priority = (pdp_psw >> 5) & 7;
        if (clockirq && priority < 6) {
            clockirq = 0;
            take_trap(0100);
            return;
        }
        if (diskirq && priority < 5) {
            diskirq = 0;
            take_trap(0220);
            return;
        }
        if (rxirq && priority < 4) {
            rxirq = 0;
            take_trap(060);
            return;
        }
        if (txirq && priority < 4) {
            txirq = 0;
            take_trap(064);
            return;
        }
    }
    if (waiting)
        return;
    pdp_fault = 0;
    instruction_pc = pdp_r[7];
    oldpsw = pdp_psw;
    if (!(mmr0 & 0160000)) {
        mmr1 = 0;
        mmr2 = instruction_pc;
    }
    ins = fetch();
    if (pdp_fault)
        goto done;
    top = ins >> 12;
    byte = top >= 9 && top <= 13;
    sign = byte ? 128 : 32768;
    mask = byte ? 255 : 65535;
    if ((top >= 1 && top <= 6) || (top >= 9 && top <= 14)) {
        sa = operand((ins >> 6) & 63, byte);
        source = operand_read(sa, byte, mode());
        if (pdp_fault)
            goto done;
        da = operand(ins & 63, byte);
        if (pdp_fault)
            goto done;
        op = top & 7;
        dest = op == 1 ? 0 : operand_read(da, byte, mode());
        if (pdp_fault)
            goto done;
        result = source;
        if (op == 1) {
            nz(result, byte);
            pdp_psw &= ~V;
        } else if (op == 2) {
            result = (source - dest) & mask;
            nz(result, byte);
            setflag(V, ((source ^ dest) & (source ^ result) & sign) != 0);
            setflag(C, source < dest);
        } else if (op == 3) {
            result = source & dest;
            nz(result, byte);
            pdp_psw &= ~V;
        } else if (op == 4) {
            result = dest & ~source;
            nz(result, byte);
            pdp_psw &= ~V;
        } else if (op == 5) {
            result = dest | source;
            nz(result, byte);
            pdp_psw &= ~V;
        } else if (top == 6) {
            wide = (uint32_t)source + dest;
            result = wide;
            nz(result, 0);
            setflag(V, ((~(source ^ dest)) & (dest ^ result) & 32768) != 0);
            setflag(C, wide > 65535);
        } else {
            result = dest - source;
            nz(result, 0);
            setflag(V, ((dest ^ source) & (dest ^ result) & 32768) != 0);
            setflag(C, dest < source);
        }
        if (op != 2 && op != 3)
            operand_write(da, result, byte, op == 1, mode());
        goto done;
    }
    op = ins & 0177400;
    switch (op) {
    case 0000400:
        condition = 1;
        break;
    case 0001000:
        condition = !(pdp_psw & Z);
        break;
    case 0001400:
        condition = (pdp_psw & Z) != 0;
        break;
    case 0002000:
        condition = ((pdp_psw & N) != 0) == ((pdp_psw & V) != 0);
        break;
    case 0002400:
        condition = ((pdp_psw & N) != 0) != ((pdp_psw & V) != 0);
        break;
    case 0003000:
        condition = !(pdp_psw & Z) && (((pdp_psw & N) != 0) == ((pdp_psw & V) != 0));
        break;
    case 0003400:
        condition = (pdp_psw & Z) || (((pdp_psw & N) != 0) != ((pdp_psw & V) != 0));
        break;
    case 0100000:
        condition = !(pdp_psw & N);
        break;
    case 0100400:
        condition = (pdp_psw & N) != 0;
        break;
    case 0101000:
        condition = !(pdp_psw & (C | Z));
        break;
    case 0101400:
        condition = (pdp_psw & (C | Z)) != 0;
        break;
    case 0102000:
        condition = !(pdp_psw & V);
        break;
    case 0102400:
        condition = (pdp_psw & V) != 0;
        break;
    case 0103000:
        condition = !(pdp_psw & C);
        break;
    case 0103400:
        condition = (pdp_psw & C) != 0;
        break;
    default:
        condition = -1;
        break;
    }
    if (condition >= 0) {
        if (condition)
            pdp_r[7] += (int8_t)(ins & 255) * 2;
        goto done;
    }
    reg = (ins >> 6) & 7;
    if ((ins & 0177000) == 0004000) {
        da = operand(ins & 63, 0);
        if (da & REGISTER_SPACE)
            fault(4);
        else if (!pdp_fault) {
            push(pdp_r[reg]);
            pdp_r[reg] = pdp_r[7];
            pdp_r[7] = da & 65535;
        }
        goto done;
    }
    if ((ins & 0177000) == 0077000) {
        pdp_r[reg]--;
        if (pdp_r[reg])
            pdp_r[7] -= 2 * (ins & 63);
        goto done;
    }
    if ((ins & 0177000) == 0074000) {
        da = operand(ins & 63, 0);
        dest = operand_read(da, 0, mode());
        result = dest ^ pdp_r[reg];
        nz(result, 0);
        pdp_psw &= ~V;
        operand_write(da, result, 0, 0, mode());
        goto done;
    }
    if ((ins & 0174000) == 0070000) {
        da = operand(ins & 63, 0);
        source = operand_read(da, 0, mode());
        if (pdp_fault)
            goto done;
        op = (ins >> 9) & 7;
        signedsource = (int16_t)source;
        signeddest = (int16_t)pdp_r[reg];
        if (op == 0) {
            signedwide = signeddest * signedsource;
            pdp_r[reg] = ((uint32_t)signedwide) >> 16;
            pdp_r[reg | 1] = signedwide;
            setflag(N, signedwide < 0);
            setflag(Z, signedwide == 0);
            pdp_psw &= ~V;
            setflag(C, signedwide < -32768 || signedwide > 32767);
        } else if (op == 1) {
            signedwide = (int32_t)(((uint32_t)pdp_r[reg] << 16) | pdp_r[reg | 1]);
            pdp_psw &= ~(N | Z | V | C);
            if (!signedsource)
                pdp_psw |= V | C;
            else if (signedwide == (-2147483647 - 1) && signedsource == -1)
                pdp_psw |= V;
            else {
                int32_t q = signedwide / signedsource;
                if (q < -32768 || q > 32767)
                    pdp_psw |= V;
                else {
                    pdp_r[reg] = q;
                    pdp_r[reg | 1] = signedwide % signedsource;
                    nz(pdp_r[reg], 0);
                }
            }
        } else if (op == 2 || op == 3) {
            count = source & 63;
            if (count & 32)
                count -= 64;
            wide = op == 2 ? pdp_r[reg] : ((uint32_t)pdp_r[reg] << 16) | pdp_r[reg | 1];
            mask = op == 2 ? 65535 : -1;
            sign = op == 2 ? 32768 : (-2147483647 - 1);
            carry = (pdp_psw & C) != 0;
            condition = 0;
            while (count > 0) {
                carry = (wide & (uint32_t)sign) != 0;
                wide = (wide << 1) & (uint32_t)mask;
                if (carry != ((wide & (uint32_t)sign) != 0))
                    condition = 1;
                count--;
            }
            while (count < 0) {
                carry = wide & 1;
                wide = (wide >> 1) | (wide & (uint32_t)sign);
                count++;
            }
            if (op == 2)
                pdp_r[reg] = wide;
            else {
                pdp_r[reg] = wide >> 16;
                pdp_r[reg | 1] = wide;
            }
            setflag(N, (wide & (uint32_t)sign) != 0);
            setflag(Z, wide == 0);
            setflag(V, condition);
            setflag(C, carry);
        } else
            fault(010);
        goto done;
    }
    if ((ins & 0177770) == 0000200) {
        reg = ins & 7;
        source = pdp_r[reg];
        result = pop();
        if (!pdp_fault) {
            pdp_r[7] = source;
            pdp_r[reg] = result;
        }
        goto done;
    }
    if ((ins & 0177700) == 0000100) {
        da = operand(ins & 63, 0);
        if (da & REGISTER_SPACE)
            fault(4);
        else
            pdp_r[7] = da & 65535;
        goto done;
    }
    if ((ins & 0177700) == 0000300) {
        da = operand(ins & 63, 0);
        dest = operand_read(da, 0, mode());
        result = (dest << 8) | (dest >> 8);
        operand_write(da, result, 0, 0, mode());
        nz(result, 1);
        pdp_psw &= ~(V | C);
        goto done;
    }
    if ((ins & 0177740) == 0000240) {
        if (ins & 020)
            pdp_psw |= ins & 15;
        else
            pdp_psw &= ~(ins & 15);
        goto done;
    }
    if ((ins & 0177770) == 0000230) {
        if (mode() == 0)
            pdp_psw = (pdp_psw & ~0340) | ((ins & 7) << 5);
        goto done;
    }
    if ((ins & 0177400) == 0104000) {
        fault(030);
        goto done;
    }
    if ((ins & 0177400) == 0104400) {
        fault(034);
        goto done;
    }
    if ((ins & 0177700) == 0006400) {
        pdp_r[6] = pdp_r[7] + 2 * (ins & 63);
        pdp_r[7] = pdp_r[5];
        pdp_r[5] = pop();
        goto done;
    }
    op = ins & 0177700;
    if (op == 0006500 || op == 0106500 || op == 0006600 || op == 0106600) {
        int previous = (pdp_psw >> 12) & 3, data = (ins & 0100000) != 0;
        da = operand(ins & 63, 0);
        if (!(da & REGISTER_SPACE))
            da = (da & 65535) | (data ? DATA_SPACE : 0);
        if ((ins & 07700) == 06500) {
            source = (da == (REGISTER_SPACE | 6) && previous != mode())
                         ? stacks[previous]
                         : operand_read(da, 0, previous);
            if (!pdp_fault)
                push(source);
        } else {
            source = pop();
            if (da == (REGISTER_SPACE | 6) && previous != mode())
                stacks[previous] = source;
            else
                operand_write(da, source, 0, 0, previous);
        }
        nz(source, 0);
        pdp_psw &= ~V;
        goto done;
    }
    if ((op >= 0005000 && op <= 0006300) || (op >= 0105000 && op <= 0106300) || op == 0006700 ||
        op == 0106700 || op == 0106400) {
        byte = (ins & 0100000) != 0;
        mask = byte ? 255 : 65535;
        sign = byte ? 128 : 32768;
        da = operand(ins & 63, byte);
        dest = operand_read(da, byte, mode());
        if (pdp_fault)
            goto done;
        carry = (pdp_psw & C) != 0;
        op = (ins >> 6) & 077;
        result = dest;
        switch (op) {
        case 050:
            result = 0;
            pdp_psw = (pdp_psw & ~(N | V | C)) | Z;
            break;
        case 051:
            result = (~dest) & mask;
            nz(result, byte);
            pdp_psw = (pdp_psw & ~V) | C;
            break;
        case 052:
            result = (dest + 1) & mask;
            nz(result, byte);
            setflag(V, result == sign);
            break;
        case 053:
            result = (dest - 1) & mask;
            nz(result, byte);
            setflag(V, dest == sign);
            break;
        case 054:
            result = (-dest) & mask;
            nz(result, byte);
            setflag(V, result == sign);
            setflag(C, result != 0);
            break;
        case 055:
            result = (dest + carry) & mask;
            nz(result, byte);
            setflag(V, carry && dest == sign - 1);
            setflag(C, carry && dest == mask);
            break;
        case 056:
            result = (dest - carry) & mask;
            nz(result, byte);
            setflag(V, carry && dest == sign);
            setflag(C, carry && dest == 0);
            break;
        case 057:
            nz(dest, byte);
            pdp_psw &= ~(V | C);
            goto done;
        case 060:
            result = (dest >> 1) | (carry ? sign : 0);
            nz(result, byte);
            setflag(C, dest & 1);
            setflag(V, ((pdp_psw & N) != 0) != ((pdp_psw & C) != 0));
            break;
        case 061:
            result = ((dest << 1) | carry) & mask;
            nz(result, byte);
            setflag(C, dest & sign);
            setflag(V, ((pdp_psw & N) != 0) != ((pdp_psw & C) != 0));
            break;
        case 062:
            result = (dest >> 1) | (dest & sign);
            nz(result, byte);
            setflag(C, dest & 1);
            setflag(V, ((pdp_psw & N) != 0) != ((pdp_psw & C) != 0));
            break;
        case 063:
            result = (dest << 1) & mask;
            nz(result, byte);
            setflag(C, dest & sign);
            setflag(V, ((pdp_psw & N) != 0) != ((pdp_psw & C) != 0));
            break;
        case 064:
            if (mode() == 0)
                pdp_psw = (pdp_psw & 0177400) | (dest & 0357);
            else
                pdp_psw = (pdp_psw & ~15) | (dest & 15);
            goto done;
        case 067:
            if (byte) {
                result = pdp_psw & 255;
                nz(result, 1);
                pdp_psw &= ~V;
            } else {
                result = (pdp_psw & N) ? 65535 : 0;
                setflag(Z, result == 0);
                pdp_psw &= ~V;
            }
            break;
        default:
            fault(010);
            goto done;
        }
        operand_write(da, result, byte, op == 067, mode());
        goto done;
    }
    switch (ins) {
    case 0:
        if (mode())
            fault(4);
        else
            pdp_halted = 1;
        break;
    case 1:
        waiting = 1;
        break;
    case 2:
    case 6:
        source = pop();
        dest = pop();
        if (!pdp_fault) {
            if (mode())
                dest = (dest & 037) | (pdp_psw & 0177740);
            psw_write(dest);
            pdp_r[7] = source;
        }
        oldpsw &= ~020;
        break;
    case 3:
        fault(014);
        break;
    case 4:
        fault(020);
        break;
    case 5:
        if (mode() == 0)
            reset_devices();
        break;
    default:
        fault(010);
        break;
    }
done:
    if (pdp_fault)
        take_trap(pdp_fault);
    else if (oldpsw & 020)
        take_trap(014);
}
