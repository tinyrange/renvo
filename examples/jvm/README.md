# JVM classfile backend

[`jvm.rbe`](../../backends/jvm.rbe) is an external CompilerJIT backend that
writes a JVM classfile directly. It does not emit Java source and does not run
`javac`. The generated public class is named `RenvoProgram`, so write it to a
matching filename:

```sh
renvo \
  -backend backends/jvm.rbe \
  -t jvm/vm32 \
  -s \
  -o RenvoProgram.class \
  examples/jvm

java -Xverify:all -cp . RenvoProgram
```

The v1 runtime preserves Renvo's VM32 data model: words and pointers are
32-bit values, pointers are offsets into a little-endian `byte[]`, and Java
references never masquerade as Renvo pointers. The classfile is version 49
(Java 5), uses only Java 5-era runtime APIs, has no `StackMapTable`
requirement, and avoids runtime Base64 or source-compilation helpers. Java 5
is therefore the minimum supported JRE; the deliberately conservative output
is also suitable input for D8.

The RBE overlays a small `java` package. `GetProperty` and
`CallStaticString` demonstrate the explicit interop boundary; the latter calls
a public static Java method with one `java.lang.String` parameter and converts
its result with `String.valueOf`. UTF-8 values are copied across linear memory.
This deliberately narrow API is the seed for generated, descriptor-checked
Java wrappers rather than an attempt to make ordinary Renvo pointers into JVM
object references. The same build-tagged package sources live in `std/`, so a
Renvo compiler already hosted on the JVM can compile programs that import
`java` without needing the original RBE overlay.

The runtime supports ordinary computation, heap allocation, direct and
indirect calls, process arguments, environment variables, standard streams,
and the file operations needed by the Renvo compiler. It does not yet provide
threads, Java object handles, constructors, fields, instance-method wrappers,
callbacks, exports, or JAR writing.

## Android DEX and APK

The same RBE contains an `android/vm32` target that translates its Java 5
class model directly to DEX 035. It does not invoke D8, javac, Gradle, or the
Android SDK:

```sh
go run ./cmd/renvo \
  -backend backends/jvm.rbe \
  -t android/vm32 -s \
  -o classes.dex \
  path/to/program

go run ./cmd/renvoapk \
  -dex classes.dex \
  -config app.conf \
  -o app.apk
```

The DEX includes `dev.renvo.app.RenvoActivity`, a small Activity adapter that
enters `RenvoProgram.run`. `renvoapk` supplies the binary manifest, ZIP32
container, and APK Signature Scheme v2 development signature.

## Self-hosting

The generated compiler needs a larger linear-memory arena than ordinary
programs. The following builds a host-produced stage 1 and then asks two JVM
stages to rebuild themselves:

```sh
mkdir -p sandbox/jvm-selfhost/{stage1,stage2,stage3}

go run ./cmd/renvo \
  -backend backends/jvm.rbe \
  -t jvm/vm32 -s -arena-size 536870912 \
  -o sandbox/jvm-selfhost/stage1/RenvoProgram.class \
  ./cmd/renvo

java -Xmx2g -cp sandbox/jvm-selfhost/stage1 RenvoProgram \
  -t jvm/vm32 -s -arena-size 536870912 \
  -o sandbox/jvm-selfhost/stage2/RenvoProgram.class \
  ./cmd/renvo

java -Xmx2g -cp sandbox/jvm-selfhost/stage2 RenvoProgram \
  -t jvm/vm32 -s -arena-size 536870912 \
  -o sandbox/jvm-selfhost/stage3/RenvoProgram.class \
  ./cmd/renvo

cmp sandbox/jvm-selfhost/stage2/RenvoProgram.class \
    sandbox/jvm-selfhost/stage3/RenvoProgram.class
```

Stage 2 and stage 3 should be byte-for-byte identical. A 512 MiB Renvo arena
is currently required; `-Xmx2g` is a comfortable host-JVM limit rather than an
amount eagerly allocated by the generated program.
