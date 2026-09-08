"""Keep Required fail-closed across independently triggered platform workflows."""

import json
import os
import subprocess
import time


WORKFLOWS = [f"Performance ({name})" for name in ("Linux", "Windows", "macOS", "Virtual")]


def conclusions(runs):
    # Reruns reuse the run ID; new runs for the same SHA supersede older runs.
    latest = {}
    for run in sorted(runs, key=lambda item: item["databaseId"], reverse=True):
        latest.setdefault(run["workflowName"], run)
    return {
        name: latest[name]["conclusion"] if latest[name]["status"] == "completed" else "pending"
        for name in WORKFLOWS if name in latest
    }


def main():
    deadline = time.monotonic() + 95 * 60
    while time.monotonic() < deadline:
        result = subprocess.run([
            "gh", "run", "list", "--repo", os.environ["GITHUB_REPOSITORY"],
            "--commit", os.environ["RENVO_PERF_SHA"], "--event", os.environ["RENVO_PERF_EVENT"],
            "--limit", "100", "--json", "workflowName,status,conclusion,databaseId",
        ], check=True, capture_output=True, text=True, timeout=60)
        states = conclusions(json.loads(result.stdout))
        print(json.dumps(states), flush=True)
        if any(state not in ("pending", "success") for state in states.values()):
            raise SystemExit("A required performance workflow failed, was cancelled, or was skipped")
        if all(states.get(name) == "success" for name in WORKFLOWS):
            return
        time.sleep(20)
    raise SystemExit("Timed out waiting for all required performance workflows")


if __name__ == "__main__":
    main()
