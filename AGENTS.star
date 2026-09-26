"""Restricted Renvo workflows; AGENTS.md remains authoritative.

  git.status(short=True)
  go.build_compiler()
  repo.compile("sandbox/hello.go", "hello")
  repo.execute("hello")
  go.test("./internal/repl", run="TestSession")
  repo.corpus("backend", "tagged_memory_access")
  repo.preflight()
  publication.status()  # Task-scoped branch/commit/push/PR helpers also available.

No general process runner, shell, go.run/tool/generate, arbitrary environment,
working-directory override, background process, or additional REPL is exported.
"""

load("//stdlib/git.star", git_tools = "tools")
load("//stdlib/golang.star", std_go_build = "build_process", std_go_test = "test", std_go_version = "version")

workspace = privileged.workspace(".", readonly = False)
git = git_tools.readonly()

_BIN = "sandbox/agent-bin"
_PROGRAMS = "sandbox/agent-programs"
_SUFFIX = ".exe" if platform.startswith("windows/") else ""

def _name(value):
    if type(value) != "string" or not value or len(value) > 100:
        fail("Expected a nonempty artifact name of at most 100 characters")
    if value[0] not in "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789":
        fail("Artifact names must start with a letter or digit")
    for char in value.elems():
        if char not in "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-":
            fail("Artifact names may contain only letters, digits, '_' and '-'")
    return value

def _relative(value):
    if type(value) != "string" or not value:
        fail("Expected a workspace-relative source or package path")
    if value.startswith("./"):
        value = value[2:]
    if not value or value.startswith("-") or "\\" in value or ":" in value:
        fail("Invalid workspace-relative path")
    for part in value.split("/"):
        if part in ["", ".", "..", "..."]:
            fail("Absolute paths, traversal and package wildcards are not allowed")
    return value

def _native_target():
    targets = {
        "linux/amd64": "linux/amd64", "linux/386": "linux/386",
        "linux/arm64": "linux/aarch64", "linux/arm": "linux/arm",
        "darwin/arm64": "darwin/arm64", "windows/amd64": "windows/amd64",
        "windows/386": "windows/386", "windows/arm64": "windows/arm64",
        "freebsd/amd64": "freebsd/amd64", "openbsd/amd64": "openbsd/amd64",
        "netbsd/amd64": "netbsd/amd64",
    }
    if platform not in targets:
        fail("No native Renvo target configured for " + platform)
    return targets[platform]

def build_compiler():
    """Build the stage0 backend and bundled bootstrap at fixed sandbox paths.

    Returns build results. Stops if the backend fails; inspect success before
    compiling. Builds only these two packages, with fixed flags and root cwd.
    """
    if "agent-bin" not in workspace.list_dir("sandbox"):
        workspace.mkdir(_BIN)
    backend = std_go_build(
        package = "./backend", output = workspace.path(_BIN + "/renvo-backend" + _SUFFIX),
        cwd = workspace, timeout_ms = 180000, output_limit = 1048576,
    )
    if not backend.success:
        return [backend]
    bootstrap = std_go_build(
        package = "./cmd/renvobootstrap", tags = ["renvo_bundle"],
        output = workspace.path(_BIN + "/renvo-bootstrap" + _SUFFIX),
        cwd = workspace, timeout_ms = 180000, output_limit = 1048576,
    )
    return [backend, bootstrap]

def compile(source, name, backend = False):
    """Compile one workspace source file/package to a named native sandbox artifact.

    backend=True uses stage0 directly for backend-subset reproducers. Otherwise
    uses the bundled bootstrap. No raw flags, custom backends, or output paths.
    A failed compilation removes its output so stale binaries cannot be run.
    """
    source = _relative(source)
    name = _name(name)
    if type(backend) != "bool":
        fail("backend must be a bool")
    if "agent-programs" not in workspace.list_dir("sandbox"):
        workspace.mkdir(_PROGRAMS)
    artifact = _PROGRAMS + "/" + name + _SUFFIX
    if name + _SUFFIX in workspace.list_dir(_PROGRAMS):
        workspace.delete(artifact)
    compiler = "renvo-backend" if backend else "renvo-bootstrap"
    args = [workspace.path(_BIN + "/" + compiler + _SUFFIX)]
    if not backend:
        args.extend(["-tags", "renvo_bundle"])
    args.extend(["-t", _native_target(), "-o", workspace.path(artifact), "./" + source])
    result = privileged.run(cwd = workspace, timeout_ms = 180000, output_limit = 1048576, *args)
    if not result.success and name + _SUFFIX in workspace.list_dir(_PROGRAMS):
        workspace.delete(artifact)
    return result

