/* Frontend regression fixture, exercised by TestFrontendCObjectMainLinkCommand.
 * Compile as a relocatable object, link with Renvo, and invoke with argument x.
 * Object main must use the exported C ABI rather than hosted appMain binding.
 */
extern int answer(int);
int main(int argc, char **argv) {
    return argc == 2 && argv[1][0] == 'x' && answer(argc) == 42 ? 0 : 1;
}
