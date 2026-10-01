import os
from pathlib import Path
import subprocess
import unittest


class ProductionSourceTests(unittest.TestCase):
    def test_promotion_sources(self):
        cases = [
            ("production", "staging", "owner/devdock", True),
            ("production", "feat/foo", "owner/devdock", False),
            ("production", "staging", "fork/devdock", False),
            ("production", "", "owner/devdock", False),
            ("staging", "fix/foo", "fork/devdock", True),
            ("", "", "", True),
        ]
        script = Path(__file__).with_name("check-production-source.sh")
        for base, head, head_repo, allowed in cases:
            with self.subTest(base=base, head=head, head_repo=head_repo):
                env = {"PATH": os.defpath, "BASE_REF": base, "HEAD_REF": head,
                       "HEAD_REPO": head_repo, "REPOSITORY": "owner/devdock"}
                result = subprocess.run(["bash", str(script)], env=env, capture_output=True, text=True)
                self.assertEqual(result.returncode == 0, allowed, result.stderr)
