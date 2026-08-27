#pragma go "board.go"

extern void board_set_led(int on);
extern void board_delay_milliseconds(unsigned int milliseconds);

int main(void) {
    for (;;) {
        board_set_led(1);
        board_delay_milliseconds(500);
        board_set_led(0);
        board_delay_milliseconds(500);
    }
}
