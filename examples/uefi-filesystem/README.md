# UEFI filesystem

This application locates the Simple File System protocol, opens the boot
volume, and creates `RENVO.TXT`. It uses UEFI file handles directly rather than
the generalized hosted filesystem API.
