# Managed heap audit and remaining scope

## Closed integration gaps

- Managed runtime source, platform services, startup and root-capture emitters
  now live in RTG definitions. Prepared and bundled backends share those hooks;
  policy validation uses a typed ABI contract rather than OS/ISA names.

- Built-in backend APIs now accept the same managed budget range as prepared
  backends: zero internally means uncapped, positive limits need not fit the
  legacy 256-byte..1-GiB arena range, and negative API limits are rejected.
  Tiny positive limits compile but fail through normal startup OOM handling.
- System-profile target selection is validated after resolution. An arena
  profile cannot silently select desktop allocation with stale arena policy.
- A custom definition's public name no longer overrides its allocator policy
  with the policy of a similarly named built-in target.
- Runtime build tags follow the resolved memory strategy, including custom
  target names, omitted tags, and switching back to arena.
- Native acceptance/stress coverage now runs through both prepared and built-in
  backends. It covers interior page-index roots and reuse, not only allocation
  base pointers. Managed self-hosting also exercises small and large budgets.

## Still deliberately unsupported

These require additional runtime contracts rather than relaxing validation:

1. **Concurrent mutators or switched/suspended stacks.** Collection currently
   scans one native stack. Supporting more requires registered stack ranges,
   register capture, and a stop-the-world/root-enumeration protocol.
2. **JIT and native FFI/object/linked-image output.** Foreign code and storage
   require explicit root/lifetime ownership and safe collector entry boundaries.
   The managed compiler uses its embedded backend, not a JIT stack.
3. **Other operating systems and native hardware qualification.** All 33 Linux
   QEMU targets have adapters and pass the managed acceptance matrix. Only
   Linux/amd64 has native managed compiler self-host coverage. Other platforms
   can author mapping, startup, register and stack adapters through RBE/RTG hooks
   without common-compiler edits, but still need qualification. Arena defaults
   are unchanged. See `QEMU.md`.
4. **Managed system profiles.** Existing profiles specify an arena budget, not
   a heap budget. Desktop builds currently use `-heap-limit` directly.
5. **Production latency guarantees.** The collector is synchronous, conservative
   and nonmoving. False retention and fragmentation remain possible. Passing
   correctness/self-host tests is not a performance-gate or pause-time claim.

The independently tested Go allocator core and definition-owned native runtime
remain separate implementations; the native page index and mark worklist are
not currently mirrored by the core model.