def execute(name, args = (), stdin = None):
    """Run a named artifact from sandbox/agent-programs, with that directory as cwd.

    Fixed 30-second deadline and 1 MiB output cap. No executable path, cwd, env,
    shell or background options. The sandbox directory is not OS isolation:
    compiled application code executes with the host user's permissions.
    """
    name = _name(name)
    if type(args) not in ["list", "tuple"]:
        fail("args must be a list or tuple of strings")
    for arg in args:
        if type(arg) != "string":
            fail("Program arguments must be strings")
    if stdin != None and type(stdin) != "string":
        fail("stdin must be text or None")
    return privileged.run(
        cwd = workspace.path(_PROGRAMS), stdin = stdin,
        timeout_ms = 30000, output_limit = 1048576,
        *([workspace.path(_PROGRAMS + "/" + name + _SUFFIX)] + list(args))
    )

def test(package, run, bundled = False):
    """Run a selected test regex in one concrete approved Go package.

    No ./..., independent corpus packages, arbitrary Go flags, env or cwd.
    Use repo.corpus for backend/frontend positive regression programs.
    """
    if package == ".":
        selected = "."
    else:
        selected = _relative(package)
        allowed = selected in ["driver", "backend", "frontend_tests"]
        for prefix in ["internal/", "std/", "cmd/", "backend/unit", "backend/target", "backend/bringup", "backend/omnibus", "backend/cmd"]:
            if selected.startswith(prefix) and (prefix.endswith("/") or selected == prefix or selected.startswith(prefix + "/")):
                allowed = True
        if not allowed:
            fail("Package is outside the approved test roots")
        selected = "./" + selected
    if type(run) != "string" or not run:
        fail("Provide a nonempty test regex; use preflight for routine broad checks")
    if type(bundled) != "bool":
        fail("bundled must be a bool")
    return std_go_test(
        packages = [selected], run = run, count = 1, timeout = "9m",
        tags = ["renvo_bundle"] if bundled else None, cwd = workspace,
        timeout_ms = 600000, output_limit = 2097152,
    )

def preflight():
    """Run tools/check preflight with its unchanged local budget and gates."""
    return privileged.run(
        "bash", workspace.path("tools/check"), "preflight",
        cwd = workspace, timeout_ms = 120000, output_limit = 2097152,
    )

def corpus(kind, filter):
    """Run a filtered backend or frontend corpus via the canonical check driver."""
    if kind not in ["backend", "frontend"]:
        fail("Corpus kind must be backend or frontend")
    if type(filter) != "string" or not filter:
        fail("Provide a nonempty corpus filter")
    return privileged.run(
        "bash", workspace.path("tools/check"), kind, filter,
        cwd = workspace, timeout_ms = 660000, output_limit = 2097152,
    )

def version():
    """Inspect the host Go version; no command arguments are accepted."""
    return std_go_version(cwd = workspace)

go = module("go", version = version, build_compiler = build_compiler, test = test)
repo = module("repo", compile = compile, execute = execute, preflight = preflight, corpus = corpus)


# Task-scoped publication: no general Git writer or process runner is exported.
_PR_BRANCH = "staragent/driver-output-permissions"
_PR_REPOSITORY = "tinyrange/renvo"
_PR_FILE = "internal/driver/host.go"

def _publish_git(args):
    return privileged.run(
        cwd = workspace, timeout_ms = 120000, output_limit = 1048576,
        *( ["git", "--literal-pathspecs", "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false"] + args)
    )

def _publish_require(result):
    if not result.success or result.stdout_truncated or result.stderr_truncated:
        fail("Publication command failed or output was truncated: " + str(result))
    return result.stdout.strip()

def _publish_on_branch():
    branch = _publish_require(_publish_git(["branch", "--show-current"]))
    if branch != _PR_BRANCH:
        fail("Publication requires branch " + _PR_BRANCH)

def _publish_clean_index():
    staged = _publish_require(_publish_git(["diff", "--cached", "--name-only"]))
    if staged:
        fail("Refusing to modify an existing staged change: " + staged)

