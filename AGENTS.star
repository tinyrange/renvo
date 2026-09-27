"""Restricted Renvo workflows; AGENTS.md remains authoritative.

  git.status(short=True)
  go.build_compiler()
  repo.compile("sandbox/hello.go", "hello")
  repo.execute("hello")
  go.test("./internal/repl", run="TestSession")
  repo.corpus("backend", "tagged_memory_access")
  repo.preflight()
  publication.status()  # Repository-scoped branch/commit/push/PR helpers available.

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


# Repository-scoped publication; no general command runner or Git writer.
_PR_REPOSITORY = "tinyrange/renvo"

def _publish_git(args):
    return privileged.run(
        cwd = workspace, timeout_ms = 120000, output_limit = 1048576,
        *(["git", "--literal-pathspecs", "-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false"] + args)
    )

def _publish_require(result):
    if not result.success or result.timed_out or result.stdout_truncated or result.stderr_truncated:
        fail("Publication command failed or output was truncated: " + str(result))
    return result.stdout.strip()

def _publish_branch_name(branch):
    branch = _compiler_branch(branch)
    if not branch.startswith("staragent/"):
        fail("Publication branches must be under staragent/")
    return branch

def _publish_on_branch(branch, expected_head):
    branch = _publish_branch_name(branch)
    _compiler_sha(expected_head)
    current = _publish_require(_publish_git(["branch", "--show-current"]))
    head = _publish_require(_publish_git(["rev-parse", "--verify", "HEAD"]))
    if current != branch or head != expected_head:
        fail("Current branch and HEAD must match the reviewed branch and SHA")

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
    """Check GitHub authentication and open repository PRs."""
    results = []
    for args in [
        ["--version"],
        ["auth", "status", "--hostname", "github.com"],
        ["pr", "list", "--repo", _PR_REPOSITORY, "--state", "open", "--limit", "100",
         "--json", "number,url,state,baseRefName,headRefName,headRefOid"],
    ]:
        result = _publish_gh(args)
        results.append(result)
        if not result.success or result.timed_out or result.stdout_truncated or result.stderr_truncated:
            return results
    return results

def publication_branch(branch, expected_head):
    """Create a new staragent/* branch at the reviewed current HEAD.

    Inspect HEAD versus the intended PR base first. Preserves working changes;
    refuses a nonempty index or an existing branch. No reset or rebase.
    """
    branch = _publish_branch_name(branch)
    _compiler_sha(expected_head)
    _publish_clean_index()
    if _publish_require(_publish_git(["rev-parse", "--verify", "HEAD"])) != expected_head:
        fail("HEAD changed since review")
    return _publish_git(["switch", "-c", branch])

def publication_commit(branch, expected_head, paths, message):
    """Stage and commit explicit files on a reviewed staragent/* branch.

    Supports added, modified and deleted files. Refuses a preexisting index,
    directories and ignored new files. The root AGENTS.star may be published;
    editing it still requires the user-approved configuration workflow.
    Review all selected changes first. No amend/reset; unrelated working changes remain untouched. Hooks
    are disabled; signing configuration is retained. On failure, inspect the
    index: staged changes may remain; no automatic rollback is attempted.
    """
    _publish_on_branch(branch, expected_head)
    _publish_clean_index()
    _compiler_text(message)
    if type(paths) not in ["list", "tuple"] or not paths or len(paths) > 500:
        fail("Provide 1 to 500 explicit file paths")
    selected = []
    for path in paths:
        path = _relative(path)
        if any([part.lower() == ".git" for part in path.split("/")]):
            fail("Git metadata cannot be committed by this workflow")
        if path != "AGENTS.star" and any([part.lower() == "agents.star" for part in path.split("/")]):
            fail("Only the root AGENTS.star may be published")
        if path in selected:
            fail("Duplicate commit path")
        selected.append(path)
    listing = _publish_git(["ls-files", "--cached", "--others", "--exclude-standard", "-z", "--"] + selected)
    _publish_require(listing)
    found = listing.stdout.split("\x00")
    found = [path for path in found if path]
    if sorted(found) != sorted(selected):
        fail("Every path must name one tracked or nonignored new file, not a directory")
    _publish_require(_publish_git(["add", "--"] + selected))
    return _publish_git(["commit", "-m", message])

def publication_push(branch, expected_head):
    """Push the reviewed SHA to its staragent/* branch in tinyrange/renvo.

    No force push, tags, deletion, arbitrary remote or refspec. Requires a clean
    index and no tracked working changes. Inspect success before opening a PR.
    """
    _publish_on_branch(branch, expected_head)
    _publish_clean_index()
    dirty = _publish_require(_publish_git(["status", "--porcelain", "--untracked-files=no"]))
    if dirty:
        fail("Commit or resolve tracked changes before publishing: " + dirty)
    return _publish_git([
        "push", "https://github.com/" + _PR_REPOSITORY + ".git",
        expected_head + ":refs/heads/" + branch,
    ])

def publication_pr(branch, base, expected_head, title, body, draft = False):
    """Open a PR for the reviewed, already-pushed staragent/* branch.

    Review the full branch diff and intended base before calling. Verifies
    local and remote heads. No direct merge or settings changes.
    """
    _publish_on_branch(branch, expected_head)
    return compiler_pr_create(branch, base, expected_head, title, body, draft)

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


# Read-only compiler PR inspection, fixed to tinyrange/renvo.
def _compiler_pr_id(value):
    if type(value) != "int" or value <= 0:
        fail("Expected a positive integer PR or Actions run ID")
    return str(value)

def compiler_pr_list():
    """List up to 100 open PRs in tinyrange/renvo with revisions and checks."""
    return _publish_gh([
        "pr", "list", "--repo", _PR_REPOSITORY, "--state", "open", "--limit", "100",
        "--json", "number,url,title,baseRefName,headRefName,headRefOid,isDraft,reviewDecision,mergeStateStatus,statusCheckRollup",
    ])

def compiler_pr_view(number):
    """Inspect PR details, commits, files, reviews, checks and merge state."""
    return _publish_gh([
        "pr", "view", _compiler_pr_id(number), "--repo", _PR_REPOSITORY,
        "--json", "number,url,title,body,state,baseRefName,baseRefOid,headRefName,headRefOid,isDraft,author,commits,files,reviews,reviewDecision,mergeable,mergeStateStatus,statusCheckRollup,autoMergeRequest,mergedAt,mergeCommit",
    ])

def compiler_pr_diff(number):
    """Read a PR diff; inspect process truncation before claiming full review."""
    return _publish_gh([
        "pr", "diff", _compiler_pr_id(number), "--repo", _PR_REPOSITORY, "--color", "never",
    ])

def compiler_pr_checks(number):
    """Read current PR check results without watching or changing them."""
    return _publish_gh([
        "pr", "checks", _compiler_pr_id(number), "--repo", _PR_REPOSITORY,
    ])

def compiler_pr_runs():
    """List up to 100 recent Actions runs, including event and head SHA."""
    return _publish_gh([
        "run", "list", "--repo", _PR_REPOSITORY, "--limit", "100",
        "--json", "databaseId,url,name,workflowName,displayTitle,event,headBranch,headSha,status,conclusion,createdAt,updatedAt",
    ])

def compiler_pr_run_view(run_id, failed_logs = False):
    """Read Actions run/jobs or failed logs; no rerun or cancellation."""
    if type(failed_logs) != "bool":
        fail("failed_logs must be a bool")
    args = ["run", "view", _compiler_pr_id(run_id), "--repo", _PR_REPOSITORY]
    if failed_logs:
        args.append("--log-failed")
    else:
        args.extend([
            "--json", "databaseId,url,name,workflowName,displayTitle,event,headBranch,headSha,status,conclusion,createdAt,updatedAt,jobs",
        ])
    return _publish_gh(args)

compiler_pr = module(
    "compiler_pr", list = compiler_pr_list, view = compiler_pr_view,
    diff = compiler_pr_diff, checks = compiler_pr_checks,
    runs = compiler_pr_runs, run_view = compiler_pr_run_view,
)


# Repository-scoped PR writes, queue-only submission, and Actions reruns.
# No generic API, direct merge, bypass, force push, or settings writer is exported.
def _compiler_sha(value):
    if type(value) != "string" or len(value) != 40:
        fail("Expected a full 40-character reviewed commit SHA")
    for char in value.elems():
        if char not in "0123456789abcdef":
            fail("Expected a lowercase hexadecimal commit SHA")
    return value

def _compiler_text(value):
    if type(value) != "string" or not value.strip() or len(value) > 20000:
        fail("Expected nonempty text of at most 20000 characters")
    return value

def _compiler_branch(value):
    if type(value) != "string" or not value or len(value) > 200:
        fail("Expected a branch name")
    for char in value.elems():
        if char not in "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/_.-":
            fail("Invalid branch name")
    if (value.startswith("-") or value.startswith("/") or value.startswith(".")) or ".." in value:
        fail("Invalid branch name")
    for part in value.split("/"):
        if not part or part.startswith(".") or (part.endswith(".") or part.endswith(".lock")):
            fail("Invalid branch name")
    return value

def _compiler_api(method, endpoint, payload):
    return privileged.run(
        "gh", "api", "--hostname", "github.com", "--method", method,
        endpoint, "--input", "-",
        stdin = json.encode(payload), cwd = workspace,
        env = {"GH_PROMPT_DISABLED": "1", "GH_PAGER": "cat"},
        timeout_ms = 120000, output_limit = 2097152,
    )

def _compiler_json(result):
    if not result.success or result.timed_out or result.stdout_truncated or result.stderr_truncated:
        fail("GitHub operation failed or output was incomplete: " + str(result))
    value = json.decode(result.stdout)
    if type(value) == "dict" and value.get("errors"):
        fail("GitHub GraphQL errors: " + str(value["errors"]))
    return value

def compiler_pr_queue_status(number):
    """Read the fixed repository PR head, queue state, reviews and merge result."""
    _compiler_pr_id(number)
    query = 'query($number:Int!){repository(owner:"tinyrange",name:"renvo"){pullRequest(number:$number){id number url state isDraft headRefOid baseRefName reviewDecision mergeStateStatus isMergeQueueEnabled isInMergeQueue mergeQueueEntry{state position} mergedAt mergeCommit{oid}}}}'
    return _compiler_api("POST", "graphql", {"query": query, "variables": {"number": number}})

def _compiler_reviewed_pr(number, expected_head):
    _compiler_sha(expected_head)
    value = _compiler_json(compiler_pr_queue_status(number))
    pr = value["data"]["repository"]["pullRequest"]
    if not pr or pr["state"] != "OPEN" or pr["headRefOid"] != expected_head:
        fail("PR must be open and still match the reviewed head")
    return pr

def compiler_pr_create(head, base, expected_head, title, body, draft = False):
    """Create a PR from an already-pushed branch in tinyrange/renvo; never pushes."""
    head = _compiler_branch(head)
    base = _compiler_branch(base)
    _compiler_sha(expected_head)
    _compiler_text(title)
    _compiler_text(body)
    if type(draft) != "bool" or head == base:
        fail("Expected boolean draft and distinct head/base branches")
    ref = _compiler_json(_publish_gh([
        "api", "--hostname", "github.com", "repos/" + _PR_REPOSITORY + "/git/ref/heads/" + head,
    ]))
    if ref["object"]["sha"] != expected_head:
        fail("Remote branch no longer matches the reviewed head")
    return _compiler_api("POST", "repos/" + _PR_REPOSITORY + "/pulls", {
        "head": head, "base": base, "title": title, "body": body, "draft": draft,
    })

def compiler_pr_edit(number, expected_head, title, body):
    """Edit only PR title/body after checking its open state and reviewed head."""
    _compiler_text(title)
    _compiler_text(body)
    _compiler_reviewed_pr(number, expected_head)
    return _compiler_api("PATCH", "repos/" + _PR_REPOSITORY + "/pulls/" + str(number), {
        "title": title, "body": body,
    })

def compiler_pr_comment(number, expected_head, body):
    """Post a PR progress comment after checking the reviewed head."""
    _compiler_text(body)
    _compiler_reviewed_pr(number, expected_head)
    return _compiler_api("POST", "repos/" + _PR_REPOSITORY + "/issues/" + str(number) + "/comments", {"body": body})

def compiler_pr_enqueue(number, expected_head):
    """Enqueue only the reviewed head; no direct merge, queue jump or bypass.

    Review the complete diff and checks first. Monitor merge-group Actions and
    final MERGED state afterwards; successful enqueue is not a completed merge.
    """
    pr = _compiler_reviewed_pr(number, expected_head)
    if pr["isDraft"] or not pr["isMergeQueueEnabled"]:
        fail("Requires a non-draft PR targeting an enabled merge queue")
    if pr["reviewDecision"] in ["CHANGES_REQUESTED", "REVIEW_REQUIRED"]:
        fail("Required review prerequisites are not satisfied")
    if pr["isInMergeQueue"]:
        return pr
    checks = _publish_gh(["pr", "checks", str(number), "--repo", _PR_REPOSITORY, "--required"])
    if not checks.success or checks.timed_out or checks.stdout_truncated or checks.stderr_truncated:
        fail("Required checks must pass before enqueue: " + str(checks))
    query = "mutation($input:EnqueuePullRequestInput!){enqueuePullRequest(input:$input){mergeQueueEntry{state position}}}"
    result = _compiler_api("POST", "graphql", {
        "query": query, "variables": {"input": {
            "pullRequestId": pr["id"], "expectedHeadOid": expected_head, "jump": False,
        }},
    })
    return _compiler_json(result)

def compiler_pr_rerun(run_id, expected_head, failed_only = True):
    """Rerun a completed PR or merge-group Actions run at the inspected SHA.

    Defaults to failed jobs only. Diagnose failures first; do not repeatedly
    rerun deterministic failures in place of fixing them.
    """
    _compiler_pr_id(run_id)
    _compiler_sha(expected_head)
    if type(failed_only) != "bool":
        fail("failed_only must be a bool")
    run = _compiler_json(compiler_pr_run_view(run_id))
    if run["headSha"] != expected_head or run["status"] != "completed":
        fail("Run must be completed and match the inspected SHA")
    if run["event"] not in ["pull_request", "merge_group"]:
        fail("Only PR and merge-group runs may be rerun")
    args = ["run", "rerun", str(run_id), "--repo", _PR_REPOSITORY]
    if failed_only:
        args.append("--failed")
    return _publish_gh(args)

compiler_pr = compiler_pr + module(
    "compiler_pr", create = compiler_pr_create, edit = compiler_pr_edit,
    comment = compiler_pr_comment, queue_status = compiler_pr_queue_status,
    enqueue = compiler_pr_enqueue, rerun = compiler_pr_rerun,
)

environment = {
    "workspace": workspace, "git": git, "go": go, "repo": repo,
    "propose_agents_star": propose_agents_star, "publication": publication,
    "compiler_pr": compiler_pr,
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
    "Use publication tools only for user-requested changes in tinyrange/renvo; " +
    "review the complete branch diff and base before publishing. " +
    "Preserve user changes. Additional capabilities require a new user-approved " +
    "configuration; do not bypass these restrictions through existing tools.",
])
