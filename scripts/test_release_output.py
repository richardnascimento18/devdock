import pathlib
import shutil
import subprocess
import tempfile
import unittest
import zipfile

from release_output import clean


class ReleaseOutputTests(unittest.TestCase):
    def test_stale_outputs_are_removed(self):
        with tempfile.TemporaryDirectory() as folder:
            output = pathlib.Path(folder) / "dist"
            output.mkdir()
            for name in ["devdock_old_linux_amd64", "devdock_old_linux_arm64", "SHA256SUMS"]:
                (output / name).write_text("stale")
            clean(output)
            self.assertEqual(list(output.iterdir()), [])
            clean(output)  # Idempotent empty preparation.

    def test_unrelated_files_are_preserved_and_refused(self):
        with tempfile.TemporaryDirectory() as folder:
            output = pathlib.Path(folder)
            unrelated = output / "personal.txt"
            unrelated.write_text("keep")
            stale = output / "devdock_old_linux_amd64"
            stale.write_text("stale")
            with self.assertRaisesRegex(ValueError, "unexpected release output"):
                clean(output)
            self.assertEqual(unrelated.read_text(), "keep")
            self.assertTrue(stale.exists())

    def test_source_archive_excludes_dirty_and_untracked_outputs(self):
        with tempfile.TemporaryDirectory() as folder:
            repo = pathlib.Path(folder) / "repo"
            repo.mkdir()
            def git(*args):
                return subprocess.run(["git", *args], cwd=repo, check=True, capture_output=True)
            git("init")
            git("config", "user.name", "Archive test")
            git("config", "user.email", "archive@example.invalid")
            tracked = repo / "tracked.txt"
            tracked.write_text("committed")
            git("add", "tracked.txt")
            git("commit", "-m", "test: archive fixture")
            tracked.write_text("dirty")
            (repo / "dist").mkdir()
            (repo / "dist" / "stale.bin").write_text("stale")
            (repo / "scripts").mkdir()
            helper = repo / "scripts" / "package-source.sh"
            shutil.copyfile(pathlib.Path(__file__).with_name("package-source.sh"), helper)
            archive = pathlib.Path(folder) / "source.zip"
            subprocess.run(["bash", str(helper), str(archive)], check=True, capture_output=True)
            with zipfile.ZipFile(archive) as output:
                self.assertEqual(output.namelist(), ["devdock/", "devdock/tracked.txt"])
                self.assertEqual(output.read("devdock/tracked.txt"), b"committed")
