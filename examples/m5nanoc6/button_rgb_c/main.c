#pragma go "board.go"

extern int board_button_pressed(void);
extern unsigned int board_random_uint32(void);
extern void board_set_rgb(unsigned char red, unsigned char green, unsigned char blue);
extern void board_delay_milliseconds(unsigned int milliseconds);

static unsigned int mix(unsigned int value) {
    value ^= value << 13;
    value ^= value >> 17;
    return value ^ (value << 5);
}

int main(void) {
    unsigned int timing = 0;
    board_set_rgb(0, 0, 0);
    for (;;) {
        while (!board_button_pressed()) {
            timing++;
        }
        board_delay_milliseconds(10);
        if (!board_button_pressed()) {
            continue;
        }

        unsigned int random = mix(board_random_uint32() ^ timing);
        board_set_rgb((unsigned char)(random >> 16),
                      (unsigned char)(random >> 8),
                      (unsigned char)random);

        while (board_button_pressed()) {
            timing++;
        }
        board_delay_milliseconds(10);
    }
}
