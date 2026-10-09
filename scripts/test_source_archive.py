import pathlib
import subprocess
import tempfile
import unittest
import zipfile


class SourceArchiveTests(unittest.TestCase):
    def test_archive_contains_only_selected_committed_tree(self):
        script = pathlib.Path(__file__).resolve().with_name("package-source.sh")
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            subprocess.run(["git", "init", "-q", str(root)], check=True)
            (root / "scripts").mkdir()
            (root / "scripts/package-source.sh").write_bytes(script.read_bytes())
            (root / "public.txt").write_text("committed")
            subprocess.run(["git", "-C", str(root), "add", "scripts/package-source.sh", "public.txt"], check=True)
            subprocess.run(["git", "-C", str(root), "-c", "user.name=Example", "-c", "user.email=example@example.invalid", "commit", "-qm", "test"], check=True)
            subprocess.run(["git", "-C", str(root), "branch", "local-only"], check=True)
            (root / "public.txt").write_text("dirty")
            (root / "private.txt").write_text("untracked")
            archive = root / "source.zip"
            subprocess.run(["bash", str(root / "scripts/package-source.sh"), str(archive)], check=True, capture_output=True)
            with zipfile.ZipFile(archive) as source:
                self.assertEqual(source.read("devdock/public.txt"), b"committed")
                self.assertEqual(set(source.namelist()), {"devdock/", "devdock/scripts/", "devdock/scripts/package-source.sh", "devdock/public.txt"})
