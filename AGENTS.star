# Trusted development capabilities for the Renvo repository.
#
# Review this file before loading it. Commands are deliberately argument-vector
# based (rather than shell snippets) so callers cannot accidentally invoke a
# shell. Only focused test helpers are exported; repository-wide suites are
# intentionally absent because they are too expensive for routine agent turns.

def status():
    """Show the Renvo working tree status."""
    return sh.run(["git", "status", "--short", "--branch"])

def diff():
    """Show unstaged changes."""
    return sh.run(["git", "diff"])

def diff_staged():
    """Show staged changes."""
    return sh.run(["git", "diff", "--cached"])

def log():
    """Show the compact commit graph."""
    return sh.run(["git", "log", "--oneline", "--decorate", "--graph", "--all", "-30"])

def show(ref="HEAD"):
    """Show one commit or object."""
    return sh.run(["git", "show", "--stat", "--oneline", ref])

def add(paths):
    """Stage the specified repository-relative paths."""
    return sh.run(["git", "add"] + paths)

def commit(message):
    """Commit staged changes with the supplied message."""
    return sh.run(["git", "commit", "-m", message])

def gofmt(paths):
    """Format specified Go files or directories with gofmt."""
    return sh.run(["gofmt", "-w"] + paths)

def go_test(packages, run, timeout="2m"):
    """Run named Go tests in explicit packages; both selections are required."""
    command = ["go", "test", "-count=1", "-timeout", timeout]
    return sh.run(command + ["-run", run] + packages, timeout_ms=150000)

def go_vet(packages):
    """Run go vet on explicitly selected packages."""
    return sh.run(["go", "vet"] + packages)

def generate(packages):
    """Run go generate on explicitly selected packages."""
    return sh.run(["go", "generate"] + packages)

def build_bootstrap():
    """Build a Go-hosted backend and bundled bootstrap under sandbox/bin."""
    first = sh.run(["mkdir", "-p", "sandbox/bin"])
    if not first.ok:
        return first
    second = sh.run(["go", "build", "-o", "sandbox/bin/renvo-backend", "./backend"])
    if not second.ok:
        return second
    return sh.run(["go", "build", "-tags", "renvo_bundle", "-o", "sandbox/bin/renvo-bootstrap", "./cmd/renvobootstrap"])

def build_standalone(target="linux/amd64"):
    """Build a stripped, bundled, self-hosted compiler using the bootstrap."""
    return sh.run(["sandbox/bin/renvo-bootstrap", "-tags", "renvo_bundle", "-t", target, "-s", "-o", "sandbox/bin/renvo-standalone", "./cmd/renvo"], timeout_ms=300000)

def compiler_help(compiler="sandbox/bin/renvo-bootstrap"):
    """Show a compiler's current options and supported targets."""
    return sh.run([compiler, "--help"])

def compile(compiler, inputs, output, target, tags=[], strip=True):
    """Compile explicit package paths or Go files with a selected compiler."""
    command = [compiler, "-t", target, "-o", output]
    if strip:
        command = command + ["-s"]
    if len(tags) > 0:
        command = command + ["-tags", ",".join(tags)]
    return sh.run(command + inputs, timeout_ms=300000)

def compile_with_bootstrap(inputs, output, target="linux/amd64", tags=[], strip=True):
    """Compile packages or Go files with the Go-hosted development bootstrap."""
    return compile("sandbox/bin/renvo-bootstrap", inputs, output, target, tags, strip)

def compile_with_standalone(inputs, output, target="linux/amd64", tags=[], strip=True):
    """Compile packages or Go files with the self-hosted standalone compiler."""
    return compile("sandbox/bin/renvo-standalone", inputs, output, target, tags, strip)

def run_script(compiler, script, args=[], tags=[], strip=True):
    """Compile and run one Go script for the compiler's native host target."""
    command = [compiler, "run"]
    if strip:
        command = command + ["-s"]
    if len(tags) > 0:
        command = command + ["-tags", ",".join(tags)]
    command = command + [script]
    if len(args) > 0:
        command = command + ["--"] + args
    return sh.run(command, timeout_ms=300000)

def run_with_bootstrap(script, args=[], tags=[], strip=True):
    """Compile and run a Go script with the development bootstrap."""
    return run_script("sandbox/bin/renvo-bootstrap", script, args, tags, strip)

def run_with_standalone(script, args=[], tags=[], strip=True):
    """Compile and run a Go script with the self-hosted compiler."""
    return run_script("sandbox/bin/renvo-standalone", script, args, tags, strip)

def run_binary(executable, args=[]):
    """Run a previously compiled native executable with explicit arguments."""
    path = executable if "/" in executable else "./" + executable
    return sh.run([path] + args, timeout_ms=120000)

def compile_and_run_with_bootstrap(inputs, output, target, args=[], tags=[], strip=True):
    """Compile a native program with the bootstrap and run it if compilation succeeds."""
    built = compile_with_bootstrap(inputs, output, target, tags, strip)
    if not built.ok:
        return built
    return run_binary(output, args)


def compiler_bug(summary, details=""):
    """Append a compiler bug report to COMPILER_BUGS.md."""
    summary = summary.strip()
    details = details.strip()
    if not summary:
        fail("compiler bug summary must not be empty")
    path = "COMPILER_BUGS.md"
    content = fs.read(path) if fs.exists(path) else "# Compiler bugs\n\n"
    entry = "## " + summary + "\n\n"
    if details:
        entry += details + "\n\n"
    fs.write(path, content + entry)
