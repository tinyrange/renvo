static void move_bytes(char *out, char *in, int count) {
 if (out < in) {
  for (int i = 0; i < count; i++) out[i] = in[i];
 } else {
  for (int i = count; i > 0; i--) out[i-1] = in[i-1];
 }
}

int main(void) {
 char bytes[6] = "abcde";
 char *first = &bytes[0];
 char *last = &bytes[4];
 if (!(first < last && last > first && first <= first && last >= first)) return 1;
 move_bytes(bytes + 1, bytes, 4);
 if (bytes[0] != 'a' || bytes[1] != 'a' || bytes[4] != 'd') return 1;
 print("PASS\n");
 return 0;
}
