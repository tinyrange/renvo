#include <ctype.h>
int isdigit(int ch) { return ch >= '0' && ch <= '9'; }
int islower(int ch) { return ch >= 'a' && ch <= 'z'; }
int isupper(int ch) { return ch >= 'A' && ch <= 'Z'; }
int isalpha(int ch) { return islower(ch) || isupper(ch); }
int isalnum(int ch) { return isalpha(ch) || isdigit(ch); }
int isblank(int ch) { return ch == ' ' || ch == '\t'; }
int iscntrl(int ch) { return (ch >= 0 && ch < 32) || ch == 127; }
int isgraph(int ch) { return ch > 32 && ch < 127; }
int isprint(int ch) { return ch >= 32 && ch < 127; }
int ispunct(int ch) { return isgraph(ch) && !isalnum(ch); }
int isspace(int ch) { return ch == ' ' || (ch >= '\t' && ch <= '\r'); }
int isxdigit(int ch) { return isdigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F'); }
int tolower(int ch) { return isupper(ch) ? ch + ('a' - 'A') : ch; }
int toupper(int ch) { return islower(ch) ? ch - ('a' - 'A') : ch; }