def _publish_gh(args):
    return privileged.run(
        cwd = workspace, timeout_ms = 120000, output_limit = 1048576,
        env = {"GH_PROMPT_DISABLED": "1", "GH_PAGER": "cat"},
        *(["gh"] + args)
    )

def publication_status():
    """Check GitHub CLI availability/authentication and existing task PRs."""
    results = []
    for args in [
        ["--version"],
        ["auth", "status", "--hostname", "github.com"],
        ["pr", "list", "--repo", _PR_REPOSITORY, "--head", _PR_BRANCH,
         "--state", "all", "--json", "number,url,state,baseRefName,headRefName"],
    ]:
        result = _publish_gh(args)
        results.append(result)
        if not result.success or result.stdout_truncated or result.stderr_truncated:
            return results
    return results

def publication_branch():
    """Create the fixed task branch at current HEAD, preserving working changes.

    Inspect HEAD versus the intended PR base first: this does not rebase or
    discard existing commits. An existing branch is an error, not overwritten.
    """
    _publish_clean_index()
    return _publish_git(["switch", "-c", _PR_BRANCH])

def publication_commit():
    """Commit only the driver permission fix; refuse a nonempty index.

    Local Git hooks are disabled for this narrow workflow. Existing signing
    configuration is retained. No amend, reset, or unrelated staging is allowed.
    """
    _publish_on_branch()
    _publish_clean_index()
    return _publish_git([
        "commit", "--only", "-m",
        "fix(driver): set explicit permissions on compiled output",
        "--", _PR_FILE,
    ])

def publication_push():
    """Push only the fixed task branch to the fixed repository; never force."""
    _publish_on_branch()
    _publish_clean_index()
    dirty = _publish_require(_publish_git(["status", "--porcelain", "--untracked-files=no"]))
    if dirty:
        fail("Commit or resolve tracked changes before publishing: " + dirty)
    return _publish_git([
        "push", "https://github.com/" + _PR_REPOSITORY + ".git",
        "refs/heads/" + _PR_BRANCH + ":refs/heads/" + _PR_BRANCH,
    ])

def publication_pr(base, title, body):
    """Open the task PR in tinyrange/renvo with an explicitly reviewed base.

    Review the full branch diff before calling. No merge or repository settings
    operations are exposed. Push must have completed successfully first.
    """
    _publish_on_branch()
    if type(base) != "string" or not base or len(base) > 200 or base.startswith("-"):
        fail("Expected a base branch name")
    for char in base.elems():
        if char not in "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/_-.":
            fail("Invalid base branch name")
    if ".." in base or base.endswith(".") or base.endswith("/") or base.startswith("/"):
        fail("Invalid base branch name")
    for text in [title, body]:
        if type(text) != "string" or not text.strip() or len(text) > 20000:
            fail("Expected nonempty bounded PR text")
    return _publish_gh([
        "pr", "create", "--repo", _PR_REPOSITORY, "--base", base,
        "--head", _PR_BRANCH, "--title", title, "--body", body,
    ])

publication = module(
    "publication", status = publication_status, branch = publication_branch,
    commit = publication_commit, push = publication_push, pr = publication_pr,
)

def propose_agents_star(content):
    """Request user approval before installing a complete configuration."""
    candidate = privileged.tempdir()
    path = candidate.path("candidate.star")
    privileged.write_file(path, content)
    privileged.edit_agents_star(path)

environment = {
    "workspace": workspace, "git": git, "go": go, "repo": repo,
    "propose_agents_star": propose_agents_star, "publication": publication,
}
# This is the existing StarAgent execution environment, not an added REPL tool.
default_repl = repl(environment)
default = privileged.model("gpt-6-astra").create(default_repl, prompt_addons = [
    "Follow AGENTS.md. Use only the exposed named workflows; no general command " +
    "execution is authorized. Do not obtain a shell by modifying a check driver, " +
    "test, compiler source, or sandbox program. Build compiler tools with " +
    "go.build_compiler(); compile reproductions with repo.compile(); execute them " +
    "with repo.execute(). Inspect each process success, stderr and truncation. " +
    "Use scoped go.test(), repo.corpus(), and repo.preflight(). Never weaken " +
    "resource gates, run go test ./..., or test backend/tests as a Go package. " +
    "Use publication tools only for the user-requested driver permissions PR; " +
    "review the complete branch diff and base before publishing. " +
    "Preserve user changes. Additional capabilities require a new user-approved " +
    "configuration; do not bypass these restrictions through existing tools.",
])
