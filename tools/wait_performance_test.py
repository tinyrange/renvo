import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location("wait_performance", Path(__file__).with_name("wait-performance.py"))
wait = importlib.util.module_from_spec(spec)
spec.loader.exec_module(wait)


class RequiredPerformanceTests(unittest.TestCase):
    def test_missing_workflows_do_not_pass(self):
        self.assertFalse(all(wait.conclusions([]).get(name) == "success" for name in wait.WORKFLOWS))

    def test_newer_failure_supersedes_success(self):
        runs = [
            dict(databaseId=1, workflowName=wait.WORKFLOWS[0], status="completed", conclusion="success"),
            dict(databaseId=2, workflowName=wait.WORKFLOWS[0], status="completed", conclusion="failure"),
        ]
        self.assertEqual(wait.conclusions(runs)[wait.WORKFLOWS[0]], "failure")

    def test_rerun_in_progress_does_not_reuse_old_conclusion(self):
        run = dict(databaseId=2, workflowName=wait.WORKFLOWS[0], status="in_progress", conclusion="success")
        self.assertEqual(wait.conclusions([run])[wait.WORKFLOWS[0]], "pending")

    def test_every_platform_must_finish(self):
        runs = [dict(databaseId=i, workflowName=name, status="completed", conclusion="success")
                for i, name in enumerate(wait.WORKFLOWS)]
        self.assertTrue(all(wait.conclusions(runs).get(name) == "success" for name in wait.WORKFLOWS))
        runs[-1]["status"] = "in_progress"
        self.assertFalse(all(wait.conclusions(runs).get(name) == "success" for name in wait.WORKFLOWS))
