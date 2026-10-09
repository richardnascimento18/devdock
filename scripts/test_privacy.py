import unittest
from privacy import findings


class PrivacyTests(unittest.TestCase):
    def test_private_values_report_categories_only(self):
        private = b"/home/" + b"personal/work\n/run/" + b"media/private/disk\n"
        self.assertEqual(findings(private), [(1, "personal home path"), (2, "mounted machine path")])

    def test_generic_examples(self):
        self.assertEqual(findings(b"/home/user/projects\n/tmp/example\nhttps://github.com/owner/repo"), [])
