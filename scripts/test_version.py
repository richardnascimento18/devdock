import unittest
from version import compare, parse


class VersionTests(unittest.TestCase):
    def test_prerelease_precedence(self):
        versions = ["0.1.0-dev.1", "0.1.0-dev.2", "0.1.0-dev.10", "0.1.0-rc.1", "0.1.0", "0.2.0", "1.0.0"]
        for left, right in zip(versions, versions[1:]):
            self.assertLess(compare(left, right), 0)
            self.assertGreater(compare(right, left), 0)
        self.assertEqual(compare("0.1.0", "0.1.0"), 0)
        self.assertLess(compare("1.0.0-1", "1.0.0-alpha"), 0)

    def test_invalid_versions(self):
        for value in ["v1.0.0", "01.0.0", "1.0", "1.0.0-01", "1.0.0+build", "1.0.0\n", "1.0.0;cmd", "1.0.0-"]:
            with self.subTest(value=value), self.assertRaises(ValueError):
                parse(value)
