# Incremental object-backend bring-up

The `bringup` package makes the boundary between a trusted C/assembly shell and
a candidate RTG backend explicit. `StandardStages` assigns stable milestones
and exported omnibus roots from a single C-ABI constant-return function through
a complete backend-owned image.

Each target records its C compiler, linker, object format, ABI, trusted shell,
and optional runner. `ValidateELFObject` rejects a candidate before linking when
its ELF class, byte order, machine, relocatable-object type, section alignment,
exports, undefined symbols, relocation-section links, or relocation kinds do
not satisfy the current milestone contract. Trusted shell imports and supported
relocations must be allowlisted; hosted symbols are never accepted implicitly.

`RunPipeline` requires one canonical linked-unit path and lowercase SHA-256.
Both build commands must declare that exact input and digest and consume the
path directly. The pipeline hashes the file again before either emitter runs,
so a stale, substituted, or separately lowered reference input fails at plan
validation rather than producing a misleading target comparison.

The pipeline executes the reference and candidate build commands without a
shell, validates both milestone objects before either link runs, links and runs
the two artifacts, decodes their debugger memory dumps through the shared
result ABI, and requires both the independently supplied profile/signature and
the completed-probe count to agree. Errors retain the reference/candidate side
and the first failed build, object, link, run, decode, validation, or comparison
step. A standalone-image milestone may omit relocatable objects; earlier
milestones must supply them.

The C and candidate emitters from #19 provide the concrete build commands and
the target configuration supplies linker, runner/debug-reader, object, image,
and memory-dump paths. This keeps orchestration independent of a particular
toolchain while making the trusted boundary explicit.
